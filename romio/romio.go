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
package romio

import (
    "fmt"
    "os"
    "io"
    "time"
    "io/ioutil"
    "path"
    "strings"
    "github.com/klauspost/compress/zip"

    "gorom/util"
    "gorom/torzip"
    "gorom/archive"
    "gorom/checksum"
)

const (
    bufferSize = 256*1024
)

///////////////////////////////////////////////////////////////////////////////
// Utility Functions
///////////////////////////////////////////////////////////////////////////////

func IsArchiveExt(machPath string) bool {
    machExt := strings.ToLower(path.Ext(machPath))
    if machExt == ".zip" {
        return true
    }
    for _, ext := range ArchiveReaderExts() {
        if ext == machExt {
            return true
        }
    }
    return false;
}

func MachName(machPath string) string {
    filename := path.Base(machPath)
    if IsArchiveExt(machPath) {
        filename = strings.TrimSuffix(filename, path.Ext(filename))
    }
    return filename
}

///////////////////////////////////////////////////////////////////////////////
// ROM Reader
///////////////////////////////////////////////////////////////////////////////

type RomFile struct {
    Name string
    Size int64
    ModTime time.Time
}

type RomInfo struct {
    name string
    path string
    files []*RomFile
}

type RomReader interface {
    Name() string
    Path() string
    Files() []*RomFile
    Stat(name string) *RomFile
    Open(file *RomFile) (io.ReadCloser, error)
    Close() error
}

func OpenRomReader(machPath string) (RomReader, error) {
    info, err := os.Stat(machPath)
    if err != nil {
        return nil, err
    }

    if info.IsDir() {
        return OpenDirReader(machPath)
    } else if info.Mode().IsRegular() {
        machExt := strings.ToLower(path.Ext(machPath))
        if machExt == ".zip" {
            return OpenZipReader(machPath)
        }
        for _, ext := range ArchiveReaderExts() {
            if ext == machExt {
                return OpenArchiveReader(machPath)
            }
        }
    }

    return nil, nil
}

func OpenRomReaderByName(machName string) (RomReader, error) {
    info, err := os.Stat(machName)
    if err == nil && info.IsDir() {
        return OpenDirReader(machName)
    } else {
        fileName := machName + ".zip"
        info, err = os.Stat(fileName)
        if err == nil && info.Mode().IsRegular() {
            machName = fileName
            return OpenZipReader(machName)
        }
        for _, ext := range ArchiveReaderExts() {
            fileName = machName + ext
            info, err = os.Stat(fileName)
            if err == nil && info.Mode().IsRegular() {
                machName = fileName
                return OpenArchiveReader(machName)
            }
        }
    }

    return nil, nil
}

func (ri *RomInfo) Stat(name string) *RomFile {
    // TODO: put files in map to speed this up?
    for i := range ri.files {
        if ri.files[i].Name == name {
            return ri.files[i]
        }
    }
    return nil
}

func IsDirReader(rr RomReader) bool {
    _, ok := rr.(*DirReader)
    return ok
}

///////////////////////////////////////////////////////////////////////////////
// ROM Writer
///////////////////////////////////////////////////////////////////////////////

type RomWriter interface {
    Name() string
    Path() string
    Create(name string) error
    First() int
    Next() int
    Open(size int64) (io.WriteCloser, error)
    Close() error
    Buffer() []byte
}

func CreateRomWriter(machPath string) (RomWriter, error) {
    var err error
    var rw RomWriter
    if strings.ToLower(path.Ext(machPath)) == ".zip" {
        rw, err = CreateZipWriter(machPath)
    } else {
        rw, err =  CreateDirWriter(machPath)
    }
    return rw, err
}

func CreateRomWriterTemp(dir string, isDir bool) (RomWriter, error) {
    var err error
    var tmpName string
    if isDir {
        tmpName, err = ioutil.TempDir(dir, "gorom*")
        if err != nil {
            return nil, err
        }
        return CreateDirWriter(tmpName)
    } else {
        fh, err := ioutil.TempFile(dir, "gorom*.zip")
        if err != nil {
            return nil, err
        }
        return NewZipWriter(fh)
    }
}

