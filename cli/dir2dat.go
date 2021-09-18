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

    "gorom/dat"
    "gorom/util"
    "gorom/checksum"
    "gorom/romio"
    "gorom/term"
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

    err := util.ScanDir(".", true, func(file os.FileInfo) error {
        name := file.Name()
        machName := strings.TrimSuffix(name, path.Ext(path.Base(name)))

        if len(machMap) > 0 {
            if _, ok := machMap[machName]; !ok {
                return nil
            }
        }

        valid := false;
        romio.ChecksumMach(name, func(name string, size int64,
                                     crc32 checksum.Crc32, sha1 checksum.Sha1) error {
            if !valid {
                term.Printf(machineStart, machName, machName)
                valid = true
            }
            term.Printf(romTag, dat.ToDatPath(name), size, crc32, sha1)
            return nil
        })
        if valid {
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