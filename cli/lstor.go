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
    "crypto/sha1"

    "gorom"
    "gorom/term"
)

func lstor(path string) error {
    torrent, err := gorom.ParseTorrent(path)
    if err != nil {
        return err
    }

    if !options.App.NoHeader {
        term.Printf("Torrent Name : %s\n", torrent.Info.Name)
        term.Printf("Announce     : %s\n", torrent.Announce)
        term.Printf("Piece Length : %d (%s)\n", torrent.Info.PieceLength, gorom.HumanizePow2(int64(torrent.Info.PieceLength)))
        term.Printf("Pieces       : %d\n", len(torrent.Info.Pieces) / sha1.Size)
        term.Printf("Files        : %d\n", len(torrent.Info.Files))

        var total int64
        for _, file := range torrent.Info.Files {
            total += file.Length
        }
        term.Printf("Total Length : %d (%s)\n", total, gorom.HumanizePow2(total))
    }

    for _, file := range torrent.Info.Files {
        if options.LsTor.NoSize {
            term.Println(joinPath(file.Path))
        } else {
            term.Printf("%s %d (%s)\n", joinPath(file.Path), file.Length, gorom.HumanizePow2(file.Length))
        }
    }
    
    return nil
}
