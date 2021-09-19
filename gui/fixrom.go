//  GoRom - Emulator ROM Management Utilities
//  Copyright (C) 2020 Scott Shumate <scott@shumatech.com>
//
//  This program is free software: you can redistribute it and/or modify
//  it under the terms of the GNU General Public License as published by
//  the Free Software Foundation, either version 3 of the License, or
//  (at your option) any later version.
//
//  This program is distributed in the hope that it will be useful,
//  but WITHOUT ANY WARRANTY; without even the implied warranty of
//  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
//  GNU General Public License for more details.
//
//  You should have received a copy of the GNU General Public License
//  along with this program.  If not, see <http://www.gnu.org/licenses/>.
package main

import (
    "fmt"
    "os"
    "path"
    "runtime"

    "gorom/dat"
    "gorom/romdb"
    "gorom/romio"
    "gorom/checksum"
)

type FixMachStats struct {
    Ok       int
    Fixed    int
    Failed   int
    Total    int
}

type FixRomStats struct {
    Ok       int
    Copied   int
    Renamed  int
    NotFound int
    Total    int
}

type Rename struct {
    machPath string
    tmpPath string
}

type CopyRom struct {
    dstName string
    srcName string
    srcPath string
}

type CopyResults struct {
    machPath string
    tmpPath string
    errmsg string
    err error
}

const (
    RomFixCopy = iota + 100
    RomFixNotFound
    RomFixRename
)

var (
    fixMachStats FixMachStats
    fixRomStats FixRomStats
)
func copyRoms(machPath string, isDir bool, roms []CopyRom, ch chan CopyResults) {
    results := CopyResults{ machPath: machPath }

    if isStop() {
        results.err = stopError
        ch <- results
        return
    }

    writer, err := romio.CreateRomWriterTemp(".", isDir)
    if err != nil {
        results.errmsg = "create temp writer"
    } else {
        results.tmpPath = writer.Path()

        for _, rom := range roms {
            err = writer.Create(rom.dstName)
            if err != nil {
                results.errmsg = "writer create"
            }
        }

        for index := writer.First(); index >= 0; index = writer.Next() {
            rom := roms[index]
            if isStop() {
                results.errmsg = rom.srcPath
                err = stopError
                break
            }
            reader, err := romio.OpenRomReader(rom.srcPath)
            if err != nil {
                results.errmsg = rom.srcPath
                break
            }
            if reader == nil {
                results.errmsg = rom.srcPath
                err = fmt.Errorf("unable to open reader")
                break
            }
            err = romio.CopyRom(writer, rom.dstName, reader, rom.srcName)
            reader.Close()
            if err != nil {
                results.errmsg = fmt.Sprintf("copy %s to %s", rom.srcName, writer.Path())
                break
            }
        }
    }

    if writer != nil {
        writer.Close()
    }

    results.err = err

    ch <-results
}

func copyProcess(renameList *[]Rename, opCount *int, opTotal *int, ch chan CopyResults) {
    results := <-ch
    if results.err != nil {
        call("log", results.machPath + " : " + results.errmsg + " : " + results.err.Error())
        if results.tmpPath != "" {
            err := os.RemoveAll(results.tmpPath)
            if err != nil {
                call("log", results.tmpPath + " : " + err.Error())
            }
        }
    } else {
        *renameList = append(*renameList, Rename{ results.machPath, results.tmpPath })
        *opTotal++
    }

    *opCount++
    call("status", float32(*opCount) / float32(*opTotal), -1, 0)
}

