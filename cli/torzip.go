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
    "io"
    "io/ioutil"
    "os"
    "runtime"

    "gorom/term"
    "gorom/torzip"

    "github.com/klauspost/compress/zip"
)

const (
    uint32max = 0xffffffff
    eocdSig = 0x06054b50
    eocdSize = 44
    eocd64Size = 56
    eocd64Sig = 0x06064b50
    eocd64LocSize = 20
    eocd64LocSig = 0x07064b50
)

func torzipFile(path string) error {
    if !options.TorZip.Force {
        ok, err := torzip.IsTorZip(path)
        if err != nil {
            return err
        }
        if ok {
            term.Printf("%s : %s\n", path, term.Yellow("already TorrentZip"))
            return nil
        }
    }

    tf, err := ioutil.TempFile(".", "torzip")
    if err != nil {
        return err
    }

    term.Printf("%s : %s\n", path, term.Cyan("converting to TorrentZip"))

    var zr *zip.ReadCloser
    var zw *torzip.Writer

    // Clean-up in case of error
    defer func() {
        if err != nil {
            if zr != nil {
                zr.Close()
            }
            if zw != nil {
                zw.Close()
            }
            tf.Close()
            err = os.Remove(tf.Name())
            if err != nil {
                term.Println(term.Red(err.Error()))
            }
        }
    }()

    zr, err = zip.OpenReader(path)
    if err != nil {
        return err
    }

    zw, err = torzip.NewWriter(tf)
    if err != nil {
        return err
    }

    for _, zrf := range zr.File {
        err = zw.Create(zrf.Name)
        if err != nil {
            return err
        }
    }

    for index := zw.First(); index >= 0; index = zw.Next() {
        file := zr.File[index]

        if options.App.Verbose {
            term.Printf("%s : add %s (%d bytes)\n", path, file.Name, file.UncompressedSize64)
        }

        var rd io.ReadCloser
        rd, err = file.Open()
        if err != nil {
            return err
        }

        var wr io.WriteCloser
        wr, err = zw.Open(int64(file.UncompressedSize64))
        if err != nil {
            return err
        }

        _, err = io.Copy(wr, rd)
        wr.Close()
        if err != nil {
            return err
        }
    }

    zr.Close()
    zw.Close()
    tf.Close()

    err = os.Remove(path)
    if err != nil {
        term.Println(term.Red(err.Error()))
    } else {
        err = os.Rename(tf.Name(), path)
        if err != nil {
            term.Println(term.Red(err.Error()))
        }
    }

    term.Printf("%s : %s\n", path, term.Green("OK"))

    return nil
}

func torzipGo(path string, ch chan int) {
    err := torzipFile(path)
    if err != nil {
        ch <- 1
        term.Printf("%s: %s\n", path, term.Red(err.Error()))
        return
    }
    ch <- 0
}

func torzipFiles(paths []string) error {
    ch := make(chan int)

    errors := 0
    goCount := 0
    goLimit := 1

    if !options.App.NoGo {
        goLimit = runtime.NumCPU()
    }
    for _, zip := range paths {
        if goCount == goLimit {
            errors += <-ch
        } else {
            goCount++
        }
        go torzipGo(zip, ch)
    }
    for ; goCount > 0; goCount-- {
        errors += <-ch
    }

    if errors > 0 {
        return fmt.Errorf("%d zip file(s) encountered errors", errors)
    }

    return nil
}
