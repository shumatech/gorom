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

func TestFltDatBrazil(t *testing.T) {
    test.RunDiffTest(t, "", "fltdat/brazil.out", func() error {
        options = Options{}
        options.FltDat.Name = []string{"Brazil"}
        return fltdat("dats/atari.dat.gz")
    })
}

func TestFltDatCapcom(t *testing.T) {
    test.RunDiffTest(t, "", "fltdat/capcom.out", func() error {
        options = Options{}
        options.FltDat.Year = []string{"199[0-2]"}
        options.FltDat.Manu = []string{"Capcom"}
        return fltdat("dats/mame.xml.gz")
    })
}

func TestFltDatStreet(t *testing.T) {
    test.RunDiffTest(t, "", "fltdat/street.out", func() error {
        options = Options{}
        options.FltDat.Desc = []string{"^Street"}
        return fltdat("dats/mame.xml.gz")
    })
}

func TestFltDatSnk(t *testing.T) {
    test.RunDiffTest(t, "", "fltdat/snk.out", func() error {
        options = Options{}
        options.FltDat.Manu = []string{"(?i)snk"}
        return fltdat("dats/mame.xml.gz")
    })
}