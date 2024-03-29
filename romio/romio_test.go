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
	"fmt"
	"os"
	"path"
	"testing"

	"gorom/checksum"
	"gorom/test"
)

func romReaderTest(t *testing.T, machName string, machPath string, format int, machine *test.Machine) {
    if !IsRomReader(format) {
        return
    }
    rr, err := OpenRomReaderByName(machName)
    if err != nil {
        test.Fail(t, err)
    }
    if rr == nil {
        test.Fail(t, "machine not found: " + machName)
    }
    defer rr.Close()

    if rr.Name() != machName {
        test.Fail(t, fmt.Sprintf("wrong name: %s != %s", rr.Name(), machName))
    }
    if rr.Format() != format {
        test.Fail(t, fmt.Sprintf("format mismatch: %d != %d", rr.Format(), format))
    }
    if rr.Path() != machPath {
        test.Fail(t, fmt.Sprintf("wrong path: %s != %s", rr.Path(), machPath))
    }
    if len(rr.Files()) != len(machine.Roms) {
        test.Fail(t, fmt.Sprintf("wrong number of files: %d != %d", len(rr.Files()), len(machine.Roms)))
    }
    for name, rom := range machine.Roms {
        file := rr.Stat(name)
        if file == nil {
            test.Fail(t, fmt.Sprintf("file not found: %s", name))
        }
        if file.Name != name {
            test.Fail(t, fmt.Sprintf("name mismatch: %s != %s", file.Name, name))
        }
        if file.Size != rom.Size {
            test.Fail(t, fmt.Sprintf("size mismatch: %d != %d", file.Size, rom.Size))
        }
    }
}

func runRomReaderTest(t *testing.T, df *test.DatFile) {
    defer test.Chdir(t, df.DataPath)()
    for machName, machine := range df.Machines {
        romReaderTest(t, machName, df.MachPath(machName), df.MachFormat(machName), &machine)
    }
}

func TestRomReaderZip(t *testing.T) {
    test.ForEachDat(t, test.ZipDats, runRomReaderTest)
}

func TestRomReaderDir(t *testing.T) {
    test.ForEachDat(t, test.DirDats, runRomReaderTest)
}

func TestRomReaderArchive(t *testing.T) {
    test.ForEachDat(t, test.ArchiveDats, runRomReaderTest)
}

func romWriterTest(t *testing.T, machName string, format int, machine *test.Machine) {
    if !IsRomWriter(format) {
        return
    }
    rw, err := CreateRomWriterTemp(".", format)
    if err != nil {
        test.Fail(t, err)
    }

    defer os.RemoveAll(rw.Path())
    defer rw.Close()

    rr, err := OpenRomReaderByName(machName)
    if err != nil {
        test.Fail(t, err)
    }
    if rr == nil {
        test.Fail(t, "machine not found: " + machName)
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

    romReaderTest(t, rw.Name(), path.Base(rw.Path()), format, machine)
}

func runRomWriterTest(t *testing.T, df *test.DatFile) {
    defer test.Chdir(t, df.DataPath)()
    for machName, machine := range df.Machines {
        romWriterTest(t, machName, df.MachFormat(machName), &machine)
    }
}

func TestRomWriterZip(t *testing.T) {
    test.ForEachDat(t, test.ZipDats, runRomWriterTest)
}

func TestRomWriterDir(t *testing.T) {
    test.ForEachDat(t, test.DirDats, runRomWriterTest)
}

func TestRomWriterArchive(t *testing.T) {
    test.ForEachDat(t, test.ArchiveDats, runRomWriterTest)
}

func runChecksumMachTest(t *testing.T, df *test.DatFile) {
    defer test.Chdir(t, df.DataPath)()

    for machName, machine := range df.Machines {
        count := 0
        err := ChecksumMach(df.MachPath(machName), ChecksumSkipHeader,
                           func (actName string,
                                 actChecksums Checksums) error {
            rom, ok := machine.Roms[actName]
            if !ok {
                test.Fail(t, fmt.Sprintf("unexpected file: %s", actName))
            }
            if rom.Size != actChecksums.Size {
                test.Fail(t, fmt.Sprintf("size mismatch: %d != %d", rom.Size, actChecksums.Size))
            }
            expCrc32, ok := checksum.NewCrc32String(rom.Crc32)
            if !ok {
                test.Fail(t, "invalid crc32")
            }
            if expCrc32 != actChecksums.Crc32 {
                test.Fail(t, "crc32 mismatch")
            }
            expSha1, ok := checksum.NewSha1String(rom.Sha1)
            if !ok {
                test.Fail(t, "invalid sha1")
            }
            if expSha1 != actChecksums.Sha1 {
                test.Fail(t, "sha1 mismatch")
            }
            count++
            return nil
        })
        if err != nil {
            test.Fail(t, err)
        }
        if count != len(machine.Roms) {
            test.Fail(t, fmt.Sprintf("file count mismatch: %d != %d", count, len(machine.Roms)))
        }
    }
}

func TestChecksumMachZip(t *testing.T) {
    test.ForEachDat(t, test.ZipDats, runChecksumMachTest)
}

func TestChecksumMachDir(t *testing.T) {
    test.ForEachDat(t, test.DirDats, runChecksumMachTest)
}

func TestChecksumMachHeader(t *testing.T) {
    test.ForEachDat(t, test.HeaderDats, runChecksumMachTest)
}

func TestChecksumMachArchive(t *testing.T) {
    test.ForEachDat(t, test.ArchiveDats, runChecksumMachTest)
}
