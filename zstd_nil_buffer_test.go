package zstd

import (
	"bytes"
	"testing"
)

// TestDecompressNilBufferUnknownSizeAvoidsBound verifies that decompressing an
// unknown-size frame with no caller buffer grows from a small size instead of
// allocating decompressSizeBufferLimit (the pessimistic upper bound).
func TestDecompressNilBufferUnknownSizeAvoidsBound(t *testing.T) {
	payload := bytes.Repeat([]byte("datadog-"), 525) // 4200 bytes, well under 1 MB
	frame := unknownSizeFrame(t, payload)

	for _, d := range decompressors() {
		t.Run(d.name, func(t *testing.T) {
			out, err := d.fn(nil, frame)
			if err != nil {
				t.Fatalf("decompress: %v", err)
			}
			if !bytes.Equal(out, payload) {
				t.Fatalf("round-trip mismatch")
			}
			if cap(out) >= decompressSizeBufferLimit {
				t.Fatalf("nil-buffer unknown-size decode allocated the bound (cap %d)", cap(out))
			}
		})
	}
}

// TestDecompressNilBufferKnownSizeExact verifies the known-size path is
// unaffected: a nil buffer is still sized to the exact content size.
func TestDecompressNilBufferKnownSizeExact(t *testing.T) {
	payload := bytes.Repeat([]byte("datadog-"), 525)
	frame, err := Compress(nil, payload)
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}

	for _, d := range decompressors() {
		t.Run(d.name, func(t *testing.T) {
			out, err := d.fn(nil, frame)
			if err != nil {
				t.Fatalf("decompress: %v", err)
			}
			if !bytes.Equal(out, payload) {
				t.Fatalf("round-trip mismatch")
			}
			if cap(out) != len(payload) {
				t.Fatalf("known-size nil decode should allocate exactly: cap %d want %d", cap(out), len(payload))
			}
		})
	}
}
