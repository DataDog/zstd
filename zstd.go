package zstd

/*
// support decoding of "legacy" zstd payloads from versions [0.4, 0.8], matching the
// default configuration of the zstd command line tool:
// https://github.com/facebook/zstd/blob/dev/programs/README.md
#cgo CFLAGS: -DZSTD_LEGACY_SUPPORT=4 -DZSTD_MULTITHREAD=1

#include "zstd.h"
*/
import "C"
import (
	"bytes"
	"errors"
	"io/ioutil"
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

const (
	// decompressSizeBufferLimit is the limit we set on creating a decompression buffer for the Decompress API
	// This is made to prevent DOS from maliciously-created payloads (aka zipbomb).
	// For large payloads with a compression ratio > 10, you can do your own allocation and pass it to the method:
	// dst := make([]byte, 1GB)
	// decompressed, err := zstd.Decompress(dst, src)
	decompressSizeBufferLimit = 1000 * 1000

	zstdFrameHeaderSizeMin = 2 // From zstd.h. Since it's experimental API, hardcoding it
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

// decompressSizeHint returns a suggested output size from the frame header, capped to guard against
// zip bombs. foundHint is false when the frame does not advertise its size (legacy v0.5 or unpledged
// streaming frames); the returned hint is then only a pessimistic upper bound.
func decompressSizeHint(src []byte) (hint int, foundHint bool) {
	// 1 MB or 50x input size
	upperBound := 50 * len(src)
	if upperBound < decompressSizeBufferLimit {
		upperBound = decompressSizeBufferLimit
	}

	hint = upperBound
	if len(src) >= zstdFrameHeaderSizeMin {
		contentSize := int(C.ZSTD_getFrameContentSize(unsafe.Pointer(&src[0]), C.size_t(len(src))))
		if contentSize >= 0 { // a negative value means the size is unknown or the header is in error
			foundHint = true
			hint = contentSize
			if hint == 0 { // When compressing the empty slice, we need an output of at least 1 to pass down to the C lib
				hint = 1
			}
		}
	}

	// Take the minimum of both
	if hint > upperBound {
		return upperBound, foundHint
	}
	return hint, foundHint
}

// Compress src into dst.  If you have a buffer to use, you can pass it to
// prevent allocation.  If it is too small, or if nil is passed, a new buffer
// will be allocated and returned.
func Compress(dst, src []byte) ([]byte, error) {
	return CompressLevel(dst, src, DefaultCompression)
}

// CompressLevel is the same as Compress but you can pass a compression level
func CompressLevel(dst, src []byte, level int) ([]byte, error) {
	bound := CompressBound(len(src))
	if cap(dst) >= bound {
		dst = dst[0:bound] // Reuse dst buffer
	} else {
		dst = make([]byte, bound)
	}

	// We need unsafe.Pointer(&src[0]) in the Cgo call to avoid "Go pointer to Go pointer" panics.
	// This means we need to special case empty input. See:
	// https://github.com/golang/go/issues/14210#issuecomment-346402945
	var cWritten C.size_t
	if len(src) == 0 {
		cWritten = C.ZSTD_compress(
			unsafe.Pointer(&dst[0]),
			C.size_t(len(dst)),
			unsafe.Pointer(nil),
			C.size_t(0),
			C.int(level))
	} else {
		cWritten = C.ZSTD_compress(
			unsafe.Pointer(&dst[0]),
			C.size_t(len(dst)),
			unsafe.Pointer(&src[0]),
			C.size_t(len(src)),
			C.int(level))
	}

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
//
// Note: for frames that do not advertise their size (legacy v0.5 or unpledged
// streaming frames) dst may be partially overwritten even if a new slice is
// returned; do not rely on dst's contents after such a call.
func Decompress(dst, src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, ErrEmptySlice
	}

	hint, foundHint := decompressSizeHint(src)

	// Reuse the caller buffer when it is large enough, or when the size is
	// unknown (the hint is then only an upper bound); otherwise allocate the hint.
	switch {
	case cap(dst) >= hint:
		dst = dst[:cap(dst)]
	case !foundHint && cap(dst) > 0:
		dst = dst[:cap(dst)]
	default:
		dst = make([]byte, hint)
	}

	written, err := DecompressInto(dst, src)
	if err == nil {
		return dst[:written], nil
	}
	if !IsDstSizeTooSmallError(err) {
		return nil, err
	}

	// We failed getting a dst buffer of correct size, use stream API
	r := NewReader(bytes.NewReader(src))
	defer r.Close()
	return ioutil.ReadAll(r)
}

// DecompressInto decompresses src into dst. Unlike Decompress, DecompressInto
// requires that dst be sufficiently large to hold the decompressed payload.
// DecompressInto may be used when the caller knows the size of the decompressed
// payload before attempting decompression.
//
// It returns the number of bytes copied and an error if any is encountered. If
// dst is too small, DecompressInto errors.
func DecompressInto(dst, src []byte) (int, error) {
	written := int(C.ZSTD_decompress(
		unsafe.Pointer(&dst[0]),
		C.size_t(len(dst)),
		unsafe.Pointer(&src[0]),
		C.size_t(len(src))))
	return written, getError(written)
}
