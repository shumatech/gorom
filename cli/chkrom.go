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
    "encoding/json"
    "fmt"
    "os"
    "runtime"

    "gorom"
    "gorom/term"
)

type ChkromStats struct {
    Ok      int
    Missing int
    Corrupt int
    BadName int
    Extra   int
    Total   int
}

const (   
    MachOk = 1 << iota
    MachMissing
    MachCorrupt

    MachRomCorrupt
    MachRomBadName
    MachRomMissing
    MachRomExtra
)

type Logger interface {
    header(name string)
    machine(machine *gorom.Machine, status int, info ...string)
    extra(path string)
    rom(rom *gorom.Rom, status int, info ...string)
    machineChkromStats(machChkromStats *ChkromStats, machRomChkromStats *ChkromStats)
    romChkromStats(romChkromStats *ChkromStats)
    close()
}

///////////////////////////////////////////////////////////////////////////////
// Stdout Logger
///////////////////////////////////////////////////////////////////////////////

type StdLogger struct {
}

func NewStdLogger() *StdLogger {
    return &StdLogger{}
}

func (log *StdLogger) header(name string) {
    term.Println(name)
}

func (log *StdLogger) machine(machine *gorom.Machine, status int, info ...string) {
    var str string
    switch {
    case status == MachOk:
        str = term.Green("OK")
    case status == MachMissing:
        str = term.Yellow("MISSING")
    case status == MachCorrupt:
        str = term.Red("CORRUPT")            
        if len(info) > 0 {
            str += term.Red(fmt.Sprintf(" (%s)", info[0]))
        }
    case (status & MachRomCorrupt != 0) || (status & MachRomBadName != 0) || (status & MachRomMissing != 0):
        str = term.Red("ROM ERRORS")
    case (status & MachRomExtra != 0):
        str = term.Blue("EXTRA ROMS")
    default:
        term.Println(status)
        panic("invalid machine status")
    }

    name := machine.Name
    if machine.Path != "" {
        name = machine.Path
    }
    term.Printf("%s : %s\n", name, str)
}

func (log *StdLogger) extra(path string) {
    term.Printf("%s : %s\n", path, term.Blue("EXTRA"))
}

func (log *StdLogger) rom(rom *gorom.Rom, status int, info ...string) {
    var str string
    switch status {
    case gorom.RomOk:
        str = term.Green("OK")
    case gorom.RomMissing:
        str = term.Yellow("MISSING")
    case gorom.RomUnknown:
        str = term.Blue("EXTRA")
    case gorom.RomCorrupt:
        str = term.Red("CORRUPT")
    case gorom.RomBadName:
        str = term.Magenta(fmt.Sprintf("BAD NAME (%s)", info[0]))
    default:
        panic("invalid rom status")
    }

    term.Printf("  %s : %s\n", rom.Name, str)
}

func (log *StdLogger) machineChkromStats(machChkromStats *ChkromStats, machRomChkromStats *ChkromStats) {
    term.Println("\nMachine Stats")
    term.Printf("  All OK          : %d (%.1f%%)\n", machChkromStats.Ok, 100.0 * float32(machChkromStats.Ok) / float32(machChkromStats.Total))
    term.Printf("  ROMs Corrupt    : %d (%.1f%%)\n", machRomChkromStats.Corrupt, 100.0 * float32(machRomChkromStats.Corrupt) / float32(machChkromStats.Total))
    term.Printf("  ROMs Bad Name   : %d (%.1f%%)\n", machRomChkromStats.BadName, 100.0 * float32(machRomChkromStats.BadName) / float32(machChkromStats.Total))
    term.Printf("  ROMs Missing    : %d (%.1f%%)\n", machRomChkromStats.Missing, 100.0 * float32(machRomChkromStats.Missing) / float32(machChkromStats.Total))
    term.Printf("  ROMs Extra      : %d (%.1f%%)\n", machRomChkromStats.Extra, 100.0 * float32(machRomChkromStats.Extra) / float32(machChkromStats.Total))
    term.Printf("  Machine Missing : %d (%.1f%%)\n", machChkromStats.Missing, 100.0 * float32(machChkromStats.Missing) / float32(machChkromStats.Total))
    term.Printf("  Machine Corrupt : %d (%.1f%%)\n", machChkromStats.Corrupt, 100.0 * float32(machChkromStats.Corrupt) / float32(machChkromStats.Total))
    term.Printf("  Total Machines  : %d\n", machChkromStats.Total)
    term.Printf("  Extra Files     : %d\n", machChkromStats.Extra)
}