///////////////////////////////////////////////////////////////////////////////
// Directory Reader
///////////////////////////////////////////////////////////////////////////////

type DirReader struct {
    RomInfo
}

func OpenDirReader(machPath string) (*DirReader, error) {
    var dr DirReader
    err := dr.init(machPath)
    return &dr, err
}

func scanDir(base string, dir string, files *[]*RomFile) error {
    return util.ScanDir(path.Join(base, dir), true, func(info os.FileInfo) error {
        if info.IsDir() {
            dir := path.Join(dir, info.Name())
            err := scanDir(base, dir, files)
            if err != nil {
                return err
            }
        } else {
            file := RomFile{
                Name: path.Join(dir, info.Name()),
                Size: info.Size(),
                ModTime: info.ModTime(),
            }
            *files = append(*files, &file)
        }
        return nil
    })
}

func (dr *DirReader) init(machPath string) error {
    files := []*RomFile{}

    err := scanDir(machPath, "", &files)
    if err != nil {
        return err
    }

    dr.path = machPath
    dr.name = MachName(machPath)
    dr.files = files

    return nil
}

func (dr *DirReader) Name() string {
    return dr.name
}

func (dr *DirReader) Path() string {
    return dr.path
}

func (dr *DirReader) Files() []*RomFile {
    return dr.files
}

func (dr *DirReader) Stat(name string) *RomFile {
    return dr.RomInfo.Stat(name)
}

func (dr *DirReader) Open(file *RomFile) (io.ReadCloser, error) {
    rd, err := os.Open(path.Join(dr.path, file.Name))
    if err != nil {
        return nil, err
    }
    return rd, nil
}

func (dr *DirReader) Close() error {
    return nil
}

///////////////////////////////////////////////////////////////////////////////
// Directory Writer
///////////////////////////////////////////////////////////////////////////////

type DirWriter struct {
    RomInfo
    names []string
    next int
    buf []byte
}

func CreateDirWriter(machPath string) (*DirWriter, error) {
    var dw DirWriter
    err := dw.init(machPath)
    return &dw, err
}

func (dw *DirWriter) init(machPath string) error {
    err := os.MkdirAll(machPath, os.ModePerm)
    if err != nil && os.IsNotExist(err) {
        return err
    }

    dw.path = machPath
    dw.name = MachName(machPath)
    dw.buf = make([]byte, bufferSize)

    return nil
}

func (dw *DirWriter) Name() string {
    return dw.name
}

func (dw *DirWriter) Path() string {
    return dw.path
}

func (dw *DirWriter) Buffer() []byte {
    return dw.buf
}

func (dw *DirWriter) Create(name string) error {
    dw.names = append(dw.names, name)
    return nil
}

func (dw *DirWriter) Open(size int64) (io.WriteCloser, error) {
    if dw.next == 0 {
        return nil, fmt.Errorf("no files created")
    }

    name := dw.names[dw.next - 1]
    dir := path.Dir(dw.name)
    if dir != "." {
        path := path.Join(dw.path, dir)
        err := os.MkdirAll(path, os.ModePerm)
        if err != nil && os.IsNotExist(err) {
            return nil, err
        }
    }

    wr, err := os.OpenFile(path.Join(dw.path, name), os.O_WRONLY | os.O_CREATE, 0644)
    if err != nil {
        return nil, err
    }

    return wr, nil
}

func (dw *DirWriter) First() int {
    if dw.next != 0 || len(dw.names) == 0 {
        return -1
    }
    dw.next++
    return 0
}

func (dw *DirWriter) Next() int {
    if dw.next == 0 || dw.next == len(dw.names) {
        return -1
    }
    index := dw.next
    dw.next++
    return index
}

func (dw *DirWriter) Close() error {
    return nil
}

///////////////////////////////////////////////////////////////////////////////
// Zip Reader
///////////////////////////////////////////////////////////////////////////////

type nopReadCloser struct {
    io.Reader
}

func (w nopReadCloser) Close() error {
    return nil
}

type ZipReader struct {
    RomInfo
    dir map[string]*zip.File
    rc *zip.ReadCloser
}

