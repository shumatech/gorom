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
package torrent

import (
    "os"
    bencode "github.com/jackpal/bencode-go"
)

type TorrentInfo struct {
    Name            string          `bencode:"name"`
    PieceLength     uint32          `bencode:"piece length"`
    Pieces          string          `bencode:"pieces"`
    Files           []TorrentFile   `bencode:"files"`
}

type TorrentFile struct {
    Length          int64           `bencode:"length"`
    Path            []string        `bencode:"path"`
}

type Torrent struct {
    Announce        string          `bencode:"announce"`
    Info            TorrentInfo     `bencode:"info"`
}

func ParseTorrent(path string) (*Torrent, error) {
    fh, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer fh.Close()

    var torrent Torrent
    err = bencode.Unmarshal(fh, &torrent)
    if err != nil {
        return nil, err
    }

    return &torrent, nil
}