func (log *StdLogger) romChkromStats(romChkromStats *ChkromStats) {
    term.Println("\nROM Stats")
    term.Printf("  OK        : %d (%.1f%%)\n", romChkromStats.Ok, 100.0 * float32(romChkromStats.Ok) / float32(romChkromStats.Total))
    term.Printf("  Corrupt   : %d (%.1f%%)\n", romChkromStats.Corrupt, 100.0 * float32(romChkromStats.Corrupt) / float32(romChkromStats.Total))
    term.Printf("  Bad Name  : %d (%.1f%%)\n", romChkromStats.BadName, 100.0 * float32(romChkromStats.BadName) / float32(romChkromStats.Total))
    term.Printf("  Missing   : %d (%.1f%%)\n", romChkromStats.Missing, 100.0 * float32(romChkromStats.Missing) / float32(romChkromStats.Total))
    term.Printf("  Total     : %d\n", romChkromStats.Total)
    term.Printf("  Extra     : %d\n", romChkromStats.Extra)
}

func (log *StdLogger) close() {
}

///////////////////////////////////////////////////////////////////////////////
// JSON Logger
///////////////////////////////////////////////////////////////////////////////

type JsonRom struct {
    Status string                   `json:"status"`
    Info string                     `json:"info"`
}

type JsonMachine struct {
    Status string                   `json:"status"`
    Path string                     `json:"path"`
    Info string                     `json:"info"`
    Roms map[string]JsonRom         `json:"roms"`
}

type JsonHeader struct {
    Name string                     `json:"name"`
}

type JsonSchema struct {
    Header JsonHeader               `json:"header"`
    Machines map[string]JsonMachine `json:"machines"`
    Extras []string                 `json:"extras"`
}

type JsonLogger struct {
    value JsonSchema
    machLast string
}

func NewJsonLogger() *JsonLogger {
    return &JsonLogger{value: JsonSchema{Machines: make(map[string]JsonMachine)}}
}

func (log *JsonLogger) header(name string) {
    log.value.Header.Name = name
}

func (log *JsonLogger) machine(machine *gorom.Machine, status int, info ...string) {
    var str string
    switch {
    case status == MachOk:
        str = "ok"
    case status == MachMissing:
        str = "missing"
    case status == MachCorrupt:
        str = "corrupt"
    case (status & MachRomCorrupt != 0) || (status & MachRomBadName != 0) || (status & MachRomMissing != 0):
        str = "errors"
    case (status & MachRomExtra != 0):
        str = "extra"
    default:
        panic("invalid machine status")
    }
    log.machLast = machine.Name
    jsonMach := JsonMachine{
        Status: str, Path: machine.Path, Roms: make(map[string]JsonRom) }
    if len(info) > 0 {
        jsonMach.Info = info[0]
    }
    log.value.Machines[machine.Name] = jsonMach
}

func (log *JsonLogger) extra(path string) {
    log.value.Extras = append(log.value.Extras, path)
}

func (log *JsonLogger) rom(rom *gorom.Rom, status int, info ...string) {
    var str string
    switch status {
    case gorom.RomOk:
        str = "ok"
    case gorom.RomMissing:
        str = "missing"
    case gorom.RomUnknown:
        str = "extra"
    case gorom.RomCorrupt:
        str = "corrupt"
    case gorom.RomBadName:
        str = "badname"
    default:
        panic("invalid rom status")
    }
    jsonRom := JsonRom{Status: str}
    if len(info) > 0 {
        jsonRom.Info = info[0]
    }
    log.value.Machines[log.machLast].Roms[rom.Name] = jsonRom
}

func (log *JsonLogger) machineChkromStats(machChkromStats *ChkromStats, machRomChkromStats *ChkromStats) {
}

func (log *JsonLogger) romChkromStats(romChkromStats *ChkromStats) {
}

func (log *JsonLogger) close() {
    j, _ := json.MarshalIndent(log.value, "", "  ")
    term.Printf("%s", j)
}

///////////////////////////////////////////////////////////////////////////////
// Check ROMs
///////////////////////////////////////////////////////////////////////////////

type ValidResults struct {
    machine *gorom.Machine
    badNames map[string]string
    extras []string
    ok bool
    err error
}

var (
    romChkromStats ChkromStats
    machChkromStats ChkromStats
    machRomChkromStats ChkromStats
    logger Logger
)

func chkromValidate(machine *gorom.Machine, rdb *gorom.RomDB, ch chan ValidResults) {
    badNames := map[string]string{}
    extras := []string{}

    var err error
    var ok bool
    if !options.ChkRom.SizeOnly {
        ok, err = gorom.ValidateChecksums(machine, rdb, badNames, &extras, nil)
    } else {
        ok, err = gorom.ValidateSizes(machine, &extras, nil)
    }

    ch <- ValidResults{ machine: machine, badNames: badNames, extras: extras, ok: ok, err: err}
}

