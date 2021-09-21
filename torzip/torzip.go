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
package torzip

import (
    "bufio"
    "bytes"
    "encoding/binary"
    "encoding/hex"
    "fmt"
    "hash"
    "hash/crc32"
    "io"
    "os"
    "sort"
    "strings"

    "gorom/torzip/zlib"
)

///////////////////////////////////////////////////////////////////////////////
// Local constants
///////////////////////////////////////////////////////////////////////////////

const (
    zipVersion             = 20
    zip64Version           = 45

    localFileSig           = 0x04034b50
    centralDirSig          = 0x02014b50
    endCentralDir64Sig     = 0x06064b50
    endCentralDir64LocSig  = 0x07064b50
    endCentralDirSig       = 0x06054b50

    localFileLen           = 30
    centralDirLen          = 46
    endCentralDir64Len     = 56
    endCentralDir64LocLen  = 20
    endCentralDirLen       = 22

    extraFieldId           = 1
    extraFieldLen          = 20

    generalPurposeFlag     = 2           // max compression
    compressionMethod      = 8           // deflate
    lastModTime            = 48128       // 11:32PM
    lastModDate            = 8600        // 12/24/1996
    versionMadeBy          = 0           // FAT/FAT32

    commentLength          = 22          // TORRENTZIPPED-XXXXXXXX

    uint16max              = 0x0ffff
    uint32max              = 0x0ffffffff

    bufferSize             = 1024*1024
)

///////////////////////////////////////////////////////////////////////////////
// Types
///////////////////////////////////////////////////////////////////////////////

type file struct {
    tzw        *Writer  // parent writer
    name       string   // path of the file
    size       int64    // uncompressed file size
    ofs        int64    // offset of file's header
    compSize   int64    // compressed file size
    hlen       int      // file header length
    crc32      uint32   // CRC-32 of uncompressed file data
    raw        bool     // raw file with compressed data
    index      int      // create order index
    sortName   string   // file name for sorting
}

type Writer struct {
    // Writer chain
    // write -> mw +-> ucw -> zw -> zcw -> bf -> ws
    //             +-> crc32
    mw       io.Writer        // split uncompressed writes between crc32 and zlib
    crc32    hash.Hash32      // CRC-32 hash of uncompressed data
    ucw      *countWriter     // count of uncompressed writes
    zw       *zlib.Writer     // zlib compression
    zcw      *countWriter     // count of compressed writes
    bf       *bufio.Writer    // writer to buffer zlib writes
    ws       io.WriteSeeker   // client writer provided on open

    files    []*file          // files in the zip
    next     int              // next index in iteration
}

///////////////////////////////////////////////////////////////////////////////
// Public API
///////////////////////////////////////////////////////////////////////////////

// This API is designed to be maximally I/O efficient by enforcing that files
// within the zip are written in the order mandated by the TorrentZip
// specification. This allows the API to create the zip file without having to
// write the compressed data to a temporary file and copying it back out. The
// limitation is that the API requires a seekable writer so that it can seek
// backwards to fix up the file header metadata once the data is compressed.
//
// How to use the API
// 1. Create a new Writer with a WriteSeeker to write the zip to
//     fh, err := os.Create("test.zip")
//     tzw, err := torzip.NewWriter(fh)
// 2. Create all files by name
//     for _, file := range files {
//         tzw.Create(file.name) etc.
//     }
// 3. Open and write files using API mandated order
//     for index := tzw.First(); index >= 0; file = index.Next() {
//         file := files[index]
//         wr, err := tzf.Open(file.size)
//         _, err = wr.Write(file.data)
//         wr.Close()
//     }
// 4. Close the Writer
//     tzw.Close()

// NewWriter - create a new TorrentZip writer to the given WriteSeeker
func NewWriter(ws io.WriteSeeker) (*Writer, error) {
    bf := bufio.NewWriterSize(ws, bufferSize)
    zcw := &countWriter{wr:bf}
    zw, err := zlib.NewWriterLevel(zcw, 9)
    if err != nil {
        return nil, err
    }
    ucw := &countWriter{wr:zw}
    crc32 := crc32.NewIEEE()
    mw := io.MultiWriter(ucw, crc32)

    return &Writer{mw:mw, crc32:crc32, ucw:ucw, zw:zw, zcw:zcw, bf:bf, ws:ws}, nil
}

// Create - create a new file in the Zip
func (tzw *Writer) Create(name string) error {
    if tzw.next != 0 {
        return fmt.Errorf("create after write")
    }

    tzf := &file{ name:name, tzw:tzw, index:len(tzw.files) }
    tzw.files = append(tzw.files, tzf)

    return nil
}

