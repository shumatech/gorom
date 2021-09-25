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
    "path"
    "gorom"
    "gorom/test"
    "fmt"
)

func runFixRom(t *testing.T, datFile string, machines []string, dirs []string) error {
    wd, err := os.Getwd()
    if err != nil {
        return err
    }
    defer os.Remove(path.Join(wd, ".gorom.db"))

    tmpdir := test.CopyDirToTemp(t, "..", wd)
    defer os.RemoveAll(tmpdir)

    err = os.Chdir(tmpdir)
    if err != nil {
        return err
    }

    for _, dir := range dirs {
        defer os.Remove(path.Join(dir, ".gorom.db"))
    }

    for i := 0; i < 2; i++ {
        ok, err := fixrom(datFile, machines, dirs)
        if err != nil {
            return err
        }
        if !ok {
            return fmt.Errorf("unexpected return value")
        }
    }

    return nil
}

func TestFixRomZipWithZip(t *testing.T) {
    test.RunDiffTest(t, "roms/badzip", "fixrom/badzip_zip.out", func() error {
        options = Options{}
        options.FixRom.Format = gorom.FormatZip
        return runFixRom(t, "../../dats/zip.dat", nil, []string{"../zip"})
    })
}

func TestFixRomZipWithDir(t *testing.T) {
    test.RunDiffTest(t, "roms/badzip", "fixrom/badzip_dir.out", func() error {
        options = Options{}
        options.FixRom.Format = gorom.FormatZip
        return runFixRom(t, "../../dats/zip.dat", nil, []string{"../dir"})
    })
}

func TestFixRomDirWithZip(t *testing.T) {
    test.RunDiffTest(t, "roms/baddir", "fixrom/baddir_zip.out", func() error {
        options = Options{}
        options.FixRom.Format = gorom.FormatDir
        return runFixRom(t, "../../dats/dir.dat", nil, []string{"../zip"})
    })
}

func TestFixRomDirWithDir(t *testing.T) {
    test.RunDiffTest(t, "roms/baddir", "fixrom/baddir_dir.out", func() error {
        options = Options{}
        options.FixRom.Format = gorom.FormatDir
        return runFixRom(t, "../../dats/dir.dat", nil, []string{"../dir"})
    })
}
