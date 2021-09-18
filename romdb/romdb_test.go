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
package romdb

import (
    "testing"
    "os"
    "gorom/test"
    "gorom/checksum"
)

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
