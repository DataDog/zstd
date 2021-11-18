//go:build external_libzstd
// +build external_libzstd

package zstd

// #cgo CFLAGS: -DUSE_EXTERNAL_ZSTD
// #cgo pkg-config: libzstd
import "C"
