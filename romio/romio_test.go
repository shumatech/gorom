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
package romio

import (
    "testing"
    "os"

    "gorom/test"
    "gorom/checksum"
)

func romReaderTest(t *testing.T, machName string, machine *test.Machine) {
    rr, err := OpenRomReaderByName(machName)
    if rr == nil {
        test.Fail(t, "machine not found: " + machName)
    }
    if err != nil {
        test.Fail(t, err)
    }
    defer rr.Close()

    if rr.Name() != machName {
        test.Fail(t, "wrong name")
    }
    path := machName
    if !IsDirReader(rr) {
        path += ".zip"
    }
    if rr.Path() != path {
        test.Fail(t, "wrong path")
    }
    if len(rr.Files()) != len(machine.Roms) {
        test.Fail(t, "wrong number of files")
    }
    for name, rom := range machine.Roms {
        file := rr.Stat(name)
        if file == nil {
            test.Fail(t, "file not found")
        }
        if file.Name != name {
            test.Fail(t, "name mismatch")
        }
        if file.Size != rom.Size {
            test.Fail(t, "size mismatch")
        }
    }
}

func runRomReaderTest(t *testing.T, df *test.DatFile) {
    defer test.Chdir(t, df.DataPath)()
    for machName, machine := range df.Machines {
        romReaderTest(t, machName, &machine)
    }
}

func TestRomReaderZip(t *testing.T) {
    test.ForEachDat(t, test.ZipDats, runRomReaderTest)
}

func TestRomReaderDir(t *testing.T) {
    test.ForEachDat(t, test.DirDats, runRomReaderTest)
}

func romWriterTest(t *testing.T, machName string, isDir bool, machine *test.Machine) {
    rw, err := CreateRomWriterTemp(".", isDir)
    if err != nil {
        test.Fail(t, err)
    }

    defer os.RemoveAll(rw.Path())
    defer rw.Close()

    rr, err := OpenRomReaderByName(machName)
    if rr == nil {
        test.Fail(t, "machine not found: " + machName)
    }
    if err != nil {
        test.Fail(t, err)
    }
    defer rr.Close()

    files := rr.Files()
    for _, file := range files {
        err = rw.Create(file.Name)
        if err != nil {
            test.Fail(t, err)
        }
    }

    for i := rw.First(); i >= 0; i = rw.Next() {
        file := files[i]
        err = CopyRom(rw, file.Name, rr, file.Name)
        if err != nil {
            test.Fail(t, err)
        }
    }

    rw.Close()

    romReaderTest(t, rw.Name(), machine)
}

func runRomWriterTest(t *testing.T, df *test.DatFile) {
    defer test.Chdir(t, df.DataPath)()
    for machName, machine := range df.Machines {
        romWriterTest(t, machName, df.IsDir, &machine)
    }
}

func TestRomWriterZip(t *testing.T) {
    test.ForEachDat(t, test.ZipDats, runRomWriterTest)
}

func TestRomWriterDir(t *testing.T) {
    test.ForEachDat(t, test.DirDats, runRomWriterTest)
}

func runChecksumMachTest(t *testing.T, df *test.DatFile) {
    defer test.Chdir(t, df.DataPath)()

    for machName, machine := range df.Machines {
        count := 0
        err := ChecksumMach(df.MachPath(machName),
                           func (actName string, actSize int64,
                                 actCrc32 checksum.Crc32, actSha1 checksum.Sha1) error {
            rom, ok := machine.Roms[actName]
            if !ok {
                test.Fail(t, "unexpected file")
            }
            if rom.Size != actSize {
                test.Fail(t, "size mismatch")
            }
            expCrc32, ok := checksum.NewCrc32String(rom.Crc32)
            if !ok {
                test.Fail(t, "invalid crc32")
            }
            if expCrc32 != actCrc32 {
                test.Fail(t, "crc32 mismatch")
            }
            expSha1, ok := checksum.NewSha1String(rom.Sha1)
            if !ok {
                test.Fail(t, "invalid sha1")
            }
            if expSha1 != actSha1 {
                test.Fail(t, "sha1 mismatch")
            }
            count++
            return nil
        })
        if err != nil {
            test.Fail(t, err)
        }
        if count != len(machine.Roms) {
            test.Fail(t, "file count mismatch")
        }
    }
}

func TestChecksumMachZip(t *testing.T) {
    test.ForEachDat(t, test.ZipDats, runChecksumMachTest)
}

func TestChecksumMachDir(t *testing.T) {
    test.ForEachDat(t, test.DirDats, runChecksumMachTest)
}