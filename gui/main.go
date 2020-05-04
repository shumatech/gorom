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
    "encoding/hex"
    "fmt"
    "gorom"
    "log"
    "os"
    "path/filepath"
    "runtime"

    "github.com/sciter-sdk/go-sciter"
    "github.com/sciter-sdk/go-sciter/window"
)

var (
    Version string
    Build string

    root *sciter.Element
    machines []*gorom.Machine

    stopChan chan struct{}
    stopError = fmt.Errorf("Operation stopped")
)

func isStop() bool {
    select {
    case <-stopChan:
        return true
    default:
        return false
    }
}

func argsToValues(args ...interface{}) []*sciter.Value {
    vals := []*sciter.Value{}
    for i := range args {
        var v *sciter.Value
        switch t := args[i].(type) {
            case nil:
                v = sciter.NullValue()
            case *sciter.Value:
                v = t
            default:
                v = sciter.NewValue(args[i])
        }
        vals = append(vals, v)
    }
    return vals
}

func call(fn string, args ...interface{}) {
    vals := argsToValues(args...)
    _, err := root.CallFunction(fn, vals...)
    if err != nil {
        log.Println("call failed:", err)
    }
}

func invoke(fn *sciter.Value, args ...interface{}) {
    vals := argsToValues(args...)
    _, err := fn.Invoke(sciter.NullValue(), "[Native Function]", vals...)
    if err != nil {
        log.Println("invoke failed:", err)
    }
}

func datLoad(args ...*sciter.Value) *sciter.Value {
    datFile := args[0].String()
    callback := args[1].Clone()

    go func() {
        machines = []*gorom.Machine{}

        dat := sciter.NewValue()
        array := sciter.NewValue()
        err := gorom.ParseDatFile(datFile, nil,
            func(header *gorom.Header) error {
                obj := sciter.NewValue()
                obj.Set("Name", header.Name)
                obj.Set("Description", header.Description)
                obj.Set("Version", header.Version)
                obj.Set("Author", header.Author)
                dat.Set("Header", obj)
                return nil
            },
            func(machine *gorom.Machine) error {
                machines = append(machines, machine)
                obj := sciter.NewValue()
                obj.Set("Name", machine.Name)
                obj.Set("Description", machine.Description)
                obj.Set("Manufacturer", machine.Manufacturer)
                obj.Set("Category", machine.Category)
                obj.Set("Year", machine.Year)
                obj.Set("ROMs", len(machine.Roms))
                obj.Set("Status", "-")
                array.Append(obj)
                return nil
            })
        if err != nil {
            call("errorMessage", err.Error())
            invoke(callback, nil)
            callback.Release()
            return
        }

        dat.Set("Machines", array)

        invoke(callback, dat)
        callback.Release()
    }()

    return sciter.NullValue()
}

func resetRoms(args ...*sciter.Value) *sciter.Value {
    for _, machine := range machines {
        for _, rom := range machine.Roms {
            rom.Status = gorom.RomUnknown
        }
    }
    return sciter.NullValue()
}

func getRoms(args ...*sciter.Value) *sciter.Value {
    index := args[0].Int()
    machine := machines[index]
    array := sciter.NewValue()

    for _, rom := range machine.Roms {
        obj := sciter.NewValue()
        obj.Set("Name", rom.Name)
        obj.Set("Size", float64(rom.Size))
        obj.Set("Crc", hex.EncodeToString(rom.Crc[:]))
        obj.Set("Sha1", hex.EncodeToString(rom.Sha1[:]))
        var str string
        switch rom.Status {
            case gorom.RomOk: str = "OK"
            case gorom.RomMissing: str = "MISSING"
            case gorom.RomCorrupt: str = "CORRUPT"
            case gorom.RomBadName: str = "BAD_NAME"
            case RomFixCopy: str = "COPIED"
            case RomFixNotFound: str = "NOT_FOUND"
            case RomFixRename: str = "RENAMED"
            default: str = "-"
        }
        obj.Set("Status", str)
        array.Append(obj)
    }

    return array
}

