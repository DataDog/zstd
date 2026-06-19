package zstd

import (
	"bytes"
	"fmt"
	"testing"
)

// perfPayload builds a compressible, log/JSON-like payload of approximately
// size bytes, representative of the per-message data the DataDog services
// compress in practice.
func perfPayload(size int) []byte {
	var b bytes.Buffer
	i := 0
	for b.Len() < size {
		fmt.Fprintf(&b,
			`{"ts":%d,"level":"INFO","service":"trace-writer","host":"i-%08x","trace_id":%d,"duration_ms":%d,"msg":"request completed"}`+"\n",
			1718800000+i, i*2654435761, int64(i)*1099511628211, i%5000)
		i++
	}
	return b.Bytes()[:size]
}

func benchmarkOneShotCompress(b *testing.B, size int) {
	src := perfPayload(size)
	dst := make([]byte, CompressBound(len(src)))
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Compress(dst, src); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkOneShotDecompress(b *testing.B, size int) {
	src := perfPayload(size)
	comp, err := Compress(nil, src)
	if err != nil {
		b.Fatal(err)
	}
	dst := make([]byte, len(src))
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := DecompressInto(dst, comp); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOneShotCompress1K(b *testing.B)    { benchmarkOneShotCompress(b, 1024) }
func BenchmarkOneShotCompress8K(b *testing.B)    { benchmarkOneShotCompress(b, 8*1024) }
func BenchmarkOneShotCompress64K(b *testing.B)   { benchmarkOneShotCompress(b, 64*1024) }
func BenchmarkOneShotDecompress1K(b *testing.B)  { benchmarkOneShotDecompress(b, 1024) }
func BenchmarkOneShotDecompress8K(b *testing.B)  { benchmarkOneShotDecompress(b, 8*1024) }
func BenchmarkOneShotDecompress64K(b *testing.B) { benchmarkOneShotDecompress(b, 64*1024) }

// The *Nil benchmarks model call sites that pass dst == nil (the common
// pattern in dd-go/dd-source), forcing a Go-heap allocation per call. Compare
// their B/op and ns/op against the reused-buffer benchmarks above to see the
// GC cost that a caller-side buffer pool would remove.

func benchmarkOneShotCompressNil(b *testing.B, size int) {
	src := perfPayload(size)
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Compress(nil, src); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkOneShotDecompressNil(b *testing.B, size int) {
	src := perfPayload(size)
	comp, err := Compress(nil, src)
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Decompress(nil, comp); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOneShotCompressNil1K(b *testing.B)    { benchmarkOneShotCompressNil(b, 1024) }
func BenchmarkOneShotCompressNil8K(b *testing.B)    { benchmarkOneShotCompressNil(b, 8*1024) }
func BenchmarkOneShotCompressNil64K(b *testing.B)   { benchmarkOneShotCompressNil(b, 64*1024) }
func BenchmarkOneShotDecompressNil1K(b *testing.B)  { benchmarkOneShotDecompressNil(b, 1024) }
func BenchmarkOneShotDecompressNil8K(b *testing.B)  { benchmarkOneShotDecompressNil(b, 8*1024) }
func BenchmarkOneShotDecompressNil64K(b *testing.B) { benchmarkOneShotDecompressNil(b, 64*1024) }

// BenchmarkOneShotCompressParallel mirrors highly concurrent services that
// compress from many goroutines, exercising context-pool scalability.
func BenchmarkOneShotCompressParallel(b *testing.B) {
	src := perfPayload(8 * 1024)
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		dst := make([]byte, CompressBound(len(src)))
		for pb.Next() {
			if _, err := Compress(dst, src); err != nil {
				b.Fatal(err)
			}
		}
	})
}
