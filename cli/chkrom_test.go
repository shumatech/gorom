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
    "testing"
    "os"
    "fmt"
    "gorom/term"
    "gorom/test"
)

func runChkRom(t *testing.T, datFile string, machines []string, expOk bool) error {
    term.Init()
    options.App.NoGo = true
    ok, err := chkrom(datFile, machines)
    os.Remove(".gorom.db")
    if err != nil {
        return err
    }
    if ok != expOk {
        return fmt.Errorf("unexpected return value")
    }
    return nil
}

func TestChkRomZip(t *testing.T) {
    test.RunDiffTest(t, "roms/zip", "chkrom/zip.out", func() error {
        options = Options{}
        return runChkRom(t, "../../dats/zip.dat", nil, true)
    })
}

func TestChkRomDir(t *testing.T) {
    test.RunDiffTest(t, "roms/dir", "chkrom/dir.out", func() error {
        options = Options{}
        return runChkRom(t, "../../dats/dir.dat", nil, true)
    })
}

func TestChkRomBadZip(t *testing.T) {
    test.RunDiffTest(t, "roms/badzip", "chkrom/badzip.out", func() error {
        options = Options{}
        return runChkRom(t, "../../dats/zip.dat", nil, false)
    })
}

func TestChkRomBadDir(t *testing.T) {
    test.RunDiffTest(t, "roms/baddir", "chkrom/baddir.out", func() error {
        options = Options{}
        return runChkRom(t, "../../dats/dir.dat", nil, false)
    })
}

func TestChkRomCorruptZip(t *testing.T) {
    test.RunDiffTest(t, "roms/corruptzip", "chkrom/corruptzip.out", func() error {
        options = Options{}
        return runChkRom(t, "../../dats/zip.dat", nil, false)
    })
}

func TestChkRomSizeZip(t *testing.T) {
    test.RunDiffTest(t, "roms/zip", "chkrom/zip.out", func() error {
        options = Options{}
        options.ChkRom.SizeOnly = true
        return runChkRom(t, "../../dats/zip.dat", nil, true)
    })
}

func TestChkDirSizeZip(t *testing.T) {
    test.RunDiffTest(t, "roms/dir", "chkrom/dir.out", func() error {
        options = Options{}
        options.ChkRom.SizeOnly = true
        return runChkRom(t, "../../dats/dir.dat", nil, true)
    })
}

func TestChkRomSizeBadZip(t *testing.T) {
    test.RunDiffTest(t, "roms/badzip", "chkrom/sizebadzip.out", func() error {
        options = Options{}
        options.ChkRom.SizeOnly = true
        return runChkRom(t, "../../dats/zip.dat", nil, false)
    })
}

func TestChkRomBadZipJson(t *testing.T) {
    test.RunDiffTest(t, "roms/badzip", "chkrom/badzipjson.out", func() error {
        options = Options{}
        options.ChkRom.JsonOut = true
        return runChkRom(t, "../../dats/zip.dat", nil, false)
    })
}
