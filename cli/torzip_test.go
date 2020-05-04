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
    "fmt"
    "gorom/checksum"
    "gorom/test"
    "os"
    "strings"
    "testing"
)

func fileFilter(out *[]byte) {
    strs := strings.Split(string(*out),"\n")

    for i := range strs {
        idx := strings.Index(strs[i], ":")
        if idx != -1 {
            strs[i] = strs[i][idx:]
        }
    }

    *out = []byte(strings.Join(strs, "\n"))
}

func runTorZipTest(t *testing.T, zip string, expHash string) error {
    tmpfile := test.CopyFileToTemp(t, ".", zip)
    defer os.Remove(tmpfile)

    err := torzipFiles([]string{tmpfile})
    if err != nil {
        test.Fail(t, err)
    }

    actSha1, err := checksum.Sha1File(tmpfile)
    if err != nil {
        test.Fail(t, err)
    }

    expSha1, ok := checksum.NewSha1String(expHash)
    if !ok {
        test.Fail(t, "invalid sha1")
    }
    if actSha1 != expSha1 {
        test.Fail(t, "SHA-1 checksum mismatch: " + fmt.Sprintf("%x", actSha1))
    }

    return nil
}

func TestTorZipRoms(t *testing.T) {
    test.RunDiffFilterTest(t, "", "torzip/roms.out", func() error {
        options = Options{}
        options.App.Verbose = true
        return runTorZipTest(t, "names/roms.zip", "460a530712c0e095a89fa72734d7971b7ec6b99c")
    }, fileFilter)
}

func TestTorZipSnaps(t *testing.T) {
    test.RunDiffFilterTest(t, "", "torzip/snaps.out", func() error {
        options = Options{}
        options.App.Verbose = true
        return runTorZipTest(t, "names/snaps.zip", "187bfe7277c93904573d467c2b5f43a34d6ca498")
    }, fileFilter)
}

func TestTorZipAlready(t *testing.T) {
    test.RunDiffFilterTest(t, "", "torzip/already.out", func() error {
        options = Options{}
        options.App.Verbose = true
        return runTorZipTest(t, "roms/zip/machine1.zip", "99ad47f1d99dd9f754add7f05098353ea3d7554f")
    }, fileFilter)
}
