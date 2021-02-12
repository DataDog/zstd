package zstd

/*
#define ZSTD_STATIC_LINKING_ONLY
#include "zstd.h"
#include "stdint.h"  // for uintptr_t

// The following *_wrapper function are used for removing superflouos
// memory allocations when calling the wrapped functions from Go code.
// See https://github.com/golang/go/issues/24450 for details.

static size_t ZSTD_compress_wrapper(uintptr_t dst, size_t maxDstSize, const uintptr_t src, size_t srcSize, int compressionLevel) {
	return ZSTD_compress((void*)dst, maxDstSize, (const void*)src, srcSize, compressionLevel);
}

static size_t ZSTD_decompress_wrapper(uintptr_t dst, size_t maxDstSize, uintptr_t src, size_t srcSize) {
	return ZSTD_decompress((void*)dst, maxDstSize, (const void *)src, srcSize);
}

static size_t ZSTD_compress_usingDict_wrapper(ZSTD_CCtx* ctx, uintptr_t dst, size_t maxDstSize, const uintptr_t src, size_t srcSize,  const uintptr_t dict, size_t dictSize, int compressionLevel) {
	return ZSTD_compress_usingDict(ctx, (void*)dst, maxDstSize, (const void*)src, srcSize, (const void*)dict, dictSize, compressionLevel);
}

static size_t ZSTD_decompress_usingDict_wrapper(ZSTD_DCtx* ctx, uintptr_t dst, size_t maxDstSize, uintptr_t src, size_t srcSize, uintptr_t dict, size_t dictSize) {
	return ZSTD_decompress_usingDict(ctx, (void*)dst, maxDstSize, (const void *)src, srcSize, (const void *)dict, dictSize);
}

*/
import "C"
import (
	"bytes"
	"errors"
	"io/ioutil"
	"runtime"
	"unsafe"
)

// Defines best and standard values for zstd cli
const (
	BestSpeed          = 1
	BestCompression    = 20
	DefaultCompression = 5
)

var (
	// ErrEmptySlice is returned when there is nothing to compress
	ErrEmptySlice = errors.New("Bytes slice is empty")
)

// CompressBound returns the worst case size needed for a destination buffer,
// which can be used to preallocate a destination buffer or select a previously
// allocated buffer from a pool.
// See zstd.h to mirror implementation of ZSTD_COMPRESSBOUND
func CompressBound(srcSize int) int {
	lowLimit := 128 << 10 // 128 kB
	var margin int
	if srcSize < lowLimit {
		margin = (lowLimit - srcSize) >> 11
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
	return CompressLevelDict(dst, src, DefaultCompression, nil)
}

// CompressLevel is the same as Compress but you can pass a compression level
func CompressLevel(dst, src []byte, level int) ([]byte, error) {
	return CompressLevelDict(dst, src, level, nil)
}

// CompressLevelDict is the same as CompressLevel but you can pass a dictionary
func CompressLevelDict(dst, src []byte, level int, dict []byte) ([]byte, error) {
	bound := CompressBound(len(src))
	if cap(dst) >= bound {
		dst = dst[0:bound] // Reuse dst buffer
	} else {
		dst = make([]byte, bound)
	}

	var written int
	if dict == nil {
		srcPtr := C.uintptr_t(uintptr(0)) // Do not point anywhere, if src is empty
		if len(src) > 0 {
			srcPtr = C.uintptr_t(uintptr(unsafe.Pointer(&src[0])))
		}

		cWritten := C.ZSTD_compress_wrapper(
			C.uintptr_t(uintptr(unsafe.Pointer(&dst[0]))),
			C.size_t(len(dst)),
			srcPtr,
			C.size_t(len(src)),
			C.int(level))
		written = int(cWritten)
	} else {
		ctx := C.ZSTD_createCStream()
		cWritten := C.ZSTD_compress_usingDict_wrapper(
			ctx,
			C.uintptr_t(uintptr(unsafe.Pointer(&dst[0]))),
			C.size_t(len(dst)),
			C.uintptr_t(uintptr(unsafe.Pointer(&src[0]))),
			C.size_t(len(src)),
			C.uintptr_t(uintptr(unsafe.Pointer(&dict[0]))),
			C.size_t(len(dict)),
			C.int(level),
		)
		written = int(cWritten)
	}

	runtime.KeepAlive(src)
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
	return DecompressDict(dst, src, nil)
}

// DecompressDict is the same as Decompress but you can pass a dictionary
func DecompressDict(dst, src []byte, dict []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, ErrEmptySlice
	}
	decompress := func(dst, src []byte) ([]byte, error) {

		var written int
		if dict == nil {
			cWritten := C.ZSTD_decompress_wrapper(
				C.uintptr_t(uintptr(unsafe.Pointer(&dst[0]))),
				C.size_t(len(dst)),
				C.uintptr_t(uintptr(unsafe.Pointer(&src[0]))),
				C.size_t(len(src)))
			written = int(cWritten)
		} else {
			dctx := C.ZSTD_createDCtx()
			cWritten := C.ZSTD_decompress_usingDict_wrapper(
				dctx,
				C.uintptr_t(uintptr(unsafe.Pointer(&dst[0]))),
				C.size_t(len(dst)),
				C.uintptr_t(uintptr(unsafe.Pointer(&src[0]))),
				C.size_t(len(src)),
				C.uintptr_t(uintptr(unsafe.Pointer(&dict[0]))),
				C.size_t(len(dict)),
			)
			written = int(cWritten)
		}

		runtime.KeepAlive(src)
		// Check error
		if err := getError(written); err != nil {
			return nil, err
		}
		return dst[:written], nil
	}

	if len(dst) == 0 {
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
