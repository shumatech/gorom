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
// CreateSha1 - create a SHA1 from a reader
///////////////////////////////////////////////////////////////////////////////

func CreateSha1(rd io.Reader) (Sha1, error) {
    var sum Sha1

    hash := sha1.New()

    _, err := io.Copy(hash, rd)
    if err != nil {
        return sum, err
    }

    copy(sum[:], hash.Sum(nil))

    return sum, nil
}

///////////////////////////////////////////////////////////////////////////////
// Sha1File - Return the SHA1 checksum for a file
///////////////////////////////////////////////////////////////////////////////

func Sha1File(path string) (Sha1, error) {
    fh, err := os.Open(path)
    if err != nil {
        return Sha1{}, err
    }
    defer fh.Close()

    return CreateSha1(fh)
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

///////////////////////////////////////////////////////////////////////////////
// ChecksumReader - Return both the CRC32 and SHA1 hash for the io.Reader
///////////////////////////////////////////////////////////////////////////////
func ChecksumReader(rd io.Reader) (Crc32, Sha1, error) {
    buffer := make([]byte, 256 * 1024)
    sha1Hash := sha1.New()
    crc32Hash := crc32.NewIEEE()

    var sha1 Sha1
    var crc32 Crc32
    for {
        len, err := rd.Read(buffer)
        if (len > 0) {
            sha1Hash.Write(buffer[:len])
            crc32Hash.Write(buffer[:len])
        }
        if err != nil {
            if err == io.EOF {
                break
            }
            return crc32, sha1, err
        }
    }

    copy(sha1[:], sha1Hash.Sum(nil))
    copy(crc32[:], crc32Hash.Sum(nil))

    return crc32, sha1, nil
}