// Close - close the writer and write the central directory
func (tzw *Writer) Close() error {
    if tzw.next != len(tzw.files) {
        return fmt.Errorf("not all files written")
    }

    // To write the central directory, remove zlib from the writer chain
    // write -> mw +-> ucw -> bf -> ws
    //             +-> crc32
    tzw.ucw.wr = tzw.bf
    tzw.ucw.count = 0

    start := tzw.zcw.count
    for _, tzf := range tzw.files {
        tzf.writeCentralDir(tzw.mw)
    }
    size := tzw.ucw.count

    tzw.writeEndCentralDir(tzw.bf, start, size)

    tzw.bf.Flush()

    return nil
}

func (tzf *file) Write(p []byte) (int, error) {
    if tzf.raw {
        // Raw writer bypasses the CRC-32 and zlib writers
        return tzf.tzw.zcw.Write(p)
    } else {
        return tzf.tzw.mw.Write(p)
    }
}

func (tzf *file) Close() error {
    tzw := tzf.tzw

    if !tzf.raw && tzw.ucw.count != tzf.size {
        return fmt.Errorf("file size mismatch")
    }

    // Make sure all compressed data is pushed through
    // before we back up and fix up the file header
    if !tzf.raw {
        tzw.zw.Reset()
    }
    tzw.bf.Flush()

    if !tzf.raw {
        tzf.crc32 = tzw.crc32.Sum32()
        tzw.crc32.Reset()
        tzw.ucw.count = 0
    }
    tzf.compSize = tzw.zcw.count - tzf.ofs - int64(tzf.hlen)
    tzf.writeLocalFile(tzw.ws)

    return nil
}

func (tzw *Writer) Open(size int64) (io.WriteCloser, error) {
    if tzw.next == 0 {
        return nil, fmt.Errorf("no file selected")
    }

    tzf := tzw.files[tzw.next - 1]
    tzf.size = size
    tzf.hlen = localFileLen + len(tzf.name)
    if size >= uint32max {
        tzf.hlen += extraFieldLen
    }
    tzf.ofs = tzw.zcw.count

    // Write a dummy local file header for now
    _, err := tzw.zcw.Write(make([]byte, tzf.hlen))
    if err != nil {
        return nil, err
    }

    return tzf, nil
}

// OpenRaw - open a raw file in the Zip that only accepts compressed data
func (tzw *Writer) OpenRaw(size int64, crc32 uint32) (io.WriteCloser, error) {
    wc, err := tzw.Open(size)
    if err != nil {
        return nil, err
    }

    tzf := wc.(*file)
    tzf.crc32 = crc32
    tzf.raw = true

    return wc, nil
}

// First - first file to open and write.  Returns the index of the creation order.
func (tzw *Writer) First() int {
    if tzw.next != 0 || len(tzw.files) == 0 {
        return -1
    }

    // Create lower case names for sorting
    for _, file := range tzw.files {
        file.sortName = strings.ToLower(file.name)
    }

    // Sort the files in lower case order
    sort.Slice(tzw.files, func(i, j int) bool {
        return tzw.files[i].sortName <  tzw.files[j].sortName
    })

    // Clean redundant empty directories
    clean := []*file{}
    for i := 0; i < len(tzw.files) - 1; i++ {
        // Redundant directories end with a / and have a subsequent file start with the same name
        sortName := tzw.files[i].sortName
        if !strings.HasSuffix(sortName, "/") ||
            !strings.HasPrefix(tzw.files[i + 1].sortName, sortName) {
            clean = append(clean, tzw.files[i])
        }
    }
    clean = append(clean, tzw.files[len(tzw.files)-1])
    tzw.files = clean

    tzw.next++

    return tzw.files[0].index
}

// First - next file to open and write.  Returns the index of the creation order.
func (tzw *Writer) Next() int {
    if tzw.next == 0 || tzw.next == len(tzw.files) {
        return -1
    }

    index := tzw.files[tzw.next].index
    tzw.next++

    return index
}

