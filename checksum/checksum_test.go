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
package checksum

import (
    "gorom/test"
    "testing"
)

func runSha1FileTest(t *testing.T, df *test.DatFile) {
    defer test.Chdir(t, df.DataPath)()

    for machName := range df.Machines {
        sha1, err := Sha1File(df.MachPath(machName))
        if err != nil {
            test.Fail(t, err)
        }

        if str, ok := df.MachineSha1[machName]; ok {
            expSha1, ok := NewSha1String(str)
            if !ok {
                test.Fail(t, "invalid sha1")
            }
            if sha1 != expSha1 {
                test.Fail(t, "sha1 checksum mismatch")
            }
        } else {
            test.Fail(t, "machine not in sha1 map")
        }
    }
}

func TestSha1File(t *testing.T) {
    test.ForEachDat(t, test.ZipDats, runSha1FileTest)
}

func runCrc32FileTest(t *testing.T, df *test.DatFile) {
    defer test.Chdir(t, df.DataPath)()

    for machName := range df.Machines {
        crc32, err := Crc32File(df.MachPath(machName))
        if err != nil {
            test.Fail(t, err)
        }

        if str, ok := df.MachineCrc32[machName]; ok {
            expCrc32, ok := NewCrc32String(str)
            if !ok {
                test.Fail(t, "invalid crc32")
            }
            if crc32 != expCrc32 {
                test.Fail(t, "crc32 checksum mismatch")
            }
        } else {
            test.Fail(t, "machine not in crc32 map")
        }
    }
}

func TestCrc32File(t *testing.T) {
    test.ForEachDat(t, test.ZipDats, runCrc32FileTest)
}
