//  GoRom - Emulator ROM Management Utilities
//  Copyright (C) 2021 Scott Shumate <scott@shumatech.com>
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
package archive

/*
#cgo pkg-config: libarchive
#include <archive.h>
#include <archive_entry.h>
*/
import "C"

import (
	"errors"
	"io"
	"unsafe"
	"path"
	"time"
	"strings"
)

///////////////////////////////////////////////////////////////////////////////
// Reader
///////////////////////////////////////////////////////////////////////////////

type Reader struct {
	name string
	archive *C.struct_archive
	entry *C.struct_archive_entry
	rc int
}

func OpenReader(name string) (*Reader, error) {
	rd := &Reader{name:name}
	if err := rd.init(); err != nil {
		C.archive_read_free(rd.archive)
		return nil, err;
	}
	return rd, nil
}

func (rd *Reader) init() error {
	rd.rc = C.ARCHIVE_OK
	rd.archive = C.archive_read_new()

    C.archive_read_support_filter_all(rd.archive);
    C.archive_read_support_format_all(rd.archive);

	if C.archive_read_open_filename(rd.archive, C.CString(rd.name),
	                                256 * 1024) != C.ARCHIVE_OK {
		s := C.GoString(C.archive_error_string(rd.archive))
		return errors.New(s)
	}
	return nil
}

func (rd *Reader) Reset() error {
	C.archive_read_free(rd.archive)
	return rd.init()
}

func (rd *Reader) Next() bool {
	rd.rc = int(C.archive_read_next_header(rd.archive, &rd.entry))
	return rd.rc == C.ARCHIVE_OK || rd.rc == C.ARCHIVE_WARN
}

func (rd *Reader) Warn() error {
	if rd.rc != C.ARCHIVE_WARN {
		return nil
	}
	s := C.GoString(C.archive_error_string(rd.archive))
	return errors.New(s)
}

func (rd *Reader) Error() error {
	if rd.rc >= C.ARCHIVE_WARN {
		return nil
	}
	s := C.GoString(C.archive_error_string(rd.archive))
	return errors.New(s)
}

func (rd *Reader) Close() {
	if rd.archive == nil {
		return
	}
	C.archive_read_free(rd.archive)
	rd.archive = nil
}

func (rd *Reader) Path() string {
	if rd.entry == nil {
		return ""
	}
	return C.GoString(C.archive_entry_pathname(rd.entry))
}

func (rd *Reader) Name() string {
	return path.Base(rd.Path())
}

func (rd *Reader) ModTime() time.Time {
	if rd.entry == nil {
		return time.Unix(0, 0)
	}
	sec := int64(C.archive_entry_mtime(rd.entry))
	nsec := int64(C.archive_entry_mtime_nsec(rd.entry))
	return time.Unix(sec, nsec)
}

func (rd *Reader) Size() int64 {
	if rd.entry == nil {
		return 0
	}
	return int64(C.archive_entry_size(rd.entry));
}

func (rd *Reader) Read(buf []byte) (int, error) {
	rc := int(C.archive_read_data(rd.archive, unsafe.Pointer(&buf[0]), C.size_t(len(buf))))
	if rc == 0 {
		return 0, io.EOF
	} else if rc < 0 {
		s := C.GoString(C.archive_error_string(rd.archive))
		return 0, errors.New(s)
	}
	return rc, nil
}

///////////////////////////////////////////////////////////////////////////////
// Writer
///////////////////////////////////////////////////////////////////////////////

type Writer struct {
	archive *C.struct_archive
	entry *C.struct_archive_entry
	header bool
}

func CreateWriter(name string) (*Writer, error) {
	archive := C.archive_write_new()
	ext := strings.ToLower(path.Ext(name))
	switch ext {
	case ".zip", ".7z":
		C.archive_write_add_filter_none(archive)
	case ".gz", ".tgz":
		C.archive_write_add_filter_gzip(archive)
	case ".bz2":
		C.archive_write_add_filter_bzip2(archive)
	case ".xz":
		C.archive_write_add_filter_xz(archive)
	case ".zst":
		C.archive_write_add_filter_zstd(archive)
	default:
		return nil, errors.New("Unknown file extension")
	}

	switch ext {
	case ".7z":
		C.archive_write_set_format_7zip(archive);
	case ".zip":
		C.archive_write_set_format_zip(archive);
	default:
		C.archive_write_set_format_pax_restricted(archive);
	}

	if C.archive_write_open_filename(archive, C.CString(name)) != C.ARCHIVE_OK {
		s := C.GoString(C.archive_error_string(archive))
		C.archive_write_free(archive)
		return nil, errors.New(s)
	}
	return &Writer{
		archive: archive,
		entry: C.archive_entry_new(),
	}, nil
}

func (wr *Writer) Close() {
	if wr.archive == nil {
		return
	}
	C.archive_entry_free(wr.entry)
	C.archive_write_free(wr.archive)
	wr.archive = nil
}

func (wr *Writer) New(name string, size int64) {
	C.archive_entry_clear(wr.entry)
	C.archive_entry_set_pathname(wr.entry, C.CString(name))
	C.archive_entry_set_size(wr.entry, C.int64_t(size))
	C.archive_entry_set_filetype(wr.entry, C.AE_IFREG)
	C.archive_entry_set_perm(wr.entry, 0644)
	wr.header = true
}

func (wr *Writer) Clone(rd *Reader) {
	C.archive_entry_free(wr.entry)
	wr.entry = C.archive_entry_clone(rd.entry)
	wr.header = true
}

func (wr *Writer) Path(name string) {
	C.archive_entry_set_pathname(wr.entry, C.CString(name))
}

func (wr *Writer) Size(size int64) {
	C.archive_entry_set_size(wr.entry, C.int64_t(size))
}

func (wr *Writer) ModTime(mt time.Time) {
	C.archive_entry_set_mtime(wr.entry, C.time_t(mt.Unix()), C.int64_t(mt.Nanosecond()))
}

func (wr *Writer) Write(buf []byte) (int, error) {
	if (wr.header) {
		C.archive_write_header(wr.archive, wr.entry)
		wr.header = false
	}
	rc := int(C.archive_write_data(wr.archive, unsafe.Pointer(&buf[0]), C.size_t(len(buf))))
	if rc < 0 {
		s := C.GoString(C.archive_error_string(wr.archive))
		return 0, errors.New(s)
	}
	return rc, nil
}