func OpenZipReader(machPath string) (*ZipReader, error) {
    var zr ZipReader
    err := zr.init(machPath)
    return &zr, err
}

func (zr *ZipReader) init(machPath string) error {
    info, err := os.Stat(machPath)
    if err != nil {
        return err
    }
    rc, err := zip.OpenReader(machPath)
    if err != nil {
        return err
    }

    files := []*RomFile{}
    dir := map[string]*zip.File{}
    for _, fh := range rc.File {
        dir[fh.Name] = fh
        file := RomFile{
            Name: fh.Name,
            Size: int64(fh.UncompressedSize64),
            ModTime: info.ModTime(),
        }
        files = append(files, &file)
    }

    zr.path = machPath
    zr.name = MachName(machPath)
    zr.files = files
    zr.dir = dir
    zr.rc = rc

    return nil
}

func (zr *ZipReader) Name() string {
    return zr.name
}

func (zr *ZipReader) Path() string {
    return zr.path
}

func (zr *ZipReader) Files() []*RomFile {
    return zr.files
}

func (zr *ZipReader) Stat(name string) *RomFile {
    return zr.RomInfo.Stat(name)
}

func (zr *ZipReader) Open(file *RomFile) (io.ReadCloser, error) {
    fh, ok := zr.dir[file.Name]
    if !ok {
        return nil, os.ErrNotExist
    }

    rc, err := fh.Open()
    if err != nil {
        return nil, err
    }
    return rc, nil
}

func (zr *ZipReader) Close() error {
    return zr.rc.Close()
}

func (zr *ZipReader) OpenRaw(file *RomFile) (io.ReadCloser, *zip.FileHeader, error) {
    fh, ok := zr.dir[file.Name]
    if !ok {
        return nil, nil, fmt.Errorf("%s: file not found in %s", file.Name, zr.path)
    }

    rc, err := fh.OpenRaw()
    if err != nil {
        return nil, nil, err
    }
    return nopReadCloser{ rc }, &fh.FileHeader, nil
}

///////////////////////////////////////////////////////////////////////////////
// Zip Writer
///////////////////////////////////////////////////////////////////////////////
type ZipWriter struct {
    RomInfo
    tzw *torzip.Writer
    fh *os.File
    buf []byte
}

func CreateZipWriter(machPath string) (*ZipWriter, error) {
    var zw ZipWriter

    fh, err := os.Create(machPath)
    if err != nil {
        return nil, err
    }

    err = zw.init(fh)

    return &zw, err
}

func NewZipWriter(fh *os.File) (*ZipWriter, error) {
    var zw ZipWriter
    err := zw.init(fh)
    return &zw, err
}

func (zw *ZipWriter) init(fh *os.File) error {
    tzw, err := torzip.NewWriter(fh)
    if err != nil {
        return err
    }

    zw.path = fh.Name()
    zw.name = MachName(zw.path)
    zw.fh = fh
    zw.tzw = tzw
    zw.buf = make([]byte, bufferSize)

    return nil
}

func (zw *ZipWriter) Name() string {
    return zw.name
}

func (zw *ZipWriter) Path() string {
    return zw.path
}

func (zw *ZipWriter) Buffer() []byte {
    return zw.buf
}

func (zw *ZipWriter) Create(name string) error {
    return zw.tzw.Create(name)
}

func (zw *ZipWriter) Open(size int64) (io.WriteCloser, error) {
    wr, err := zw.tzw.Open(size)
    if err != nil {
        return nil, err
    }
    return wr, nil
}

func (zw *ZipWriter) First() int {
    return zw.tzw.First()
}

func (zw *ZipWriter) Next() int {
    return zw.tzw.Next()
}

func (zw *ZipWriter) Close() error {
    defer zw.fh.Close()
    return zw.tzw.Close()
}

func (zw *ZipWriter) OpenRaw(fh *zip.FileHeader) (io.WriteCloser, error) {
    return zw.tzw.OpenRaw(int64(fh.UncompressedSize64), fh.CRC32)
}

///////////////////////////////////////////////////////////////////////////////
// Archive Reader
///////////////////////////////////////////////////////////////////////////////

