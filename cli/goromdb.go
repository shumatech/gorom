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

    "gorom"
    "gorom/term"
    "gorom/checksum"
)

func dbLookup(rdb *gorom.RomDB, hexstr string) error {
    sum, ok := checksum.NewSha1String(hexstr)
    if !ok {
        return fmt.Errorf("Invalid checksum")
    }
    entry, err := rdb.Lookup(sum)
    if err != nil {
        return err
    }
    if entry == nil {
        return fmt.Errorf("Checksum not found")
    }

    term.Printf("%s %s %x %s\n", entry.MachPath, entry.RomPath, entry.Sum, entry.ModTime)

    return nil
}

func dbScan(rdb *gorom.RomDB) error {
    goLimit := 0
    if options.App.NoGo {
        goLimit = 1
    }
    return rdb.Scan(goLimit, nil, func(machPath string, err error) {
        if err == nil {
            term.Println(machPath)
        } else {
            term.Println(machPath, err)            
        }
    })
}

func dbDump(rdb *gorom.RomDB) error {
    rdb.Dump()
    return nil
}

func goromdb() error {
    rdb, err := gorom.OpenRomDB(".")
    if err != nil {
        return err;
    }
    defer rdb.Close()   

    if options.GoRomDB.Lookup != "" {
        return dbLookup(rdb, options.GoRomDB.Lookup)
    } else if options.GoRomDB.Scan {
        return dbScan(rdb)
    } else if options.GoRomDB.Dump {
        return dbDump(rdb)
    } else {
        return fmt.Errorf("No database operation specified")
    }
}
