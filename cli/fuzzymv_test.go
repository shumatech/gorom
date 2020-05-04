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
    "gorom/test"
    "os"
    "sort"
    "strings"
    "testing"
)

func sortFilter(out *[]byte) {
    s := strings.Split(string(*out),"\n")

    sort.Slice(s, func(i, j int) bool {
        return s[i] < s[j]
    })

    *out = []byte(strings.Join(s, "\n"))
}

func TestFuzzyMvSnaps(t *testing.T) {
    test.RunDiffFilterTest(t, "names", "fuzzymv/snaps.out", func() error {
        options = Options{}
        
        defer os.RemoveAll("roms")
        test.Unzip(t, "roms.zip")

        defer os.RemoveAll("snaps")
        test.Unzip(t, "snaps.zip")

        return fuzzymv("roms", "snaps")
    }, sortFilter)
}
