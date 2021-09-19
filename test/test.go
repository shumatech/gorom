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
package test

import (
    "bytes"
    "gorom/term"
    "io"
    "io/ioutil"
    "os"
    "path"
    "runtime"
    "testing"
    "strings"

    "github.com/andreyvit/diff"
    "github.com/klauspost/compress/zip"
)


var (
    _, b, _, _   = runtime.Caller(0)
    TestDir      = path.Dir(b)
    CreateOutput = false
)



///////////////////////////////////////////////////////////////////////////////
// Test Data
///////////////////////////////////////////////////////////////////////////////

type Rom struct {
    Size int64
    Crc32 string
    Sha1 string
}

type RomMap map[string]Rom

type Machine struct {
    Description string
    Year string
    Manufacturer string
    Category string
    Roms map[string]Rom
}

type MachineMap map[string]Machine

type DatFile struct {
    Name string
    Description string
    Version string
    Author string
    Path string
    DataPath string
    MachExt string
    Machines MachineMap
    MachineSha1 map[string]string
    MachineCrc32 map[string]string
}

var (
    machineMap  = MachineMap {
        "machine1" : {
            "machine1", "", "", "",
            RomMap {
                "rom_1.bin" : { 4096, "c26a1549", "325701a893c1102805329f8af2d8410e40c14c79" },
                "rom_2.bin" : { 4096, "b7426747", "1d19fbe4b8e3b27a6244cff1375ca62629610923" },
            },
        },
        "machine2" : {
            "machine2", "", "", "",
            RomMap {
                "rom_3.bin" : { 4096, "04167f96", "2936ac223eec87c3df372560cd62f76b209d488a" },
                "rom_4.bin" : { 4096, "c506e1b8", "d7ed430be515f9b9400248a7cf6ef53006fd29b0" },
                "rom_5.bin" : { 4096, "4b3d43d8", "ca383f60af75d30d9e33f9b9dd551b8c50f2c454" },
            },
        },
        "machine3" : {
            "machine3", "", "", "",
            RomMap {
                "rom_6.bin" : { 4096, "321f42ee", "4544856e00b9efb13c1d5e6ee52ee29c80316d90" },
                "rom_7.bin" : { 4096, "661dbe11", "4045f6b8da2684e64037dfc3a4589d519638d154" },
                "rom_8.bin" : { 4096, "a063b5c3", "eca357e2c830407b89741f098f507f5d41513f43" },
                "rom_9.bin" : { 4096, "ad119cd7", "9ca412192ff0714760cb9c1f21e73f1f4a693d28" },
            },
        },
    }

    machineZipSha1 = map[string]string {
        "machine1" : "99ad47f1d99dd9f754add7f05098353ea3d7554f",
        "machine2" : "d92f1cb145d5241b9fc95107d627ffe9043dfc7d",
        "machine3" : "44b12a1df5edbd1c80a484096b76ee6e4585a48a",
    }

    machineZipCrc32 = map[string]string {
        "machine1" : "547b2b55",
        "machine2" : "d1799f96",
        "machine3" : "00889ad4",
    }

    ZipDats = []DatFile {
        { "ziproms", "Zip_ROMs", "", "", "dats/zip.dat", "roms/zip", ".zip", machineMap, machineZipSha1, machineZipCrc32},
    }

    DirDats = []DatFile {
        { "dirroms", "Dir_ROMs", "", "", "dats/dir.dat", "roms/dir", "", machineMap, nil, nil },
    }

    HeaderDats = []DatFile {
        { "ziproms", "Zip_ROMs", "", "", "dats/zip.dat", "roms/header", ".zip", machineMap, nil, nil },
    }

    ArchiveDats = []DatFile {
        { "7zroms",  "7z_ROMs",  "", "", "dats/7z.dat",  "roms/7z",  ".7z",  machineMap, nil, nil},
        { "rarroms", "Rar_ROMs", "", "", "dats/rar.dat", "roms/rar", ".rar", machineMap, nil, nil},
        { "tgzroms", "Tgz_ROMs", "", "", "dats/tgz.dat", "roms/tgz", ".tgz", machineMap, nil, nil},
    }
)

func (df *DatFile)MachPath(machName string) string {
    return machName + df.MachExt
}

func (df *DatFile)IsDir() bool {
    return df.MachExt == ""
}

func ForEachDat(t *testing.T, dats []DatFile, callback func(t *testing.T, dat *DatFile)) {
    for _, dat := range dats {
        callback(t, &dat)
    }
}

