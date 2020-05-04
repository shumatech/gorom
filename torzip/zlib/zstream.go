// Copyright 2015, Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package zlib

/*
#cgo LDFLAGS: -lz
#cgo CFLAGS: -Werror=implicit

#cgo windows LDFLAGS: -Lc:/mingw/lib
#cgo windows CFLAGS: -Ic:/mingw/include

#include "zlib.h"

// deflateInit2 is a macro, so using a wrapper function
// using deflateInit2 instead of deflateInit to be able to specify gzip format
int zlibstream_deflate_init(char *strm, int level) {
  ((z_stream*)strm)->zalloc = Z_NULL;
  ((z_stream*)strm)->zfree = Z_NULL;
  ((z_stream*)strm)->opaque = Z_NULL;
  return deflateInit2((z_stream*)strm, Z_BEST_COMPRESSION, Z_DEFLATED,
                        -15, // 15 is default
                        8, Z_DEFAULT_STRATEGY); // default values
}

unsigned int zlibstream_avail_in(char *strm) {
  return ((z_stream*)strm)->avail_in;
}

unsigned int zlibstream_avail_out(char *strm) {
  return ((z_stream*)strm)->avail_out;
}

char* zlibstream_msg(char *strm) {
  return ((z_stream*)strm)->msg;
}

void zlibstream_set_in_buf(char *strm, void *buf, unsigned int len) {
  ((z_stream*)strm)->next_in = (Bytef*)buf;
  ((z_stream*)strm)->avail_in = len;
}

void zlibstream_set_out_buf(char *strm, void *buf, unsigned int len) {
  ((z_stream*)strm)->next_out = (Bytef*)buf;
  ((z_stream*)strm)->avail_out = len;
}

int zlibstream_deflate(char *strm, int flag) {
  return deflate((z_stream*)strm, flag);
}

void zlibstream_deflate_end(char *strm) {
  deflateEnd((z_stream*)strm);
}

int zlibstream_deflate_reset(char *strm) {
  return deflateReset((z_stream*)strm);
}

*/
import "C"

import (
	"fmt"
	"unsafe"
)

const (
	zNoFlush = C.Z_NO_FLUSH
)

// z_stream is a buffer that's big enough to fit a C.z_stream.
// This lets us allocate a C.z_stream within Go, while keeping the contents
// opaque to the Go GC. Otherwise, the GC would look inside and complain that
// the pointers are invalid, since they point to objects allocated by C code.
type zstream [unsafe.Sizeof(C.z_stream{})]C.char

func (strm *zstream) deflateInit(level int) error {
	result := C.zlibstream_deflate_init(&strm[0], C.int(level))
	if result != Z_OK {
		return fmt.Errorf("zlib: failed to initialize deflate (%v): %v", result, strm.msg())
	}
	return nil
}

func (strm *zstream) deflateEnd() {
	C.zlibstream_deflate_end(&strm[0])
}

func (strm *zstream) availIn() int {
	return int(C.zlibstream_avail_in(&strm[0]))
}

func (strm *zstream) availOut() int {
	return int(C.zlibstream_avail_out(&strm[0]))
}

func (strm *zstream) msg() string {
	return C.GoString(C.zlibstream_msg(&strm[0]))
}

func (strm *zstream) setInBuf(buf []byte, size int) {
	if buf == nil {
		C.zlibstream_set_in_buf(&strm[0], nil, C.uint(size))
	} else {
		C.zlibstream_set_in_buf(&strm[0], unsafe.Pointer(&buf[0]), C.uint(size))
	}
}

func (strm *zstream) setOutBuf(buf []byte, size int) {
	if buf == nil {
		C.zlibstream_set_out_buf(&strm[0], nil, C.uint(size))
	} else {
		C.zlibstream_set_out_buf(&strm[0], unsafe.Pointer(&buf[0]), C.uint(size))
	}
}

func (strm *zstream) deflate(flag int) {
	ret := C.zlibstream_deflate(&strm[0], C.int(flag))
	if ret == Z_STREAM_ERROR {
		// all the other error cases are normal,
		// and this should never happen
		panic(fmt.Errorf("zlib: unexpected error"))
	}
}

func (strm *zstream) deflateReset() {
	ret := C.zlibstream_deflate_reset(&strm[0])
	if ret == Z_STREAM_ERROR {
		panic(fmt.Errorf("zlib: unexpected error"))
	}
}
