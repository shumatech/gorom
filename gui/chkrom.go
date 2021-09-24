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
    "runtime"
    "os"

    "gorom/dat"
    "gorom/util"
    "gorom/checksum"
    "gorom/romdb"
    "gorom/romio"
)

type ChkromStats struct {
    Ok      int
    Missing int
    Corrupt int
    BadName int
    Extra   int
    Total   int
}

type ValidResults struct {
    machine *dat.Machine
    badNames map[string]string
    extras []string
    ok bool
    err error
}

const (
    MachUnknown = 1 << iota
    MachOk
    MachMissing
    MachCorrupt

    MachRomCorrupt
    MachRomBadName
    MachRomMissing
    MachRomExtra
)

var (
    chkRomStats ChkromStats
    chkMachStats ChkromStats
    chkMachRomStats ChkromStats
)

func validateChecksums(machine *dat.Machine, rdb *romdb.RomDB, ch chan ValidResults) {
    if isStop() {
        ch <- ValidResults{ machine: machine, ok: false, err: stopError}
        return
    }

    badNames := map[string]string{}
    extras := []string{}

    ok, err := dat.ValidateChecksums(machine, rdb, badNames, &extras, func(name string, sum checksum.Sha1) error {
        if isStop() {
            return stopError
        }
        return nil
    })

    ch <- ValidResults{ machine: machine, badNames: badNames, extras: extras, ok: ok, err: err}
}

func chkromResults(count *int, machMap map[string]int, ch chan ValidResults) {
    results := <- ch

    machine := results.machine
    extras := results.extras
    ok := results.ok
    err := results.err

    chkMachStats.Total++
    var machStatus int

    if err == stopError {
        chkRomStats.Total += len(machine.Roms)
        machStatus = MachUnknown
        for _, rom := range machine.Roms {
            rom.Status = dat.RomUnknown
        }
    } else if err != nil {
        chkRomStats.Total += len(machine.Roms)
        chkRomStats.Corrupt += len(machine.Roms)
        chkMachStats.Corrupt++
        machStatus = MachCorrupt
        for _, rom := range machine.Roms {
            rom.Status = dat.RomCorrupt
        }
    } else if ok {
        machStatus = MachOk
        for _, rom := range machine.Roms {
            switch rom.Status {
            case dat.RomCorrupt:
                machStatus |= MachRomCorrupt
            case dat.RomBadName:
                machStatus |= MachRomBadName
            case dat.RomMissing:
                machStatus |= MachRomMissing
            }
        }

        if len(extras) > 0 {
            chkMachRomStats.Extra++
            machStatus |= MachRomExtra
        }

        if machStatus == MachOk {
            chkMachStats.Ok++
        } else if machStatus != MachRomExtra {
            if machStatus & MachRomCorrupt != 0 {
                chkMachRomStats.Corrupt++
            }
            if machStatus & MachRomBadName != 0 {
                chkMachRomStats.BadName++
            }
            if machStatus & MachRomMissing != 0 {
                chkMachRomStats.Missing++
            }
        }

        for _, rom := range machine.Roms {
            chkRomStats.Total++

            switch rom.Status {
            case dat.RomOk:
                chkRomStats.Ok++
            case dat.RomCorrupt:
                chkRomStats.Corrupt++
            case dat.RomBadName:
                chkRomStats.BadName++
            case dat.RomMissing:
                chkRomStats.Missing++
            }
        }

        chkRomStats.Extra += len(extras)
    } else {
        chkRomStats.Total += len(machine.Roms)
        chkRomStats.Missing += len(machine.Roms)
        chkMachStats.Missing++
        machStatus = MachMissing
        for _, rom := range machine.Roms {
            rom.Status = dat.RomMissing
        }
    }

    if machStatus != MachUnknown {
        var str string
        switch machStatus {
            case MachOk: str = "OK"
            case MachMissing: str = "MISSING"
            case MachCorrupt: str = "CORRUPT"
            default: str = "ERRORS"
        }
        *count++
        call("status", float32(*count) / float32(len(machines)), machMap[machine.Name], str)
    }
}

func chkromStart(dir string, options Options) error {
    chkRomStats = ChkromStats{}
    chkMachStats = ChkromStats{}
    chkMachRomStats = ChkromStats{}

    var rdb *romdb.RomDB
    var err error

    wd, _ := os.Getwd()
    err = os.Chdir(dir)
    if err != nil {
        return err
    }
    defer os.Chdir(wd)

    rdb, err = romdb.OpenRomDB(".", options.headers)
    if err != nil {
        return err
    }
    defer rdb.Close()

    ch := make(chan ValidResults)

    goCount := 0
    goLimit := runtime.NumCPU()
    machMap := map[string]int{}
    resultCount := 0
    for i, machine := range machines {
        machMap[machine.Name] = i
        if goCount == goLimit {
            chkromResults(&resultCount, machMap, ch)
        } else {
            goCount++
        }

        go validateChecksums(machine, rdb, ch)

        if isStop() {
            return stopError;
        }
    }

    for ; goCount > 0; goCount-- {
        chkromResults(&resultCount, machMap, ch)
    }

    err = util.ScanDir(".", true, func(file os.FileInfo) error {
        if isStop() {
            return stopError
        }
        path := file.Name()
        name := romio.MachName(path)
        _, ok := machMap[name]
        if !ok {
            call("log", name + " : EXTRA")
            chkMachStats.Extra++
        }
        return nil
    })
    if err != nil {
        return err
    }

    return nil
}
