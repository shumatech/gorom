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
    "gorom/test"
)

func TestDir2DatZip(t *testing.T) {
    test.RunDiffTest(t, "roms/zip", "dir2dat/zip.out", func() error {
        options = Options{}
        options.Dir2Dat.Name = "ziproms"
        options.Dir2Dat.Desc = "Zip_ROMs"
        return dir2dat([]string{})
    })
}

func TestDir2DatDir(t *testing.T) {
    test.RunDiffTest(t, "roms/dir", "dir2dat/dir.out", func() error {
        options = Options{}
        options.Dir2Dat.Name = "dirroms"
        options.Dir2Dat.Desc = "Dir_ROMs"
        return dir2dat([]string{})
    })
}

