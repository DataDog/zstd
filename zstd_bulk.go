package zstd

/*
#include "zstd.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

// BulkProcessor implements Bulk processing dictionary API
type BulkProcessor struct {
	cDict *C.struct_ZSTD_CDict_s
	dDict *C.struct_ZSTD_DDict_s
}

// NewBulkProcessor creates a new BulkProcessor with a pre-trained dictionary and compression level
func NewBulkProcessor(dictionary []byte, compressionLevel int) (*BulkProcessor, error) {
	p := &BulkProcessor{}
	p.cDict = C.ZSTD_createCDict(
		unsafe.Pointer(&dictionary[0]),
		C.size_t(len(dictionary)),
		C.int(compressionLevel),
	)
	if p.cDict == nil {
		return nil, errors.New("failed to create dictionary")
	}
	p.dDict = C.ZSTD_createDDict(
		unsafe.Pointer(&dictionary[0]),
		C.size_t(len(dictionary)),
	)
	if p.dDict == nil {
		return nil, errors.New("failed to create dictionary")
	}
	return p, nil
}

// Compress compresses the `src` with the dictionary
func (p *BulkProcessor) Compress(dst, src []byte) ([]byte, error) {
	bound := CompressBound(len(src))
	if cap(dst) >= bound {
		dst = dst[0:bound]
	} else {
		dst = make([]byte, bound)
	}

	var cSrc unsafe.Pointer
	if len(src) == 0 {
		cSrc = unsafe.Pointer(nil)
	} else {
		cSrc = unsafe.Pointer(&src[0])
	}

	cctx := C.ZSTD_createCCtx()
	cWritten := C.ZSTD_compress_usingCDict(
		cctx,
		unsafe.Pointer(&dst[0]),
		C.size_t(len(dst)),
		cSrc,
		C.size_t(len(src)),
		p.cDict,
	)
	C.ZSTD_freeCCtx(cctx)

	written := int(cWritten)
	if err := getError(written); err != nil {
		return nil, err
	}
	return dst[:written], nil
}

// Decompress compresses the `dst` with the dictionary
func (p *BulkProcessor) Decompress(dst, src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, ErrEmptySlice
	}
	contentSize := uint64(C.ZSTD_getFrameContentSize(unsafe.Pointer(&src[0]), C.size_t(len(src))))
	if contentSize == C.ZSTD_CONTENTSIZE_ERROR || contentSize == C.ZSTD_CONTENTSIZE_UNKNOWN {
		return nil, errors.New("could not determine the content size")
	}

	if cap(dst) >= int(contentSize) {
		dst = dst[0:contentSize]
	} else {
		dst = make([]byte, contentSize)
	}

	if contentSize == 0 {
		return dst, nil
	}

	dctx := C.ZSTD_createDCtx()
	cWritten := C.ZSTD_decompress_usingDDict(
		dctx,
		unsafe.Pointer(&dst[0]),
		C.size_t(contentSize),
		unsafe.Pointer(&src[0]),
		C.size_t(len(src)),
		p.dDict,
	)
	C.ZSTD_freeDCtx(dctx)

	written := int(cWritten)
	if err := getError(written); err != nil {
		return nil, err
	}

	return dst[:written], nil
}

// Cleanup frees compression and decompression dictionaries from memory
func (p *BulkProcessor) Cleanup() {
	C.ZSTD_freeCDict(p.cDict)
	C.ZSTD_freeDDict(p.dDict)
}
