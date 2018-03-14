package zstd

/*
#define ZSTD_STATIC_LINKING_ONLY
#include "zstd.h"
*/
import "C"
import (
	"bytes"
	"errors"
	"io/ioutil"
	"unsafe"
)

const (
	BestSpeed       = 1
	BestCompression = 20
)

var (
	ErrEmptySlice      = errors.New("Bytes slice is empty")
	DefaultCompression = 5
)

// CompressBound returns the worst case size needed for a destination buffer,
// which can be used to preallocate a destination buffer or select a previously
// allocated buffer from a pool.
func CompressBound(srcSize int) int {
	lowLimit := 256 * 1024 // 256 kB
	var margin int
	if srcSize < lowLimit {
		margin = (lowLimit - srcSize) >> 12
	}
	return srcSize + (srcSize >> 8) + margin
}

// cCompressBound is a cgo call to check the go implementation above against the c code.
func cCompressBound(srcSize int) int {
	return int(C.ZSTD_compressBound(C.size_t(srcSize)))
}

// Compress src into dst.  If you have a buffer to use, you can pass it to
// prevent allocation.  If it is too small, or if nil is passed, a new buffer
// will be allocated and returned.
func Compress(dst, src []byte) ([]byte, error) {
	return CompressLevel(dst, src, DefaultCompression)
}

// CompressLevel is the same as Compress but you can pass a compression level
func CompressLevel(dst, src []byte, level int) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, ErrEmptySlice
	}
	bound := CompressBound(len(src))
	if cap(dst) >= bound {
		dst = dst[0:bound] // Reuse dst buffer
	} else {
		dst = make([]byte, bound)
	}

	cWritten := C.ZSTD_compress(
		unsafe.Pointer(&dst[0]),
		C.size_t(len(dst)),
		unsafe.Pointer(&src[0]),
		C.size_t(len(src)),
		C.int(level))

	written := int(cWritten)
	// Check if the return is an Error code
	if err := getError(written); err != nil {
		return nil, err
	}
	return dst[:written], nil
}

// Decompress src into dst.  If you have a buffer to use, you can pass it to
// prevent allocation.  If it is too small, or if nil is passed, a new buffer
// will be allocated and returned.
func Decompress(dst, src []byte) ([]byte, error) {
	decompress := func(dst, src []byte) ([]byte, error) {

		cWritten := C.ZSTD_decompress(
			unsafe.Pointer(&dst[0]),
			C.size_t(len(dst)),
			unsafe.Pointer(&src[0]),
			C.size_t(len(src)))

		written := int(cWritten)
		// Check error
		if err := getError(written); err != nil {
			return nil, err
		}
		return dst[:written], nil
	}

	if dst == nil {
		// Attempt to use zStd to determine decompressed size (may result in error or 0)
		size := int(C.size_t(C.ZSTD_getDecompressedSize(unsafe.Pointer(&src[0]), C.size_t(len(src)))))

		if err := getError(size); err != nil {
			return nil, err
		}

		if size > 0 {
			dst = make([]byte, size)
		} else {
			dst = make([]byte, len(src)*3) // starting guess
		}
	}
	for i := 0; i < 3; i++ { // 3 tries to allocate a bigger buffer
		result, err := decompress(dst, src)
		if !IsDstSizeTooSmallError(err) {
			return result, err
		}
		dst = make([]byte, len(dst)*2) // Grow buffer by 2
	}

	// We failed getting a dst buffer of correct size, use stream API
	r := NewReader(bytes.NewReader(src))
	defer r.Close()
	return ioutil.ReadAll(r)
}
