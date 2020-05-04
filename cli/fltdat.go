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
    "bytes"
    "compress/gzip"
    "encoding/xml"
    "io"
    "log"
    "os"
    "path"
    "regexp"

    "gorom"
    "gorom/term"
)

func newRegExpList(exprList []string) []*regexp.Regexp {
    var reList []*regexp.Regexp
    for _, expr := range exprList {
        re, err := regexp.Compile(expr)
        if err != nil {
            log.Printf("%s: %s", expr, err)
        } else {
            reList = append(reList, re)
        }
    }
    return reList
}

func findRegExp(str string, regExpList []*regexp.Regexp) bool {
    if len(regExpList) == 0 {
        return true
    }
    for _, re := range regExpList {
        if re.FindString(str) != "" {
            return true
        }
    }
    return false
}

func fltdat(datFile string) error {
    nameList := newRegExpList(options.FltDat.Name)
    descList := newRegExpList(options.FltDat.Desc)
    manuList := newRegExpList(options.FltDat.Manu)
    yearList := newRegExpList(options.FltDat.Year)
    catList := newRegExpList(options.FltDat.Cat)

    var rd io.Reader
    if datFile == "" {
        rd = os.Stdin
    } else {
        df, err := os.Open(datFile)
        if err != nil {
            return err
        }
        defer df.Close()
        rd = df
    }

    var buffer bytes.Buffer
    if path.Ext(datFile) == ".gz" {
        gz, err := gzip.NewReader(rd)
        if err != nil {
            return err
        }
        defer gz.Close()

        buffer.ReadFrom(gz)
    } else {
        buffer.ReadFrom(rd)
    }
    bufBytes := buffer.Bytes()
    decoder := xml.NewDecoder(bytes.NewReader(bufBytes))
    start := int64(0)
    for {
        tok, _ := decoder.Token()
        if tok == nil {
            break
        }

        switch v := tok.(type) {
        case xml.StartElement:
            filter := false
            if v.Name.Local == "machine" || v.Name.Local == "game" {
                var machine gorom.Machine
                decoder.DecodeElement(&machine, &v)

                filter = !findRegExp(machine.Name, nameList) ||
                         !findRegExp(machine.Description, descList) ||
                         !findRegExp(machine.Manufacturer, manuList) ||
                         !findRegExp(machine.Year, yearList) ||
                         !findRegExp(machine.Category, catList);
                if options.FltDat.Invert {
                    filter = !filter
                }
            }

            end := decoder.InputOffset()
            if !filter {
                term.Print(string(bufBytes[start:end]))
            }
            start = end

        case xml.EndElement:
            end := decoder.InputOffset()
            term.Print(string(bufBytes[start:end]))
            start = end
        }
    }
    term.Print(string(bufBytes[start:]))

    return nil
}
