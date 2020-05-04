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
    "fmt"
)

func runChkTor(t *testing.T, torrent string, expOk bool) error {
    ok, err := chktor(torrent)
    if err != nil {
        return err
    }
    if ok != expOk {
        return fmt.Errorf("test failed: unexpected return value")
    }
    return nil
}

func TestChkTorZip(t *testing.T) {
    test.RunDiffTest(t, "roms/zip", "chktor/zip.out", func() error {
        options = Options{}
        return runChkTor(t, "../../torrents/zip.torrent", true)
    })
}

func TestChkTorDir(t *testing.T) {
    test.RunDiffTest(t, "roms/dir", "chktor/dir.out", func() error {
        options = Options{}
        return runChkTor(t, "../../torrents/dir.torrent", true)
    })
}

func TestChkTorBadZip(t *testing.T) {
    test.RunDiffTest(t, "roms/badzip", "chktor/badzip.out", func() error {
        options = Options{}
        return runChkTor(t, "../../torrents/zip.torrent", false)
    })
}

func TestChkTorBadDir(t *testing.T) {
    test.RunDiffTest(t, "roms/baddir", "chktor/baddir.out", func() error {
        options = Options{}
        return runChkTor(t, "../../torrents/dir.torrent", false)
    })
}

func TestChkTorCorruptZip(t *testing.T) {
    test.RunDiffTest(t, "roms/corruptzip", "chktor/corruptzip.out", func() error {
        options = Options{}
        return runChkTor(t, "../../torrents/zip.torrent", false)
    })
}
