package zstd

import (
	"bytes"
	"io"
	"testing"
)

func failOnError(t *testing.T, msg string, err error) {
	if err != nil {
		t.Fatalf("%s: %s", msg, err)
	}
}

func testCompressionDecompression(t *testing.T, dict []byte, payload []byte) {
	var w bytes.Buffer
	writer := NewWriter(&w, dict, 5)
	_, err := writer.Write(payload)
	failOnError(t, "Failed writing to compress object", err)
	failOnError(t, "Failed to close compress object", writer.Close())
	out := w.Bytes()
	t.Logf("Compressed %v -> %v bytes", len(payload), len(out))
	failOnError(t, "Failed compressing", err)
	rr := bytes.NewReader(out)
	// Check that we can decompress with Decompress()
	decompressed, err := Decompress(nil, out)
	failOnError(t, "Failed to decompress with Decompress()", err)
	if string(payload) != string(decompressed) {
		t.Fatalf("Payload did not match, lengths: %v & %v", len(payload), len(decompressed))
	}

	// Decompress
	r := NewReader(rr, dict)
	dst := make([]byte, len(payload))
	n, err := r.Read(dst)
	if err != nil {
		failOnError(t, "Failed to read for decompression", err)
	}
	dst = dst[:n]
	if string(payload) != string(dst) { // Only print if we can print
		if len(payload) < 100 && len(dst) < 100 {
			t.Fatalf("Cannot compress and decompress: %s != %s", payload, dst)
		} else {
			t.Fatalf("Cannot compress and decompress (lengths: %v bytes & %v bytes)", len(payload), len(dst))
		}
	}
	// Check EOF
	n, err = r.Read(dst)
	if err != io.EOF && len(dst) > 0 { // If we want 0 bytes, that should work
		t.Fatalf("Error should have been EOF, was %s instead: (%v bytes read: %s)", err, n, dst[:n])
	}
	failOnError(t, "Failed to close decompress object", r.Close())
}

func TestStreamSimpleCompressionDecompression(t *testing.T) {
	testCompressionDecompression(t, nil, []byte("Hello world!"))
}

func TestStreamEmptySlice(t *testing.T) {
	testCompressionDecompression(t, nil, []byte{})
}

func TestZstdReaderLong(t *testing.T) {
	var long bytes.Buffer
	for i := 0; i < 10000; i++ {
		long.Write([]byte("Hellow World!"))
	}
	testCompressionDecompression(t, nil, long.Bytes())
}

func TestStreamCompressionDecompression(t *testing.T) {
	payload := []byte("Hello World!")
	repeat := 10000
	var intermediate bytes.Buffer
	w := NewWriter(&intermediate, nil, 4)
	for i := 0; i < repeat; i++ {
		_, err := w.Write(payload)
		failOnError(t, "Failed writing to compress object", err)
	}
	w.Close()
	// Decompress
	r := NewReader(&intermediate, nil)
	dst := make([]byte, len(payload))
	for i := 0; i < repeat; i++ {
		n, err := r.Read(dst)
		failOnError(t, "Failed to decompress", err)
		if n != len(payload) {
			t.Fatalf("Did not read enough bytes: %v != %v", n, len(payload))
		}
		if string(dst) != string(payload) {
			t.Fatalf("Did not read the same %s != %s", string(dst), string(payload))
		}
	}
	// Check EOF
	n, err := r.Read(dst)
	if err != io.EOF {
		t.Fatalf("Error should have been EOF, was %s instead: (%v bytes read: %s)", err, n, dst[:n])
	}
	failOnError(t, "Failed to close decompress object", r.Close())
}

func TestStreamDict(t *testing.T) {
	// Build dict
	dict := make([]byte, 32*1024*1024) // 32 KB dicionnary, way overkill
	dict, err := TrainFromData(dict, [][]byte{
		[]byte("Hello nice world!"),
		[]byte("It's a very nice world!"),
	})
	if err != nil {
		t.Fatalf("Failed creating dict: %s", err)
	}
	// Simple
	testCompressionDecompression(t, dict, []byte("Hello world!"))
	// Long
	var long bytes.Buffer
	for i := 0; i < 10000; i++ {
		long.Write([]byte("Hellow World!"))
	}
	testCompressionDecompression(t, dict, long.Bytes())
}

func TestStreamRealPayload(t *testing.T) {
	if raw == nil {
		t.Skip(ErrNoPayloadEnv)
	}
	testCompressionDecompression(t, nil, raw)
}

func BenchmarkStreamCompression(b *testing.B) {
	if raw == nil {
		b.Fatal(ErrNoPayloadEnv)
	}
	var intermediate bytes.Buffer
	w := NewWriter(&intermediate, nil, 5)
	defer w.Close()
	b.SetBytes(int64(len(raw)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := w.Write(raw)
		if err != nil {
			b.Fatalf("Failed writing to compress object: %s", err)
		}
	}
}

func BenchmarkStreamDecompression(b *testing.B) {
	if raw == nil {
		b.Fatal(ErrNoPayloadEnv)
	}
	compressed, err := Compress(nil, raw)
	if err != nil {
		b.Fatalf("Failed to compress: %s", err)
	}
	_, err = Decompress(nil, compressed)
	if err != nil {
		b.Fatalf("Problem: %s", err)
	}

	dst := make([]byte, len(raw))
	b.SetBytes(int64(len(raw)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := bytes.NewReader(compressed)
		r := NewReader(rr, nil)
		_, err := r.Read(dst)
		if err != nil {
			b.Fatalf("Failed to decompress: %s", err)
		}
	}
}