// Determines if a zip file is already in TorrentZip format by verifying the
// comment CRC matches the central directory CRC
func IsTorZip(zip string) (bool, error) {
    f, err := os.Open(zip)
    if err != nil {
        return false, err
    }
    defer f.Close()

    // Torrent zipped files always have a fixed sized end of
    // central director (EOCD) record up to the EOF
    eocdSize := endCentralDirLen + commentLength
    b, err := readOffset(f, int64(-eocdSize), eocdSize)
    if err != nil {
        return false, err
    }

    // Check the signature
    if binary.LittleEndian.Uint32(b[0:]) != endCentralDirSig {
        return false, nil
    }

    // Check for the TorrentZipped comment
    if string(b[22:36]) != "TORRENTZIPPED-" {
        return false, nil
    }

    // Decode the CRC32 in the comment
    sum, err := hex.DecodeString(string(b[36:44]))
    if err != nil {
        return false, err
    }

    // Get the size and offset of the central directory
    size := int64(binary.LittleEndian.Uint32(b[12:]))
    ofs := int64(binary.LittleEndian.Uint32(b[16:]))

    // Handle Zip64 records
    if size == uint32max || ofs == uint32max {
        // Read the Zip64 EOCD locator
        b, err := readOffset(f, int64(-(eocdSize + endCentralDir64LocLen)), endCentralDir64LocLen)
        if err != nil {
            return false, err
        }

        // Check the signature
        if binary.LittleEndian.Uint32(b[0:]) != endCentralDir64LocSig {
            return false, nil
        }

        eocd64ofs := int64(binary.LittleEndian.Uint64(b[8:]))

        // Read the Zip64 EOCD record
        b, err = readOffset(f, eocd64ofs, endCentralDir64LocLen)
        if err != nil {
            return false, err
        }

        // Check the signature
        if binary.LittleEndian.Uint32(b[0:]) != endCentralDir64Sig {
            return false, nil
        }

        if size == uint32max {
            size = int64(binary.LittleEndian.Uint64(b[40:]))
        }
        if ofs == uint32max {
            ofs = int64(binary.LittleEndian.Uint64(b[48:]))
        }
    }

    // CRC32 check the central directory
    if _, err = f.Seek(ofs, io.SeekStart); err != nil {
        return false, err
    }

    crc := crc32.NewIEEE()
    if _, err = io.CopyN(crc, f, size); err != nil {
        return false, err
    }

    return bytes.Equal(sum, crc.Sum(nil)), nil
}

///////////////////////////////////////////////////////////////////////////////
// Count Writer
///////////////////////////////////////////////////////////////////////////////

type countWriter struct {
    wr    io.Writer
    count int64
}
func (cw *countWriter) Write(p []byte) (int, error) {
    n, err := cw.wr.Write(p)
    cw.count += int64(n)
    return n, err
}

///////////////////////////////////////////////////////////////////////////////
// Buffer
///////////////////////////////////////////////////////////////////////////////

type buffer struct {
    data []byte
    end []byte
}

func newBuffer(n int) *buffer {
    d := make([]byte, n)
    return &buffer{d, d}
}

func (b *buffer) write(d []byte) {
    copy(b.end, d)
    b.end = b.end[len(d):]
}

func (b *buffer) uint16(v uint16) {
    binary.LittleEndian.PutUint16(b.end, v)
    b.end = b.end[2:]
}

func (b *buffer) uint32(v uint32) {
    binary.LittleEndian.PutUint32(b.end, v)
    b.end = b.end[4:]
}

func (b *buffer) uint64(v uint64) {
    binary.LittleEndian.PutUint64(b.end, v)
    b.end = b.end[8:]
}

///////////////////////////////////////////////////////////////////////////////
// Helper functions
///////////////////////////////////////////////////////////////////////////////

func (tzf *file) writeLocalFile(ws io.WriteSeeker) error {
    extraLen := 0

    var version uint16 = zipVersion
    var size uint32 = uint32(tzf.size)
    var compSize uint32 = uint32(tzf.compSize)

    if tzf.size > uint32max {
        version = zip64Version
        size = uint32max
        compSize = uint32max
        extraLen = extraFieldLen
    } else if tzf.compSize > uint32max {
        // TODO: corner case where compressed >= 4GB && uncompressed < 4GB
        return fmt.Errorf("compressed data too large")
    }

    b := newBuffer(tzf.hlen)
    b.uint32(localFileSig)
    b.uint16(version)
    b.uint16(generalPurposeFlag)
    b.uint16(compressionMethod)
    b.uint16(lastModTime)
    b.uint16(lastModDate)
    b.uint32(tzf.crc32)
    b.uint32(compSize)
    b.uint32(size)
    b.uint16(uint16(len(tzf.name)))
    b.uint16(uint16(extraLen))

    b.write([]byte(tzf.name))
    if extraLen > 0 {
        b.uint16(extraFieldId)
        b.uint16(uint16(extraLen - 4))
        b.uint64(uint64(tzf.size))
        b.uint64(uint64(tzf.compSize))
    }

    _, err := ws.Seek(tzf.ofs, os.SEEK_SET)
    if err != nil {
        return err
    }

    _, err = ws.Write(b.data)
    if err != nil {
        return err
    }

    _, err = ws.Seek(0, os.SEEK_END)
    //_, err = ws.Seek(0, os.SEEK_END)
    if err != nil {
        return err
    }

    return nil
}

