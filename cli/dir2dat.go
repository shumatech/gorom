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
    "os"
    "strings"
    "path"

    "gorom"
    "gorom/checksum"
    "gorom/term"

    "github.com/klauspost/compress/zip"
)

const xmlDeclaration string = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE datafile PUBLIC "-//Logiqx//DTD ROM Management Datafile//EN" "http://www.logiqx.com/Dats/datafile.dtd">
`

const headerTag string = `	<header>
		<name>%s</name>
		<description>%s</description>
		<version></version>
		<author></author>
	</header>
`

const datafileStart string = `<datafile>`
const datafileEnd string = `</datafile>`

const machineStart string = `	<machine name="%s">
		<description>%s</description>
`
const machineEnd string = `	</machine>`

const romTag string = `		<rom name="%s" size="%d" crc="%08x" sha1="%x"/>
`

func machZip(path string) error {
    return checksum.Sha1Zip(path, func(fh *zip.File, sha1 checksum.Sha1) error {
        term.Printf(romTag, gorom.ToDatPath(fh.Name), fh.UncompressedSize64, fh.CRC32, sha1)
        return nil
    })
}

func machDir(dir string) error {
    return gorom.ScanDir(dir, true, func(file os.FileInfo) error {
        path := path.Join(dir, file.Name())
        if file.IsDir() {
            err := machDir(path)
            if err != nil {
                return err
            }
        } else {
            sha1, err := checksum.Sha1File(path)
            if err != nil {
                return err
            }
            crc32, err := checksum.Crc32File(path)
            if err != nil {
                return err
            }
            term.Printf(romTag, gorom.ToDatPath(path), file.Size(), crc32, sha1)
        }
        return nil
    })
}

func dir2dat(machFilter []string) error {
    machMap := make(map[string]bool)
    for _, arg := range machFilter {
        machName := strings.TrimSuffix(arg, path.Ext(path.Base(arg)))
        machName = strings.TrimSuffix(machName, "/")
        machMap[machName] = true
    }

    term.Println(xmlDeclaration)
    term.Println(datafileStart)
    term.Printf(headerTag, options.Dir2Dat.Name, options.Dir2Dat.Desc)

    err := gorom.ScanDir(".", true, func(file os.FileInfo) error {
        name := file.Name()
        machName := strings.TrimSuffix(name, path.Ext(path.Base(name)))

        if len(machMap) > 0 {
            if _, ok := machMap[machName]; !ok {
                return nil
            }
        }

        if file.IsDir() {
            term.Printf(machineStart, machName, machName)
            machDir(name)
            term.Println(machineEnd)
        } else if path.Ext(name) == ".zip" {
            term.Printf(machineStart, machName, machName)
            machZip(name)
            term.Println(machineEnd)
        }

        return nil
    })
    if err != nil {
        return err
    }

    term.Println(datafileEnd)

    return nil
}
