package zstd

/*
#define ZSTD_STATIC_LINKING_ONLY
#include "zstd.h"
*/
import "C"

// ErrorCode is an error returned by the zstd library.
type ErrorCode int

func (e ErrorCode) Error() string {
	a := C.ZSTD_getErrorName(C.size_t(e))
	return C.GoString(a)
}

func cIsError(code int) bool {
	return int(C.ZSTD_isError(C.size_t(code))) != 0
}

// getError returns an error for the return code, or nil if it's not an error
func getError(code int) error {
	if code < 0 && cIsError(code) {
		return ErrorCode(code)
	}
	return nil
}

func IsDstSizeTooSmallError(e error) bool {
	if e != nil && e.Error() == "Destination buffer is too small" {
		return true
	}
	return false
}
