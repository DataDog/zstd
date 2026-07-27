package zstd

import (
	"bytes"
	"testing"
)

// TestDecompressSizeHintUnknownIsMultipleOfInput verifies that when the frame
// does not advertise its size, the hint is a small multiple of the input rather
// than the flat upper bound.
func TestDecompressSizeHintUnknownIsMultipleOfInput(t *testing.T) {
	payload := bytes.Repeat([]byte("datadog-"), 525)
	frame := unknownSizeFrame(t, payload)

	hint, found := decompressSizeHint(frame)
	if found {
		t.Fatal("streaming frame should not advertise its size")
	}
	if want := 3 * len(frame); hint != want {
		t.Fatalf("unknown-size hint = %d, want %d (3x input)", hint, want)
	}
}

// TestDecompressNilBufferUnknownSizeAvoidsBound verifies that decompressing an
// unknown-size frame with no caller buffer does not allocate the flat upper
// bound (decompressSizeBufferLimit).
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