type ArchiveReader struct {
    RomInfo
    dir map[string]int
    rc *archive.Reader
    index int
}

func ArchiveReaderExts() []string {
    return []string{".7z", ".rar", ".tgz", ".gz"}
}

func OpenArchiveReader(machPath string) (*ArchiveReader, error) {
    var ar ArchiveReader
    err := ar.init(machPath)
    return &ar, err
}

func (ar *ArchiveReader) init(machPath string) error {
    info, err := os.Stat(machPath)
    if err != nil {
        return err
    }
    rc, err := archive.OpenReader(machPath)
    if err != nil {
        return err
    }

    files := []*RomFile{}
    dir := map[string]int{}
    index := 0

    for rc.Next() {
        name := rc.Name()
        dir[name] = index
        file := RomFile{
            Name: name,
            Size: rc.Size(),
            ModTime: info.ModTime(),
        }
        files = append(files, &file)
        index++
    }
    if err = rc.Error(); err != nil {
        rc.Close()
        return err
    }

    ar.path = machPath
    ar.name = MachName(machPath)
    ar.files = files
    ar.dir = dir
    ar.rc = rc
    ar.index = index

    return nil
}

func (ar *ArchiveReader) Name() string {
    return ar.name
}

func (ar *ArchiveReader) Path() string {
    return ar.path
}

func (ar *ArchiveReader) Files() []*RomFile {
    return ar.files
}

func (ar *ArchiveReader) Stat(name string) *RomFile {
    return ar.RomInfo.Stat(name)
}

func (ar *ArchiveReader) Open(file *RomFile) (io.ReadCloser, error) {
    index, ok := ar.dir[file.Name]
    if !ok {
        return nil, os.ErrNotExist
    }

    if ar.index > index {
        ar.rc.Reset()
        ar.index = -1
    }

    for ar.index < index {
        ar.rc.Next()
        if err := ar.rc.Error(); err != nil {
            return nil, err
        }
        ar.index++
    }

    return nopReadCloser{ar.rc}, nil
}

func (ar *ArchiveReader) Close() error {
    ar.rc.Close()
    return nil
}

///////////////////////////////////////////////////////////////////////////////
// Copy ROM Algorithm
///////////////////////////////////////////////////////////////////////////////

func CopyRom(writer RomWriter, dstName string, reader RomReader, srcName string) error {
    srcFile := reader.Stat(srcName)
    if srcFile == nil {
        return os.ErrNotExist
    }

    // If both reader and writer are Zips, then do a raw copy so we aren't needlessly
    // decompressing and compressing the data
    zr, ok := reader.(*ZipReader)
    if ok {
        zw, ok := writer.(*ZipWriter)
        if ok {
            rc, fh, err := zr.OpenRaw(srcFile)
            if err != nil {
                return err
            }
            defer rc.Close()

            wr, err := zw.OpenRaw(fh)
            if err != nil {
                return err
            }
            defer wr.Close()

            _, err = io.CopyBuffer(wr, rc, writer.Buffer())
            return err
        }
    }

    rc, err := reader.Open(srcFile)
    if err != nil {
        return err
    }
    defer rc.Close()

    wr, err := writer.Open(srcFile.Size)
    if err != nil {
        return err
    }
    defer wr.Close()

    _, err = io.CopyBuffer(wr, rc, writer.Buffer())
    return err
}

///////////////////////////////////////////////////////////////////////////////
// Checksum ROM
///////////////////////////////////////////////////////////////////////////////

type ChecksumFunc func(name string, size int64, crc32 checksum.Crc32, sha1 checksum.Sha1) error

func ChecksumMach(machPath string, checksumFunc ChecksumFunc) error {
    rr, err := OpenRomReader(machPath)
    if rr == nil || err != nil {
        return err
    }
    defer rr.Close()

    for _, file := range rr.Files() {
        rc, err := rr.Open(file)
        if err != nil {
            return err
        }
        defer rc.Close()

        crc32, sha1, err := checksum.ChecksumReader(rc)
        if err != nil {
            return err
        }

        err = checksumFunc(file.Name, file.Size, crc32, sha1)
        if err != nil {
            return err
        }
    }

    return nil
}