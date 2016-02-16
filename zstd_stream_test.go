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
	t.Logf("Compressed payload: %v %v", out, w.Bytes())
	rr := bytes.NewReader(out)
	// Decompress
	r := NewReader(rr, dict)
	dst := make([]byte, len(payload)+10)
	n, err := r.Read(dst)
	if err != io.EOF && n != 0 {
		failOnError(t, "Failed to read for decompression", err)
	}
	dst = dst[:n]
	if string(payload) != string(dst) {
		t.Fatalf("Cannot compress and decompress: %s != %s", payload, dst)
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
