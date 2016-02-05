package zstd

/*
#include <zstd.h>
#cgo LDFLAGS: /usr/local/lib/libzstd.a
*/
import "C"
import "unsafe"

// Zstd is a zstd compressor object that wraps a zstd context
// You must call Close() at the end to free the context
type Zstd struct {
	CompressionLevel int

	compressCtx   *C.ZSTD_CCtx
	decompressCtx *C.ZSTD_DCtx
	dict          []byte
}

// NewZstd creates a new object, that can optionnaly be initialized with
// a precomputed dictionnary. If dict is nil, compress without a dictionnary
// the underlying byte array should not be changed during the use of the object.
func NewZstd(dict []byte, compressionLevel int) *Zstd {
	compressCtx := C.ZSTD_createCCtx()
	decompressCtx := C.ZSTD_createDCtx()
	return &Zstd{
		CompressionLevel: compressionLevel,
		compressCtx:      compressCtx,
		decompressCtx:    decompressCtx,
		dict:             dict,
	}
}

// Close frees all the underlying C objects (context)
func (z *Zstd) Close() error {
	code := C.ZSTD_freeCCtx(z.compressCtx)
	code2 := C.ZSTD_freeDCtx(z.decompressCtx)
	if getError(int(code)) != nil {
		return getError(int(code))
	}
	return getError(int(code2))
}

// Compress compress a payload using the given context
func (z *Zstd) Compress(dst, src []byte) ([]byte, error) {
	bound := CompressBound(len(src))
	if cap(dst) >= bound {
		dst = dst[0:bound] // Reuse dst buffer
	} else {
		dst = make([]byte, bound)
	}
	cDst := unsafe.Pointer(&dst[0])
	cDstCap := C.size_t(len(dst))
	cSrc := unsafe.Pointer(&src[0])
	cSrcSize := C.size_t(len(src))
	var cDict unsafe.Pointer
	if z.dict != nil {
		cDict = unsafe.Pointer(&z.dict[0])
	}
	cDictSize := C.size_t(len(z.dict))
	cCompressionLevel := C.int(z.CompressionLevel)

	cWritten := C.ZSTD_compress_usingDict(z.compressCtx,
		cDst, cDstCap,
		cSrc, cSrcSize,
		cDict, cDictSize,
		cCompressionLevel)
	written := int(cWritten)
	// Check if the return is an Error code
	if getError(written) != nil {
		return nil, getError(written)
	}
	return dst[:written], nil
}

// Decompress decompress a payload using the given context
func (z *Zstd) Decompress(dst, src []byte) ([]byte, error) {
	decompress := func(dst, src []byte) ([]byte, error) {
		cDst := unsafe.Pointer(&dst[0])
		cDstCap := C.size_t(len(dst))
		cSrc := unsafe.Pointer(&src[0])
		cSrcSize := C.size_t(len(src))
		var cDict unsafe.Pointer
		if z.dict != nil {
			cDict = unsafe.Pointer(&z.dict[0])
		}
		cDictSize := C.size_t(len(z.dict))

		cWritten := C.ZSTD_decompress_usingDict(z.decompressCtx,
			cDst, cDstCap,
			cSrc, cSrcSize,
			cDict, cDictSize)

		written := int(cWritten)
		// Check error
		if getError(written) != nil {
			return nil, getError(written)
		}
		return dst[:written], nil
	}

	if dst == nil {
		// x is the 95 percentile compression ratio of zstd on points.mlti payloads
		dst = make([]byte, len(src)*3)
	}
	for i := 0; i < 3; i++ { // 3 tries to allocate a bigger buffer
		result, err := decompress(dst, src)
		if err != ErrDstSizeTooSmall {
			return result, err
		}
		dst = make([]byte, len(dst)*2) // Grow buffer by 2
	}
	return []byte{}, ErrDstSizeTooSmall
}
