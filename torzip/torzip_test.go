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
package torzip

import (
    "io"
    "io/ioutil"
    "os"
    "strings"
    "testing"

    "github.com/klauspost/compress/zip"

    "gorom/test"
    "gorom/checksum"
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

func torzipTest(t *testing.T, dir string, src string, expSum string) {
    defer test.Chdir(t, dir)()

    tf, err := ioutil.TempFile(".", "torzip*.zip")
    if err != nil {
        test.Fail(t, err)
    }
    defer os.Remove(tf.Name())

    tzw, err := NewWriter(tf)
    if err != nil {
        test.Fail(t, err)
    }

    zr, err := zip.OpenReader(src)
    if err != nil {
        test.Fail(t, err)
    }

    for _, fh := range zr.File {
        err = tzw.Create(fh.Name)
        if err != nil {
            test.Fail(t, err)
        }
    }

    for i := tzw.First(); i >= 0; i = tzw.Next() {
        fh := zr.File[i]

        wr, err := tzw.Open(int64(fh.UncompressedSize64))
        if err != nil {
            test.Fail(t, err)
        }

        rd, err := fh.Open()
        if err != nil {
            test.Fail(t, err)
        }

        _, err = io.Copy(wr, rd)
        if err != nil {
            test.Fail(t, err)
        }

        wr.Close()
    }

    zr.Close()
    tzw.Close()
    tf.Close()

    actSha1, err := checksum.Sha1File(tf.Name())
    if err != nil {
        test.Fail(t, err)
    }

    expSha1,ok := checksum.NewSha1String(expSum)
    if !ok {
        test.Fail(t, "invalid sha1")
    }
    if actSha1 != expSha1 {
        test.Fail(t, "checksum validation failed")
    }
}

func runIsTorZipTest(t *testing.T, datFile *test.DatFile) {
    defer test.Chdir(t, datFile.DataPath)()

    for machName := range datFile.Machines {
        is, err := IsTorZip(datFile.MachPath(machName))
        if err != nil {
            test.Fail(t, err)
        }
        if !is {
            test.Fail(t, "unexpected return")
        }
    }
}

func TestIsTorZip(t *testing.T) {
    test.ForEachDat(t, test.ZipDats, runIsTorZipTest)
}

func TestNotIsTorZip(t *testing.T) {
    defer test.Chdir(t, "names")()

    for _, path := range([]string{"roms.zip", "snaps.zip"}) {
        is, err := IsTorZip(path)
        if err != nil {
            test.Fail(t, err)
        }
        if is {
            test.Fail(t, "unexpected return")
        }
    }
}
