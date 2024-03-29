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
package checksum

import (
    "os"
    "io"
    "hash/crc32"
    "crypto/sha1"
    "encoding/xml"
    "encoding/hex"
)

///////////////////////////////////////////////////////////////////////////////
// Checksum types
///////////////////////////////////////////////////////////////////////////////

type Sha1 [sha1.Size]byte

type Crc32 [crc32.Size]byte

func NewSha1String(hexstr string) (sum Sha1, ok bool) {
    s, err := hex.DecodeString(hexstr)
    if err != nil || len(s) != sha1.Size {
        return
    }

    ok = true
    copy(sum[:], s)
    return
}

func NewCrc32String(hexstr string) (sum Crc32, ok bool) {
    s, err := hex.DecodeString(hexstr)
    if err != nil || len(s) != crc32.Size {
        return
    }

    ok = true
    copy(sum[:], s)
    return
}

///////////////////////////////////////////////////////////////////////////////
// XML marshaling
///////////////////////////////////////////////////////////////////////////////

func (sha1 *Sha1) UnmarshalXMLAttr(attr xml.Attr) error {
    var value []byte
    var err error
    if value, err = hex.DecodeString(attr.Value); err != nil {
        return err
    }
    copy((*sha1)[:], value)
    return nil
}

func (crc32 *Crc32) UnmarshalXMLAttr(attr xml.Attr) error {
    var value []byte
    var err error
    if value, err = hex.DecodeString(attr.Value); err != nil {
        return err
    }
    copy((*crc32)[:], value)
    return nil
}

///////////////////////////////////////////////////////////////////////////////
// Sha1File - Return the SHA1 checksum for a file
///////////////////////////////////////////////////////////////////////////////

func Sha1File(path string) (Sha1, error) {
    var sum Sha1

    fh, err := os.Open(path)
    if err != nil {
        return Sha1{}, err
    }
    defer fh.Close()

    hash := sha1.New()

    _, err = io.Copy(hash, fh)
    if err != nil {
        return sum, err
    }

    copy(sum[:], hash.Sum(nil))

    return sum, nil
}

///////////////////////////////////////////////////////////////////////////////
// Crc32File - Return the CRC32 hash for a file
///////////////////////////////////////////////////////////////////////////////

func Crc32File(path string) (Crc32, error) {
    var sum Crc32
    fh, err := os.Open(path)
    if err != nil {
        return sum, err
    }
    defer fh.Close()

    hash := crc32.NewIEEE()

    _, err = io.Copy(hash, fh)
    if err != nil {
        return sum, err
    }

    copy(sum[:], hash.Sum(nil))

    return sum, nil
}
