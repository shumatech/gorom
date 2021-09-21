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
    "runtime"
    "path"

    "gorom/dat"
    "gorom/romdb"
    "gorom/romio"
    "gorom/util"
    "gorom/term"
)

const (
    TrashDir = ".trash"
)

type FixromStats struct {
    Ok      int
    Fixed   int
    Failed  int
    Extra   int
    Total   int
}

type Rename struct {
    machPath string
    tmpPath string
}

func printHeader(header *dat.Header) error {
    if !options.App.NoHeader {
        term.Println(header.Name)
    }
    return nil
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

func copyRoms(machPath string, roms []CopyRom, ch chan CopyResults) {
    results := CopyResults{ machPath: machPath }

    machExt := romio.MachExt(machPath)
    writer, err := romio.CreateRomWriterTemp(".", machExt)
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

func copyProcess(renameList *[]Rename, ch chan CopyResults) {
    results := <-ch
    util.Progressf(results.machPath)
    if results.err != nil {
        util.Progressf("")
        term.Printf(term.Red("%s : %s : %s\n", results.machPath, results.errmsg, results.err))
        if results.tmpPath != "" {
            err := os.RemoveAll(results.tmpPath)
            if err != nil {
                util.Progressf("")
                term.Println(term.Red("trash %s: %s", results.tmpPath, err))
            }
        }
    } else {
        *renameList = append(*renameList, Rename{ results.machPath, results.tmpPath })
    }
}

func fixrom(datFile string, machines []string, dirs []string) (bool, error) {
    var renameList []Rename
    var FixromStats FixromStats

    var machSet util.StringSet
    if options.FixRom.ExtraTrash {
        machSet = util.NewStringSet()
    }

    // Current directory takes precedence
    dirs = append([]string{"."}, dirs...)

    // Scan all of the provided directories
    romDBs := []*romdb.RomDB{}
    for _, dir := range dirs {
        term.Printf("Scanning directory %s\n", dir)

        rdb, err := romdb.OpenRomDB(dir, options.App.SkipHeader)
        if err != nil {
            return false, err
        }
        defer rdb.Close()
        if !options.FixRom.SkipScan {
            goLimit := 0
            if options.App.NoGo {
                goLimit = 1
            }
            err = rdb.Scan(goLimit, nil, func(machPath string, err error) {
                if err != nil {
                    util.Progressf(term.Red("%s: %s\n", machPath, err))
                } else {
                    util.Progressf("%s", machPath)
                }
            })
            if err != nil {
                return false, err
            }
        }
        romDBs = append(romDBs, rdb)
        util.Progressf("");
    }

    goCount := 0
    goLimit := 1
    if !options.App.NoGo {
        goLimit = runtime.NumCPU()
    }

    ch := make(chan CopyResults, 1)

    err := dat.ParseDatFile(datFile, machines, printHeader, func(machine *dat.Machine) error {
        badNames := map[string]string{}
        extras := []string{}

        FixromStats.Total++

        if options.FixRom.ExtraTrash {
            machSet.Set(machine.Name)
        }

        // Validate all the ROM checksums in the machine
        valid, err := dat.ValidateChecksums(machine, romDBs[0], badNames, &extras, nil)
        if err != nil {
            term.Printf(term.Red("%s: %s\n", machine.Name, err))
        }
        util.Progressf("")

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
                FixromStats.Ok++
                if !options.App.NoOk {
                    term.Printf("%s : %s\n", machine.Path, term.Green("OK"))
                }
                return nil
            }
        }

        // Set the machine path if the machine is not valid
        if !valid {
            machine.Path = machine.Name + options.FixRom.DefaultFmt
        }

        term.Printf("%s : %s\n", machine.Path, term.Cyan("FIXING"))

        roms := []CopyRom{}

        // Copy OK and bad name ROMs from the old machine if it is valid
        if valid {
            for _, rom := range machine.Roms {
                if rom.Status == dat.RomOk {
                    if !options.App.NoOk {
                        term.Printf("  %s : %s\n", rom.Name, term.Green("OK"))
                    }
                    roms = append(roms, CopyRom{ dstName: rom.Name, srcName: rom.Name, srcPath: machine.Path })
                } else  if rom.Status == dat.RomBadName {
                    term.Printf("  %s : %s\n", rom.Name, term.Magenta("RENAME from %s", badNames[rom.Name]))
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
                        return err
                    }
                    if entry != nil {
                        break
                    }
                }

                // Stop the fix if not found
                if entry == nil {
                    term.Printf("  %s : %s\n", rom.Name, term.Red("NOT FOUND"))
                    ok = false
                    break
                }

                // Copy the found ROM to the new machine
                path := path.Join(rdb.Dir, entry.MachPath)
                term.Printf("  %s : %s\n", rom.Name, term.Cyan("COPY from %s", path))
                roms = append(roms, CopyRom{ dstName: rom.Name, srcName: entry.RomPath, srcPath: path })
            }
        }

        // Start the copy job if everything was OK
        if ok {
            if goCount == goLimit {
                copyProcess(&renameList, ch)
            } else {
                goCount++
            }

            FixromStats.Fixed++
            term.Printf("  %s\n", term.Green("OK"))
            go copyRoms(machine.Path, roms, ch)
        } else {
            FixromStats.Failed++
            term.Printf("  %s\n", term.Red("FAILED"))
        }

        return nil
    })
    if err != nil {
        return false, err
    }

    // Process the copy results
    if goCount > 0 {
        term.Println("Waiting for copy jobs to complete")
        for ; goCount > 0; goCount-- {
            copyProcess(&renameList, ch)
        }
        util.Progressf("")
    }

    // Rename all of the temp files and move old files to trash
    if len(renameList) > 0 {
        term.Println("Renaming temporary files")
        err = os.Mkdir(".trash", 0755)
        if err != nil && os.IsNotExist(err) {
            return false, err
        }

        for _, r := range renameList {
            util.Progressf(r.machPath)
            _, err = os.Stat(r.machPath)
            if err == nil || os.IsExist(err) {
                err = os.Rename(r.machPath, path.Join(".trash", r.machPath))
                if err != nil {
                    util.Progressf("")
                    term.Println(term.Red("trash %s: %s", r.machPath, err))
                    continue
                }
            }

            err = os.Rename(r.tmpPath, r.machPath)
            if err != nil {
                util.Progressf("")
                term.Println(term.Red("rename %s to %s: %s", r.tmpPath, r.machPath, err))
            }
        }
        util.Progressf("")
    }

    // Delete extra files if option given
    if options.FixRom.ExtraTrash && len(machines) == 0 {
        err = util.ScanDir(".", true, func(file os.FileInfo) error {
            machPath := file.Name()
            machName := romio.MachName(machPath)
            if !machSet.IsSet(machName) {
                FixromStats.Extra++
                term.Printf("%s : %s\n", machPath, term.Blue("EXTRA"))
                err = os.Rename(machPath, path.Join(TrashDir, machPath))
                if err != nil {
                    term.Println(term.Red("trash %s: %s", machPath, err))
                }
            }
            return nil
        })
        if err != nil {
            return false, err
        }
    }

    term.Println("\nMachine Stats")
    term.Printf("  OK     : %d (%.1f%%)\n", FixromStats.Ok, 100.0 * float32(FixromStats.Ok) / float32(FixromStats.Total))
    term.Printf("  Fixed  : %d (%.1f%%)\n", FixromStats.Fixed, 100.0 * float32(FixromStats.Fixed) / float32(FixromStats.Total))
    term.Printf("  Failed : %d (%.1f%%)\n", FixromStats.Failed, 100.0 * float32(FixromStats.Failed) / float32(FixromStats.Total))
    term.Printf("  Total  : %d\n", FixromStats.Total)
    if options.FixRom.ExtraTrash {
        term.Printf("  Extra  : %d\n", FixromStats.Extra)
    }

    return (FixromStats.Failed == 0), nil
}

