package zstd

// The lib is going to be compiled in /tmp

/*
#include <zstd.h>
#cgo LDFLAGS: /usr/local/lib/libzstd.a
*/
import "C"
import (
	"errors"
	"unsafe"
)

var (
	ErrGeneric                           = errors.New("Error (generic)")
	ErrPrefixUnknown                     = errors.New("Unknown frame descriptor")
	ErrFrameParameterUnsupported         = errors.New("Unsupported frame parameter")
	ErrFrameParameterUnsupportedBy32bits = errors.New("Frame parameter unsupported in 32-bits mode")
	ErrInitMissing                       = errors.New("Context should be init first")
	ErrMemoryAllocation                  = errors.New("Allocation error : not enough memory")
	ErrStageWrong                        = errors.New("Operation not authorized at current processing stage")
	ErrDstSizeTooSmall                   = errors.New("Destination buffer is too small")
	ErrSrcSizeWrong                      = errors.New("Src size incorrect")
	ErrCorruptionDetected                = errors.New("Corrupted block detected")
	ErrTableLogTooLarge                  = errors.New("tableLog requires too much memory")
	ErrMaxSymbolValueTooLarge            = errors.New("Unsupported max possible Symbol Value : too large")
	ErrMaxSymbolValueTooSmall            = errors.New("Specified maxSymbolValue is too small")
	ErrDictionaryCorrupted               = errors.New("Dictionary is corrupted")
)

// CompressBound returns the worst case size needed for a destination buffer
// Implentation is taken from the C code
func CompressBound(srcSize int) int {
	return 512 + srcSize + (srcSize >> 7) + 12
}

// Internal call to the C function to check that our implentation match
func c_CompressBound(srcSize int) int {
	return int(C.ZSTD_compressBound(C.size_t(srcSize)))
}

// getError return whether the returned int signify an error
// otherwise returns nil
func getError(code int) error {
	switch code {
	case -1:
		return ErrGeneric
	case -2:
		return ErrPrefixUnknown
	case -3:
		return ErrFrameParameterUnsupported
	case -4:
		return ErrFrameParameterUnsupportedBy32bits
	case -5:
		return ErrInitMissing
	case -6:
		return ErrMemoryAllocation
	case -7:
		return ErrStageWrong
	case -8:
		return ErrDstSizeTooSmall
	case -9:
		return ErrSrcSizeWrong
	case -10:
		return ErrCorruptionDetected
	case -11:
		return ErrTableLogTooLarge
	case -12:
		return ErrMaxSymbolValueTooLarge
	case -13:
		return ErrMaxSymbolValueTooSmall
	case -14:
		return ErrDictionaryCorrupted
	}

	return nil
}

func c_isError(code int) bool {
	isErr := int(C.ZSTD_isError(C.size_t(code)))
	if isErr != 0 {
		return true
	}
	return false
}

// Compress allocate a byte array and compress the data
func Compress(dst, src []byte) ([]byte, error) {
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

	cWritten := C.ZSTD_compress(cDst, cDstCap, cSrc, cSrcSize, 5)
	written := int(cWritten)
	// Check if the return is an Error code
	if getError(written) != nil {
		return nil, getError(written)
	}
	return dst[:written], nil
}

func Decompress(dst, src []byte) ([]byte, error) {
	decompress := func(dst, src []byte) ([]byte, error) {
		cDst := unsafe.Pointer(&dst[0])
		cDstCap := C.size_t(len(dst))
		cSrc := unsafe.Pointer(&src[0])
		cSrcSize := C.size_t(len(src))

		cWritten := C.ZSTD_decompress(cDst, cDstCap, cSrc, cSrcSize)
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