func chkrom(args ...*sciter.Value) *sciter.Value {
    dir := args[0].String()
    callback := args[1].Clone()
    stopChan = make(chan struct{})

    go func() {
        err := chkromStart(dir)
        if err != nil {
            call("errorMessage", err.Error())
            invoke(callback, nil)
            callback.Release()
            return;
        }

        statsToValue := func(stats *ChkromStats) *sciter.Value {
            obj := sciter.NewValue()
            obj.Set("Ok", stats.Ok)
            obj.Set("Missing", stats.Missing)
            obj.Set("Corrupt", stats.Corrupt)
            obj.Set("BadName", stats.BadName)
            obj.Set("Extra", stats.Extra)
            obj.Set("Total", stats.Total)
            return obj
        }

        stats := sciter.NewValue()
        stats.Set("RomStats", statsToValue(&chkRomStats))
        stats.Set("MachStats", statsToValue(&chkMachStats))
        stats.Set("MachRomStats", statsToValue(&chkMachRomStats))

        invoke(callback, stats)
        callback.Release()
    }()

    return sciter.NullValue()
}

func stop(args ...*sciter.Value) *sciter.Value {
    close(stopChan)
    return sciter.NullValue()
}

func fixrom(args ...*sciter.Value) *sciter.Value {
    dir := args[0].String()

    srcDirs := []string{}
    for i := 0; i < args[1].Length(); i++ {
        srcDirs = append(srcDirs, args[1].Index(i).String())
    }
    callback := args[2].Clone()

    stopChan = make(chan struct{})
    go func() {
        err := fixromStart(dir, srcDirs, false)
        if err != nil {
            call("errorMessage", err.Error())
            invoke(callback, nil)
            callback.Release()
            return
        }

        stats := sciter.NewValue()

        machStats := sciter.NewValue()
        machStats.Set("Ok", fixMachStats.Ok)
        machStats.Set("Fixed", fixMachStats.Fixed)
        machStats.Set("Failed", fixMachStats.Failed)
        machStats.Set("Total", fixMachStats.Total)
        stats.Set("MachStats", machStats)

        romStats := sciter.NewValue()
        romStats.Set("Ok", fixRomStats.Ok)
        romStats.Set("Copied", fixRomStats.Copied)
        romStats.Set("Renamed", fixRomStats.Renamed)
        romStats.Set("NotFound", fixRomStats.NotFound)
        romStats.Set("Total", fixRomStats.Total)
        stats.Set("RomStats", romStats)

        invoke(callback, stats)
        callback.Release()
    }()

    return sciter.NullValue()
}

func about(args ...*sciter.Value) *sciter.Value {
    obj := sciter.NewValue()
    obj.Set("version", Version)
    obj.Set("build", Build)
    return obj
}

func setEventHandler(w *window.Window) {
    w.DefineFunction("datLoad", datLoad)
    w.DefineFunction("getRoms", getRoms)
    w.DefineFunction("resetRoms", resetRoms)
    w.DefineFunction("chkrom", chkrom)
    w.DefineFunction("stop", stop)
    w.DefineFunction("fixrom", fixrom)
    w.DefineFunction("about", about)
}

func init() {
    runtime.LockOSThread()
}

func main() {
    sciter.SetOption(sciter.SCITER_SET_SCRIPT_RUNTIME_FEATURES, 
        sciter.ALLOW_FILE_IO |
        sciter.ALLOW_SOCKET_IO |
        sciter.ALLOW_EVAL |
        sciter.ALLOW_SYSINFO)

    flags := sciter.SW_TITLEBAR |
        sciter.SW_RESIZEABLE |
        sciter.SW_CONTROLS |
        sciter.SW_MAIN

    if Debug {
        flags |= sciter.SW_ENABLE_DEBUG
    }

    w, err := window.New(flags, sciter.NewRect(0, 0, 1024, 768))
    if err != nil {
        log.Fatal(err)
    }

    if Debug {
        ok := w.SetOption(sciter.SCITER_SET_DEBUG_MODE, 1)
        if !ok {
            log.Fatalf("set debug mode failed")
        }
        wd, _ := os.Getwd();
        err = w.LoadFile("file://" + filepath.ToSlash(wd) + "/res/main.html")
    } else {
        w.SetResourceArchive(resources)
        err = w.LoadFile("this://app/main.html")
    }
    if err != nil {
        log.Fatalf("LoadFile failed: %s", err.Error())
    }

    root, err = w.GetRootElement()
    if err != nil {
        log.Fatalf("GetRootElement failed: %s", err.Error())
    }

    w.SetTitle("GoROM")
    setEventHandler(w)
    w.Show()
    w.Run()
}
