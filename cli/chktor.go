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
    "io"
    "bytes"
    "strings"
    "crypto/sha1"
    "path"

    "gorom"
    "gorom/term"
)

func joinPath(path []string) string {
    return strings.Join(path, "/")
}

func verifyFiles(files []gorom.TorrentFile) (missing int, badSize int) {
    term.Println("\nVerifying files...")
    for _, file := range files {
        path := joinPath(file.Path)

        info, err := os.Stat(path)
        if err == nil {
            if info.Size() != file.Length {
                term.Printf("%s : %s\n", path, term.Red("BAD SIZE (%d != %d)", info.Size(), file.Length))
                badSize++
            } else if !options.App.NoOk {
                term.Printf("%s : %s\n", path, term.Green("OK"))
            }
        } else {
            term.Printf("%s : %s\n", path, term.Yellow("MISSING"))
            missing++
        }
    }

    return
}

func fillPiece(piece *bytes.Buffer, pieceLen uint32, files []gorom.TorrentFile, index *int, offset *int64) error {
    left := int64(pieceLen)

    for *index < len(files) && left > 0 {
        path := joinPath(files[*index].Path)

        fh, err := os.Open(path)
        if err != nil {
            return err
        }

        info, err := fh.Stat()
        if err != nil {
            return err
        }

        size := info.Size()

        if *offset > 0 {
            fh.Seek(*offset, 0)
            size -= *offset
        }

        if size > left {
            _, err = io.CopyN(piece, fh, left)
        } else {
            _, err = io.Copy(piece, fh)
        }

        fh.Close()

        if err != nil {
            return err
        }

        if size > left {
            *offset += left
            break
        }

        (*index)++
        *offset = 0
        left -= size
    }

    return nil
}

func extrasDir(dir string, fileSet gorom.StringSet, extras *int) error {
    return gorom.ScanDir(dir, true, func(file os.FileInfo) error {
        path := path.Join(dir, file.Name())
        if file.IsDir() {
            err := extrasDir(path, fileSet, extras)
            if err != nil {
                return err
            }
        } else {
            if !fileSet.IsSet(path) {
                term.Printf("%s : %s\n", path, term.Blue("EXTRA"))
                (*extras)++
            }
        }
        return nil
    })
}

func findExtras(files []gorom.TorrentFile) (int, error) {
    term.Println("\nFinding extra files...")

    fileSet := gorom.NewStringSet()

    for _, file := range files {
        path := joinPath(file.Path)
        fileSet.Set(path)
    }

    extras := 0
    err := extrasDir(".", fileSet, &extras)

    return extras, err
}

func checksumPiece(piece *bytes.Buffer, checksum []byte) (bool, error) {
    hash := sha1.New()

    _, err := io.Copy(hash, piece)
    if err != nil {
        return false, err
    }

    return bytes.Equal(hash.Sum(nil), checksum), nil
}

func validPieces(info *gorom.TorrentInfo) (bool, error) {
    term.Println("\nValidating pieces...")

    buffer := make([]byte, 0, info.PieceLength)
    piece := bytes.NewBuffer(buffer)

    var index int
    var offset int64

    checksums := []byte(info.Pieces)

    pieceCount := len(info.Pieces) / sha1.Size
    pieceNum := 0
    for ; pieceNum < pieceCount; pieceNum++ {
        gorom.Progressf("%d/%d", pieceNum + 1, pieceCount)
        prevIndex := index
        err := fillPiece(piece, info.PieceLength, info.Files, &index, &offset)
        if err != nil {
            return false, err
        }

        checksumOfs := pieceNum * sha1.Size
        ok, err := checksumPiece(piece, checksums[checksumOfs:checksumOfs + sha1.Size])
        if err != nil {
            return false, err
        }

        if !ok {
            term.Println(term.Red("\nChecksum error in piece %d", pieceNum))
            term.Println("Possible files...")
            for i := prevIndex; i < index || (offset != 0 && i <= index); i++ {
                path := joinPath(info.Files[i].Path)
                term.Println(path)
            }
            return false, nil
        }
    }

    gorom.Progressf("")

    if (pieceNum == pieceCount) {
        term.Println(term.Green("OK"))
    }

    return true, nil
}

func chktor(path string) (bool, error) {
    torrent, err := gorom.ParseTorrent(path)
    if err != nil {
        return false, err
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

    missing, badSize := verifyFiles(torrent.Info.Files)

    var extras int
    if !options.App.NoExtra {
        extras, err = findExtras(torrent.Info.Files)
        if err != nil {
            return false, err
        }
    }

    term.Println("\nTorrent Stats")
    term.Println("  Missing  :", missing)
    term.Println("  Bad Size :", badSize)
    if !options.App.NoExtra {
        term.Println("  Extras   :", extras)
    }

    if missing != 0 || badSize != 0 {
        if !options.ChkTor.NoValid {
            term.Println("\nSkipping piece validation due to file errors")
        }
        return false, nil
    }

    if !options.ChkTor.NoValid {
        ok, err := validPieces(&torrent.Info)
        if err != nil {
            return false, err
        }
        if !ok {
            return false, nil
        }
    }
    return true, nil
}