func chkromResults(ch chan ValidResults) {
    results := <- ch

    machine := results.machine
    badNames := results.badNames
    extras := results.extras
    ok := results.ok
    err := results.err

    machChkromStats.Total++
    machStatus := MachOk

    if err != nil {
        logger.machine(machine, MachCorrupt, err.Error())
        romChkromStats.Total += len(machine.Roms)
        romChkromStats.Corrupt += len(machine.Roms);
        machChkromStats.Corrupt++
    } else if ok {
        for _, rom := range machine.Roms {
            switch rom.Status {
            case gorom.RomCorrupt:
                machStatus |= MachRomCorrupt
            case gorom.RomBadName:
                machStatus |= MachRomBadName
            case gorom.RomMissing:
                machStatus |= MachRomMissing
            }
        }

        if len(extras) > 0 {
            machRomChkromStats.Extra++
            machStatus |= MachRomExtra
        }

        if machStatus == MachOk {
            if !options.App.NoOk {
                logger.machine(machine, machStatus)
            }
            machChkromStats.Ok++
        } else if machStatus == MachRomExtra {
            logger.machine(machine, machStatus)
        } else {
            logger.machine(machine, machStatus)
            if machStatus & MachRomCorrupt != 0 {
                machRomChkromStats.Corrupt++
            }
            if machStatus & MachRomBadName != 0 {
                machRomChkromStats.BadName++
            }
            if machStatus & MachRomMissing != 0 {
                machRomChkromStats.Missing++
            }
        }

        for _, rom := range machine.Roms {
            romChkromStats.Total++

            switch rom.Status {
            case gorom.RomOk:
                if !options.App.NoOk && !options.ChkRom.NoRom {
                    logger.rom(rom, rom.Status)
                }
                romChkromStats.Ok++
            case gorom.RomCorrupt:
                if !options.ChkRom.NoRom {
                    logger.rom(rom, rom.Status)
                }
                romChkromStats.Corrupt++;
            case gorom.RomBadName:
                if !options.ChkRom.NoRom {
                    logger.rom(rom, rom.Status, badNames[rom.Name])
                }
                romChkromStats.BadName++;
            case gorom.RomMissing:
                if !options.ChkRom.NoRom {
                    logger.rom(rom, rom.Status)
                }
                romChkromStats.Missing++;
            }
        }

        for _, name := range extras {
            if !options.ChkRom.NoRom {
                logger.rom(&gorom.Rom{Name: name}, gorom.RomUnknown)
            }
            romChkromStats.Extra++
        }
    } else {
        logger.machine(machine, MachMissing)
        romChkromStats.Total += len(machine.Roms)
        romChkromStats.Missing += len(machine.Roms);
        machChkromStats.Missing++
    }
}

func chkrom(datFile string, machines []string) (bool, error) {
    romChkromStats = ChkromStats{}
    machChkromStats = ChkromStats{}
    machRomChkromStats = ChkromStats{}

    if options.ChkRom.JsonOut {
        logger = NewJsonLogger()
    } else {
        logger = NewStdLogger()
    }

    machSet := gorom.NewStringSet()

    var rdb *gorom.RomDB
    var err error
    if !options.ChkRom.SizeOnly {
        rdb, err = gorom.OpenRomDB(".")
        if err != nil {
            return false, err
        }
        defer rdb.Close()
    }

    ch := make(chan ValidResults, 1)

    goCount := 0
    goLimit := 1
    if !options.App.NoGo {
        goLimit = runtime.NumCPU()
    }

    err = gorom.ParseDatFile(datFile, machines, func(header *gorom.Header) error {
        if !options.App.NoHeader {
            logger.header(header.Name)
        }
        gorom.Progressf("Parsing DAT file...\n")
        return nil
    }, func(machine *gorom.Machine) error {
        machSet.Set(machine.Name)

        if goCount == goLimit {
            chkromResults(ch)
        } else {
            goCount++
        }

        go chkromValidate(machine, rdb, ch)

        return nil
    })
    if err != nil {
        return false, err
    }

    for ; goCount > 0; goCount-- {
        chkromResults(ch)
    }

    if len(machines) == 0 && !options.App.NoExtra {
        err = gorom.ScanDir(".", true, func(file os.FileInfo) error {
            path := file.Name()
            name := gorom.MachName(path)
            if !machSet.IsSet(name) {
                machChkromStats.Extra++;
                logger.extra(path)
            }
            return nil
        })
        if err != nil {
            return false, err
        }
    }

    if !options.ChkRom.NoStats {
        if (machChkromStats.Total > 0) {
            logger.machineChkromStats(&machChkromStats, &machRomChkromStats)
        }

        if (romChkromStats.Total > 0) {
            logger.romChkromStats(&romChkromStats)
        }
    }

    logger.close()

    return (romChkromStats.Ok == romChkromStats.Total), nil
}
