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
package dat

import (
    "testing"
    "fmt"
    "os"

    "gorom"
    "gorom/test"
    "gorom/romdb"
    "gorom/checksum"
)

func headerEqual(header *Header, df *test.DatFile) bool {
    return header.Name == df.Name &&
           header.Description == df.Description &&
           header.Author == df.Author
}

func romEqual(rom *Rom, testRom *test.Rom) bool {
    crc, ok := checksum.NewCrc32String(testRom.Crc32)
    if !ok {
        return false
    }
    sha1, ok := checksum.NewSha1String(testRom.Sha1)
    if !ok {
        return false
    }
    return rom.Size == testRom.Size &&
           rom.Crc == crc &&
           rom.Sha1 == sha1
}

func machineEqual(machine *Machine, testMap test.MachineMap) bool {
    testMachine, ok := testMap[machine.Name]
    if !ok {
        return false
    }
    if machine.Description != testMachine.Description ||
       machine.Year != testMachine.Year ||
       machine.Manufacturer != testMachine.Manufacturer ||
       machine.Category != testMachine.Category {
        return false
    }
    for _, rom := range machine.Roms {
        testRom, ok := testMachine.Roms[rom.Name]
        if !ok {
            return false
        }
        if !romEqual(rom, &testRom) {
            return false
        }

    }
    return true
}

func runDatFileTest(t *testing.T, df *test.DatFile) {
    defer test.Chdir(t, "")()
    index := 0
    err := ParseDatFile(df.Path, []string{}, func(header *Header) error {
        if !headerEqual(header, df) {
            return fmt.Errorf("header fields do not match")
        }
        return nil
    }, func(machine *Machine) error {
        if !machineEqual(machine, df.Machines) {
            return fmt.Errorf("machine fields do not match")
        }
        index++
        return nil
    })
    if err != nil {
        test.Fail(t, err)
    }
    if index != len(df.Machines) {
        test.Fail(t, "machine count mismatch")
    }
}

func TestDatFileZip(t *testing.T) {
    test.ForEachDat(t, test.ZipDats, runDatFileTest)
}

func TestDatFileDir(t *testing.T) {
    test.ForEachDat(t, test.DirDats, runDatFileTest)
}

func createMachine(machName string, testMach *test.Machine) *Machine {
    machine := &Machine{ machName, testMach.Description, testMach.Year,
        testMach.Manufacturer, testMach.Category, nil, "", gorom.FormatZip }

    for romName, testRom := range testMach.Roms {
        crc32, ok := checksum.NewCrc32String(testRom.Crc32)
        if !ok {
            return nil
        }
        sha1, ok := checksum.NewSha1String(testRom.Sha1)
        if !ok {
            return nil
        }
        machine.Roms = append(machine.Roms, &Rom{ romName, testRom.Size, crc32, sha1, 0 })
    }
    return machine
}

func validateChecksumTest(t *testing.T, df *test.DatFile) {
    rdb, err := romdb.OpenRomDB(".", false)
    if err != nil {
        test.Fail(t, err)
    }
    defer rdb.Close()

    machIndex := 0
    for machName, testMach := range df.Machines {
        machine := createMachine(machName, &testMach)
        if machine == nil {
            test.Fail(t, "unable to create machine")
        }
        romIndex := 0
        ok, err := ValidateChecksums(machine, rdb, nil, nil, func(name string, sha1 checksum.Sha1) error {
            expSha1, ok := checksum.NewSha1String(testMach.Roms[name].Sha1)
            if !ok {
                return fmt.Errorf("invalid sha1")
            }
            if sha1 != expSha1 {
                return fmt.Errorf("checksum mismatch in machine %s rom %s", machine.Name, name)
            }
            romIndex++
            return nil;
        })
        if err != nil {
            test.Fail(t, err)
        }
        if !ok {
            test.Fail(t, "machine " + machine.Name + " not found")
        }
        if romIndex != len(machine.Roms) {
            test.Fail(t, "ROM count mismatch")
        }
        for _, rom := range machine.Roms {
            if rom.Status != RomOk {
                test.Fail(t, "ROM not OK")
            }
        }
        if machine.Path != df.MachPath(machName) {
            test.Fail(t, fmt.Sprintf("machine path mismatch: %s != %s", machine.Path, df.MachPath(machName)))
        }
        if machine.Format != df.MachFormat(machName) {
            test.Fail(t, fmt.Sprintf("machine format mismatch: %d != %d", machine.Format, df.MachFormat(machName)))
        }
        machIndex++
    }
    if machIndex != len(df.Machines) {
        test.Fail(t, "machine count mismatch")
    }
}

func runValidateChecksumTest(t *testing.T, df *test.DatFile) {
    defer test.Chdir(t, df.DataPath)()
    defer os.Remove(".gorom.db")
    // Run twice - first without a database and second with one
    validateChecksumTest(t, df)
    validateChecksumTest(t, df)
}

func TestValidateChecksumZip(t *testing.T) {
    test.ForEachDat(t, test.ZipDats, runValidateChecksumTest)
}

func TestValidateChecksumDir(t *testing.T) {
    test.ForEachDat(t, test.DirDats, runValidateChecksumTest)
}

func validateSizeTest(t *testing.T, df *test.DatFile) {
    machIndex := 0
    for machName, testMach := range df.Machines {
        machine := createMachine(machName, &testMach)
        if machine == nil {
            test.Fail(t, "unable to create machine")
        }
        romIndex := 0
        ok, err := ValidateSizes(machine, nil, func(name string, size int64) {
            if size != testMach.Roms[name].Size {
                t.Fatalf("size mismatch in machine %s rom %s", machine.Name, name)
            }
            romIndex++
        })
        if err != nil {
            test.Fail(t, err)
        }
        if !ok {
            test.Fail(t, "machine " + machine.Name + " not found")
        }
        if romIndex != len(machine.Roms) {
            test.Fail(t, "ROM count mismatch")
        }
        for _, rom := range machine.Roms {
            if rom.Status != RomOk {
                test.Fail(t, "ROM not OK")
            }
        }
        if machine.Path != df.MachPath(machName) {
            test.Fail(t, fmt.Sprintf("machine path mismatch: %s != %s", machine.Path, df.MachPath(machName)))
        }
        if machine.Format != df.MachFormat(machName) {
            test.Fail(t, fmt.Sprintf("machine format mismatch: %d != %d", machine.Format, df.MachFormat(machName)))
        }
        machIndex++
    }
    if machIndex != len(df.Machines) {
        test.Fail(t, "machine count mismatch")
    }
}

func runValidateSizeTest(t *testing.T, df *test.DatFile) {
    defer test.Chdir(t, df.DataPath)()
    defer os.Remove(".gorom.db")
    // Run twice - first without a database and second with one
    validateSizeTest(t, df)
    validateSizeTest(t, df)
}

func TestValidateSizeZip(t *testing.T) {
    test.ForEachDat(t, test.ZipDats, runValidateSizeTest)
}

func TestValidateSizeDir(t *testing.T) {
    test.ForEachDat(t, test.DirDats, runValidateSizeTest)
}