///////////////////////////////////////////////////////////////////////////////
// Functions
///////////////////////////////////////////////////////////////////////////////

func Fail(t *testing.T, a interface{}) {
    t.Fatalf("\n%v", a)
}

func CopyFileToTemp(t *testing.T, dst string, src string) string {
    sf, err := os.Open(src)
    if err != nil {
        Fail(t, err)
    }
    defer sf.Close()

    tf, err := ioutil.TempFile(dst, "gorom*")
    if err != nil {
        Fail(t, err)
    }
    defer tf.Close()

    _, err = io.Copy(tf, sf)
    if err != nil {
        os.Remove(tf.Name())
        Fail(t, err)
    }

    return tf.Name()
}

func CopyDir(t *testing.T, dst string, src string) {
    fh, err := os.Open(src)
    if err != nil {
        Fail(t, err)
    }

    flist, err := fh.Readdir(0)
    if err != nil {
        Fail(t, err)
    }

    for _, info := range flist {
        srcPath := path.Join(src, info.Name())
        dstPath := path.Join(dst, info.Name())
        if info.IsDir() {
            err := os.Mkdir(dstPath, info.Mode())
            if err != nil {
                Fail(t, err)
            }
            CopyDir(t, dstPath, srcPath)
        } else {
            err := os.Link(srcPath, dstPath)
            if err != nil {
                Fail(t, err)
            }
        }
    }
}

func CopyDirToTemp(t *testing.T, dst string, src string) string {
    tmpdir, err := ioutil.TempDir(dst, "gorom*")
    if err != nil {
        Fail(t, err)
    }

    CopyDir(t, tmpdir, src)

    return tmpdir
}

func Chdir(t *testing.T, dir string) func() {
    wd, err := os.Getwd()
    if err != nil {
        Fail(t, err)
    }

    err = os.Chdir(path.Join(TestDir, dir))
    if err != nil {
        Fail(t, err)
    }

    return func() { os.Chdir(wd) }
}

func Unzip(t *testing.T, zipPath string) {
    rc, err := zip.OpenReader(zipPath)
    if err != nil {
        Fail(t, err)
    }
    defer rc.Close()

    for _, fh := range rc.File {
        dir := path.Dir(fh.Name)
        if dir != "." {
            _, err := os.Stat(dir)
            if err != nil && !os.IsNotExist(err) {
                Fail(t, err)
            }
            if os.IsNotExist(err) {
                err = os.MkdirAll(dir, 0755)
                if err != nil {
                    Fail(t, err)
                }
            }
        }

        if !strings.HasSuffix(fh.Name, "/") {
            dst, err := os.OpenFile(fh.Name, os.O_WRONLY|os.O_CREATE, fh.Mode())
            if err != nil {
                Fail(t, err)
            }

            src, err := fh.Open()
            if err != nil {
                dst.Close()
                Fail(t, err)
            }

            _, err = io.Copy(dst, src)
            dst.Close()
            src.Close()
            if err != nil {
                Fail(t, err)
            }
        }
    }
}

func RunDiffFilterTest(t *testing.T, chdir string, expectPath string, testFunc func() error, filterFunc func(out *[]byte)) {
    var file *os.File
    var err error
    var expectOut []byte

    if CreateOutput {
        term.Println("Creating test output", expectPath)
        file, err = os.OpenFile(path.Join(TestDir, expectPath), os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0644)
        if err != nil {
            Fail(t, err)
        }
    } else {
        file, err = os.Open(path.Join(TestDir, expectPath))
        if err != nil {
            Fail(t, err)
        }

        expectOut, err = ioutil.ReadAll(file)
        if err != nil {
            Fail(t, err)
        }
    }
    defer file.Close()

    defer Chdir(t, chdir)()

    term.CaptureStart()
    err = testFunc()
    actualOut := term.CaptureStop()
    if err != nil {
        Fail(t, err)
    }

    if filterFunc != nil {
        filterFunc(&actualOut)
    }

    if CreateOutput {
        _, err = file.Write(actualOut)
        if err != nil {
            Fail(t, err)
        }
        term.Println(string(actualOut))
    } else {
        if !bytes.Equal(expectOut, actualOut) {
            Fail(t, diff.LineDiff(string(expectOut), string(actualOut)))
        }
    }
}

func RunDiffTest(t *testing.T, chdir string, expectPath string, testFunc func() error) {
    RunDiffFilterTest(t, chdir, expectPath, testFunc, nil)
}