func fixromStart(dir string, srcDirs []string, defaultDir bool) error {
    fixMachStats = FixMachStats{}
    fixRomStats = FixRomStats{}

    var renameList []Rename

    wd, _ := os.Getwd()
    err := os.Chdir(dir)
    if err != nil {
        return err
    }
    defer os.Chdir(wd)

    // Current directory takes precedence
    srcDirs = append([]string{"."}, srcDirs...)

    opCount := 0
    opTotal := len(machines)

    // Count the files in each source directory
    for _, dir := range srcDirs {
        fh, err := os.Open(dir)
        if err != nil {
            return err
        }

        flist, err := fh.Readdir(0)
        fh.Close()
        if err != nil {
            return err
        }

        for _, file := range flist {
            if (file.Name()[0] != '.') {
                opTotal++
            }
        }
    }

    // Scan all of the provided directories
    romDBs := []*romdb.RomDB{}
    for _, dir := range srcDirs {
        call("log", "Scanning " + dir)

        rdb, err := romdb.OpenRomDB(dir, false) // TODO: add an option to skip headers?
        if err != nil {
            return err
        }
        defer rdb.Close()
        err = rdb.Scan(0, stopChan, func(machPath string, err error) {
            if err != nil {
                call("log", machPath + " : " + err.Error())
            }
            opCount++
            call("status", float32(opCount) / float32(opTotal), -1, 0)
        })
        if err != nil {
            return err
        }
        if isStop() {
            return stopError
        }
        romDBs = append(romDBs, rdb)
    }

    goCount := 0
    goLimit := runtime.NumCPU()

    ch := make(chan CopyResults)

    for i, machine := range machines {
        if isStop() {
            break
        }

        badNames := map[string]string{}
        extras := []string{}

        fixMachStats.Total++

        // Validate all the ROM checksums in the machine
        valid, err := dat.ValidateChecksums(machine, romDBs[0], badNames, &extras, func(name string, sum checksum.Sha1) error {
            if isStop() {
                return stopError
            }
            return nil
        })
        if err == stopError {
            break
        }
        if err != nil {
            call("log", machine.Name + " : " + err.Error())
        }

        // Machine is OK if there are no extras and all ROMS are OK
        if valid && len(extras) == 0 {
            ok := true
            for _, rom := range machine.Roms {
                if rom.Status != dat.RomOk {
                    ok = false
                    break
                }
            }
            if ok {
                fixMachStats.Ok++
                fixRomStats.Ok += len(machine.Roms)
                fixRomStats.Total += len(machine.Roms)
                opCount++
                call("status", float32(opCount) / float32(opTotal), i, "OK")
                continue
            }
        }

        // Determine the the new machine is a dir or zip
        machIsDir := (valid && machine.IsDir) || (!valid && defaultDir)

        // Set the machine path if the machine is not valid
        if !valid {
            machine.Path = machine.Name
            if !machIsDir {
                machine.Path += ".zip"
            }
        }

        roms := []CopyRom{}

        // Copy OK and bad name ROMs from the old machine if it is valid
        if valid {
            for _, rom := range machine.Roms {
                if rom.Status == dat.RomOk {
                    fixRomStats.Ok++
                    fixRomStats.Total++
                    roms = append(roms, CopyRom{ dstName: rom.Name, srcName: rom.Name, srcPath: machine.Path })
                } else  if rom.Status == dat.RomBadName {
                    rom.Status = RomFixRename
                    fixRomStats.Renamed++
                    fixRomStats.Total++
                    roms = append(roms, CopyRom{ dstName: rom.Name, srcName: badNames[rom.Name], srcPath: machine.Path })
                }
            }
        }

        // Find corrupt/missing ROMS in sources and copy to new machine
        ok := true
        for _, rom := range machine.Roms {
            if rom.Status == dat.RomUnknown || rom.Status == dat.RomCorrupt || rom.Status == dat.RomMissing {
                var entry *romdb.RomDBEntry
                var rdb *romdb.RomDB

                // Walk the sources to find the checksum 
                for _, rdb = range romDBs {
                    entry, err = rdb.Lookup(rom.Sha1)
                    if err != nil {
                        call("log", "checksum lookup : " + err.Error())
                        break
                    }
                    if entry != nil {
                        break
                    }
                }

                if entry == nil {
                    rom.Status = RomFixNotFound
                    fixRomStats.NotFound++
                    fixRomStats.Total++
                    ok = false
                } else {
                    // Copy the found ROM to the new machine
                    path := path.Join(rdb.Dir, entry.MachPath)
                    rom.Status = RomFixCopy
                    fixRomStats.Copied++
                    fixRomStats.Total++
                    roms = append(roms, CopyRom{ dstName: rom.Name, srcName: entry.RomPath, srcPath: path })
                }
            }
        }

        // Start the copy job if everything was OK
        if ok {
            if goCount == goLimit {
                copyProcess(&renameList, &opCount, &opTotal, ch)
            } else {
                goCount++
            }

            fixMachStats.Fixed++
            opTotal++ // Add one for copy job
            opCount++
            call("status", float32(opCount) / float32(opTotal), i, "FIXED")
            go copyRoms(machine.Path, machIsDir, roms, ch)
        } else {
            fixMachStats.Failed++
            opCount++
            call("status", float32(opCount) / float32(opTotal), i, "FAILED")
        }
    }

    // Process the copy results
    if goCount > 0 {
        call("log", "Waiting for copy jobs to complete")
        for ; goCount > 0; goCount-- {
            copyProcess(&renameList, &opCount, &opTotal, ch)
        }
    }

    // If stopped, then delete all of the temporary files to rename
    if isStop() {
        for _, r := range renameList {
            err := os.RemoveAll(r.tmpPath)
            if err != nil {
                call("log", r.tmpPath + " : " + err.Error())
            }
        }
        return stopError
    }

    // Rename all of the temp files and move old files to trash
    if len(renameList) > 0 {
        call("log", "Renaming temporary files")
        err := os.Mkdir(".trash", 0755)
        if err != nil && os.IsNotExist(err) {
            return  err
        }

        for _, r := range renameList {
            opCount++
            _, err = os.Stat(r.machPath)
            if err == nil || os.IsExist(err) {
                err = os.Rename(r.machPath, path.Join(".trash", r.machPath))
                if err != nil {
                    call("log", r.machPath + " : " + err.Error())
                    continue
                }
            }

            err = os.Rename(r.tmpPath, r.machPath)
            if err != nil {
                call("log", "rename " + r.tmpPath + " to " + r.machPath + " : " + err.Error())
            }
        }

        call("status", float32(opCount) / float32(opTotal), -1, 0)
    }

    return nil
}