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
    "strings"
    "regexp"
    "gorom/test"
)

func dateFilter(out *[]byte) {
    strs := strings.Split(string(*out),"\n")

    re := regexp.MustCompile(`20\d\d-\d\d-\d\d`)

    for i := range strs {
        idx := re.FindStringIndex(strs[i])
        if idx != nil {
            strs[i] = strs[i][:idx[0]]
        }
    }

    *out = []byte(strings.Join(strs, "\n"))
}

func TestGoRomDB(t *testing.T) {
    test.RunDiffFilterTest(t, "roms/zip", "goromdb/zip.out", func() error {
        defer os.Remove(".gorom.db")

        options = Options{}
        options.GoRomDB.Scan = true
        options.App.NoGo = true
        err := goromdb()
        if err != nil {
            return err
        }

        options = Options{}
        options.GoRomDB.Dump = true
        err = goromdb()
        if err != nil {
            return err
        }

        options = Options{}
        options.GoRomDB.Lookup = test.ZipDats[0].Machines["machine2"].Roms["rom_4.bin"].Sha1
        err = goromdb()
        if err != nil {
            return err
        }

        options = Options{}
        options.GoRomDB.Lookup = "1111111111111111111111111111111111111111"
        err = goromdb()
        if err.Error() != "Checksum not found" {
            if err == nil {
                err = fmt.Errorf("Checksum was found")
            }
            return err
        }

        options = Options{}
        options.GoRomDB.Lookup = "not a checksum"
        err = goromdb()
        if err.Error() != "Invalid checksum" {
            if err == nil {
                err = fmt.Errorf("Checksum NOT invalid")
            }
            return err
        }

        return nil
    }, dateFilter)
}

