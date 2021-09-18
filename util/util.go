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
package util

import (
    "fmt"
    "math"
    "os"
    "os/signal"
    "path/filepath"
    "sort"
    "strings"

    "gorom/term"

    "github.com/klauspost/compress/zip"
)

var (
    Progress = true
)

///////////////////////////////////////////////////////////////////////////////
// Progress Print
///////////////////////////////////////////////////////////////////////////////
func Progressf(format string, a ...interface{}) {
    if (term.IsTerminal && Progress) {
        str := fmt.Sprintf(format, a...)
        width := len(str)
        if (width >= term.TerminalWidth) {
            width = term.TerminalWidth - 1
        }

        term.Print("\r")
        term.Print(str[:width])
        term.ClrEol();
    }
}

///////////////////////////////////////////////////////////////////////////////
// Convert paths to slashes
///////////////////////////////////////////////////////////////////////////////
func ToSlash(paths []string) []string {
    slashes := make([]string, len(paths))
    for i := range paths {
        slashes[i] = filepath.ToSlash(paths[i])
    }
    return slashes
}

///////////////////////////////////////////////////////////////////////////////
// String Set (syntatic sugar)
///////////////////////////////////////////////////////////////////////////////

type StringSet map[string]bool

func NewStringSet() StringSet {
    return map[string]bool{}
}

func (set StringSet) Set(key string) {
    set[key] = true
}

func (set StringSet) Unset(key string) {
    delete(set, key)
}

func (set StringSet) IsSet(key string) bool {
    _, ok := set[key]
    return ok
}

///////////////////////////////////////////////////////////////////////////////
// String BiMap
///////////////////////////////////////////////////////////////////////////////

type StringBiMap struct {
    keys map[string]string
    values map[string]string
}

func NewStringBiMap() *StringBiMap {
    return &StringBiMap{
        keys: make(map[string]string),
        values: make(map[string]string),
    }
}

func (sbm *StringBiMap) Get(key string) (value string, ok bool) {
    value, ok =  sbm.keys[key]
    return
}

func (sbm *StringBiMap) GetValue(value string) (key string, ok bool) {
    key, ok =  sbm.values[value]
    return
}

func (sbm *StringBiMap) Set(key string, value string) bool {
    if _, ok :=  sbm.keys[key]; ok {
        return false
    }
    if _, ok :=  sbm.values[value]; ok {
        return false
    }

    sbm.keys[key] = value
    sbm.values[value] = key

    return true
}

func (sbm *StringBiMap) Delete(key string, value string) bool {
    if _, ok :=  sbm.keys[key]; !ok {
        return false
    }
    if _, ok :=  sbm.values[value]; !ok {
        return false
    }

    delete(sbm.keys, key)
    delete(sbm.values, value)

    return true
}

func (sbm *StringBiMap) Keys() map[string]string {
    return sbm.keys
}

func (sbm *StringBiMap) Values() map[string]string {
    return sbm.values
}

///////////////////////////////////////////////////////////////////////////////
// Signal Handling
///////////////////////////////////////////////////////////////////////////////

func SignalInit(onExit func()) {
    // Ignore all signals by default
    signal.Ignore()

    c := make(chan os.Signal, 1)

    // Only handle termination signals
    signal.Notify(c, os.Interrupt, os.Kill)

    go func() {
        s := <-c
        term.Printf("\n%s signal received -- exiting\n", s)
        if onExit != nil {
            onExit()
        }
        os.Exit(1)
    }()
}

///////////////////////////////////////////////////////////////////////////////
// ScanDir - Execute a callback for every file/directory in directory.  If
// the callback returns false then scanning stops. If the callback returns an
// error then scanning stops and the error is propogated up.
///////////////////////////////////////////////////////////////////////////////

type FileInfoFunc func(file os.FileInfo) error

func ScanDir(dir string, skipDot bool, fileFunc FileInfoFunc) error {
    if dir == "" {
        dir = "."
    }
    fh, err := os.Open(dir)
    if err != nil {
        return err
    }

    flist, err := fh.Readdir(0)
    if err != nil {
        return err
    }

    fh.Close()

    sort.Slice(flist, func(i, j int) bool {
        return flist[i].Name() < flist[j].Name()
    })

    for _, file := range flist {
        // Skip dot files
        if skipDot && file.Name()[0] == '.' {
            continue
        }

        err := fileFunc(file)
        if err != nil {
            return err
        }
    }

    return nil
}

///////////////////////////////////////////////////////////////////////////////
// ScanZip - Execute a callback for every file in the Zip. If the callback
// returns an error then scanning stops and the error is propogated up.
///////////////////////////////////////////////////////////////////////////////

type ZipInfoFunc func(fh *zip.File) (bool, error)

func ScanZip(path string, zipFunc ZipInfoFunc) error {
    rc, err := zip.OpenReader(path)
    if err != nil {
        return err
    }

    defer rc.Close()

    for _, fh := range rc.File {
        ok, err := zipFunc(fh)
        if err != nil {
            return err
        } else if !ok {
            break
        }
    }
    return nil
}

///////////////////////////////////////////////////////////////////////////////
// HumanizePow2 - Humanize a power of 2 number
///////////////////////////////////////////////////////////////////////////////

func HumanizePow2(num int64) string {
    sizes := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB" }

    fnum := float64(num)
    var i int
    for i = 0; i  < len(sizes); i++ {
        if math.Round(fnum * 1000) < 1024000 {
            break
        }
        fnum /= 1024
    }

    str := fmt.Sprintf("%.3f", fnum)
    str = strings.TrimRight(str, "0")
    str = strings.TrimRight(str, ".")

    return (str + " " + sizes[i])
}