func (tzf *file) writeCentralDir(wr io.Writer) error {
    extraLen := 0

    var version uint16 = zipVersion
    var size uint32 = uint32(tzf.size)
    var compSize uint32 = uint32(tzf.compSize)
    var ofs uint32 = uint32(tzf.ofs)

    if tzf.size > uint32max || tzf.compSize > uint32max || tzf.ofs > uint32max {
        version = zip64Version
        extraLen += 4
    }
    if tzf.size > uint32max {
        extraLen += 8
        size = uint32max
    }
    if tzf.compSize > uint32max {
        extraLen += 8
        compSize = uint32max
    }
    if tzf.ofs > uint32max {
        extraLen += 8
        ofs = uint32max
    }

    b := newBuffer(centralDirLen + len(tzf.name) + extraLen)
    b.uint32(centralDirSig)
    b.uint16(versionMadeBy)
    b.uint16(version)
    b.uint16(generalPurposeFlag)
    b.uint16(compressionMethod)
    b.uint16(lastModTime)
    b.uint16(lastModDate)
    b.uint32(tzf.crc32)
    b.uint32(compSize)
    b.uint32(size)
    b.uint16(uint16(len(tzf.name)))
    b.uint16(uint16(extraLen))
    b.uint16(0)
    b.uint16(0)
    b.uint16(0)
    b.uint32(0)
    b.uint32(ofs)

    b.write([]byte(tzf.name))

    if extraLen > 0 {
        b.uint16(extraFieldId)
        b.uint16(uint16(extraLen - 4))
        if tzf.size > uint32max {
            b.uint64(uint64(tzf.size))
        }
        if tzf.compSize > uint32max {
            b.uint64(uint64(tzf.compSize))
        }
        if tzf.ofs > uint32max {
            b.uint64(uint64(tzf.ofs))
        }
    }

    _, err := wr.Write(b.data)
    if err != nil {
        return err
    }

    return nil
}

func (tzw *Writer) writeEndCentralDir(wr io.Writer, ofs int64, size int64) error {
    records := len(tzw.files)

    if records > uint16max || ofs > uint32max || size > uint32max {
        b := newBuffer(endCentralDir64Len)
        b.uint32(endCentralDir64Sig)
        b.uint64(endCentralDir64Len - 12)
        b.uint16(zip64Version)
        b.uint16(zip64Version)
        b.uint32(0)
        b.uint32(0)
        b.uint64(uint64(records))
        b.uint64(uint64(records))
        b.uint64(uint64(size))
        b.uint64(uint64(ofs))

        _, err := tzw.ws.Write(b.data)
        if err != nil {
            return err
        }

        b = newBuffer(endCentralDir64LocLen)
        b.uint32(endCentralDir64LocSig)
        b.uint32(0)
        b.uint64(uint64(ofs + size))
        b.uint32(1)

        _, err = tzw.ws.Write(b.data)
        if err != nil {
            return err
        }
    }

    if records > uint16max {
        records = uint16max
    }
    if ofs > uint32max {
        ofs = uint32max
    }
    if size > uint32max {
        size = uint32max
    }

    b := newBuffer(endCentralDirLen + commentLength)
    b.uint32(endCentralDirSig)
    b.uint16(0)
    b.uint16(0)
    b.uint16(uint16(records))
    b.uint16(uint16(records))
    b.uint32(uint32(size))
    b.uint32(uint32(ofs))
    b.uint16(commentLength)

    comment := fmt.Sprintf("TORRENTZIPPED-%08X", tzw.crc32.Sum(nil))
    b.write([]byte(comment))

    _, err := wr.Write(b.data)
    if err != nil {
        return err
    }

    return nil
}

func readOffset(f *os.File, ofs int64, size int) ([]byte, error) {
    whence := io.SeekStart
    if ofs < 0 {
        whence = io.SeekEnd
    }
    if _, err := f.Seek(ofs, whence); err != nil {
        return nil, err
    }

    b := make([] byte, size)
    got, err := f.Read(b)
    if err != nil {
        return nil, err
    }

    if got != size {
        return nil, io.ErrUnexpectedEOF
    }
    return b, nil
}
