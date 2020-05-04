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
package gorom

import (
    "testing"
    "os"
    "fmt"
    "gorom/test"
    "gorom/checksum"
    "github.com/klauspost/compress/zip"
)

///////////////////////////////////////////////////////////////////////////////
// Database Tests
///////////////////////////////////////////////////////////////////////////////

func runDatabaseTest(t *testing.T, df *test.DatFile) {
    defer test.Chdir(t, df.DataPath)()

    defer os.Remove(".gorom.db")
    rdb, err := OpenRomDB("")
    if err != nil {
        test.Fail(t, err)
    }
    defer rdb.Close()

    err = rdb.Scan(1, nil, nil)
    if err != nil {
        test.Fail(t, err)
    }

    for machName, machine := range df.Machines {
        for romName, rom := range machine.Roms {
            sha1, ok := checksum.NewSha1String(rom.Sha1)
            if !ok {
                test.Fail(t, "invalid sha1")
            }
            entry, err := rdb.Lookup(sha1)
            if err != nil {
                test.Fail(t, err)
            }
            if entry == nil {
                test.Fail(t, "sha1 checksum not found")
            }
            if entry.MachPath != df.MachPath(machName) || entry.RomPath != romName {
                test.Fail(t, "database entry does not match")
            }
        }
    }
}

func TestDatabaseZip(t *testing.T) {
    test.ForEachDat(t, test.ZipDats, runDatabaseTest)
}

func TestDatabaseDir(t *testing.T) {
    test.ForEachDat(t, test.DirDats, runDatabaseTest)
}

///////////////////////////////////////////////////////////////////////////////
// Datfile Tests
///////////////////////////////////////////////////////////////////////////////

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

///////////////////////////////////////////////////////////////////////////////
// Validate Tests
///////////////////////////////////////////////////////////////////////////////

func createMachine(machName string, testMach *test.Machine) *Machine {
    machine := &Machine{ machName, testMach.Description, testMach.Year,
        testMach.Manufacturer, testMach.Category, nil, "", false }

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
    rdb, err := OpenRomDB(".")
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
        if machine.Path != df.MachPath(machName) || machine.IsDir != df.IsDir {
            test.Fail(t, "machine attribute mismatch")
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
        if machine.Path != df.MachPath(machName) || machine.IsDir != df.IsDir {
            test.Fail(t, "machine attribute mismatch")
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

///////////////////////////////////////////////////////////////////////////////
// ROM I/O  Tests
///////////////////////////////////////////////////////////////////////////////

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

///////////////////////////////////////////////////////////////////////////////
// Misc Tests
///////////////////////////////////////////////////////////////////////////////

func scanZipTest(t *testing.T, machPath string, machine *test.Machine) {
    count := 0
    err := ScanZip(machPath, func (fh *zip.File) (bool, error) {
        rom, ok := machine.Roms[fh.Name]
        if !ok {
            test.Fail(t, "unexpected file")
        }
        if int64(fh.UncompressedSize64) != rom.Size {
            test.Fail(t, "size mismatch")
        }
        count++
        return true, nil
    })
    if err != nil {
        test.Fail(t, err)
    }
    if count != len(machine.Roms) {
        test.Fail(t, "file count mismatch")
    }
}

func runScanZipTest(t *testing.T, df *test.DatFile) {
    defer test.Chdir(t, df.DataPath)()
    for machName, machine := range df.Machines {
        scanZipTest(t, df.MachPath(machName), &machine)
    }
}

func TestScanZip(t *testing.T) {
    test.ForEachDat(t, test.ZipDats, runScanZipTest)
}

func TestHumanizePow2(t *testing.T) {
    nums := []int64{ 0, 1023, 1024, 1025, 1024*1024, 1024*1024-1, 1024*1024*1024, 1024*1024*1024-1}
    strs := []string{ "0 B", "1023 B", "1 KiB", "1.001 KiB", "1 MiB", "1023.999 KiB", "1 GiB", "1 GiB" }

    for i := 0; i < len(nums); i++ {
        s := HumanizePow2(nums[i])
        if s != strs[i] {
            test.Fail(t, "unexpected string " + strs[i] + " " + s)
        }
    }
}

func TestStringBiMap(t *testing.T) {
    sbm := NewStringBiMap()

    if !sbm.Set("k1", "v1") {
        test.Fail(t, "Set method")
    }

    if !sbm.Set("k2", "v2") {
        test.Fail(t, "Set method")
    }

    if sbm.Set("k2", "v3") {
        test.Fail(t, "Set method")
    }

    if sbm.Set("k3", "v2") {
        test.Fail(t, "Set method")
    }

    if v, ok := sbm.Get("k1"); !ok || v != "v1" {
        test.Fail(t, "Get method")
    }

    if v, ok := sbm.Get("k2"); !ok || v != "v2" {
        test.Fail(t, "Get method")
    }

    if _, ok := sbm.Get("k3");  ok {
        test.Fail(t, "Get method")
    }

    if k, ok := sbm.GetValue("v1"); !ok || k != "k1" {
        test.Fail(t, "GetValue method")
    }

    if k, ok := sbm.GetValue("v2"); !ok || k != "k2" {
        test.Fail(t, "GetValue method")
    }

    if _, ok := sbm.GetValue("v3"); ok {
        test.Fail(t, "GetValue method")
    }

    if len(sbm.Keys()) != 2 {
        test.Fail(t, "Keys method")
    }

    if len(sbm.Values()) != 2 {
        test.Fail(t, "Values method")
    }

    if !sbm.Delete("k1", "v1") {
        test.Fail(t, "Delete method")
    }

    if sbm.Delete("k1", "v1") {
        test.Fail(t, "Delete method")
    }

    if !sbm.Delete("k2", "v2") {
        test.Fail(t, "Delete method")
    }

    if sbm.Delete("k2", "v2") {
        test.Fail(t, "Delete method")
    }
}
