package zstd

import (
	"bytes"
	"testing"
)

// unknownSizeFrame returns a zstd frame that does not advertise its decompressed
// size in the header (as produced by the streaming writer, and by legacy zstd
// v0.5 frames), verifying that is the case so callers exercise the intended path.
func unknownSizeFrame(t *testing.T, payload []byte) []byte {
	t.Helper()
	var b bytes.Buffer
	w := NewWriter(&b)
	if _, err := w.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	frame := b.Bytes()
	if _, found := decompressSizeHint(frame); found {
		t.Fatal("streaming frame unexpectedly advertises its size")
	}
	return frame
}

type namedDecompressor struct {
	name string
	fn   func(dst, src []byte) ([]byte, error)
}

// decompressors are the entry points that share the caller-buffer-reuse logic.
// ctx.Decompress uses a fresh context per call so tests don't share state.
func decompressors() []namedDecompressor {
	return []namedDecompressor{
		{"Decompress", Decompress},
		{"ctx.Decompress", func(dst, src []byte) ([]byte, error) { return NewCtx().Decompress(dst, src) }},
	}
}

// sameBuffer reports whether out was decoded into buf's backing array.
func sameBuffer(out, buf []byte) bool {
	return len(out) > 0 && len(buf) > 0 && &out[0] == &buf[0]
}

// TestDecompressReusesCallerBufferUnknownSize verifies that for an unknown-size
// frame, an adequate caller buffer is decoded into rather than discarded in
// favour of allocating decompressSizeBufferLimit.
func TestDecompressReusesCallerBufferUnknownSize(t *testing.T) {
	payload := bytes.Repeat([]byte("datadog-"), 525) // 4200 bytes
	frame := unknownSizeFrame(t, payload)

	for _, d := range decompressors() {
		t.Run(d.name, func(t *testing.T) {
			buf := make([]byte, 8192) // adequate for the payload, far below the bound
			out, err := d.fn(buf, frame)
			if err != nil {
				t.Fatalf("decompress: %v", err)
			}
			if !bytes.Equal(out, payload) {
				t.Fatalf("round-trip mismatch")
			}
			if !sameBuffer(out, buf) {
				t.Fatalf("caller buffer should have been reused")
			}
		})
	}
}

// TestDecompressUnknownSizeTooSmallBuffer verifies the fallback still works when
// the caller buffer is too small for an unknown-size frame.
func TestDecompressUnknownSizeTooSmallBuffer(t *testing.T) {
	payload := bytes.Repeat([]byte("datadog-"), 525)
	frame := unknownSizeFrame(t, payload)

	for _, d := range decompressors() {
		t.Run(d.name, func(t *testing.T) {
			out, err := d.fn(make([]byte, 8), frame) // too small; must fall back
			if err != nil {
				t.Fatalf("decompress: %v", err)
			}
			if !bytes.Equal(out, payload) {
				t.Fatalf("round-trip mismatch on fallback")
			}
		})
	}
}

// TestDecompressUnknownSizeNilBuffer verifies nil dst still decompresses.
func TestDecompressUnknownSizeNilBuffer(t *testing.T) {
	payload := bytes.Repeat([]byte("datadog-"), 525)
	frame := unknownSizeFrame(t, payload)

	for _, d := range decompressors() {
		t.Run(d.name, func(t *testing.T) {
			out, err := d.fn(nil, frame)
			if err != nil {
				t.Fatalf("decompress: %v", err)
			}
			if !bytes.Equal(out, payload) {
				t.Fatalf("round-trip mismatch on nil dst")
			}
		})
	}
}

// TestDecompressKnownSizeReusesCallerBuffer verifies that for frames that do
// advertise their size, an adequate caller buffer is still reused.
func TestDecompressKnownSizeReusesCallerBuffer(t *testing.T) {
	payload := bytes.Repeat([]byte("datadog-"), 525)
	frame, err := Compress(nil, payload)
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}

	for _, d := range decompressors() {
		t.Run(d.name, func(t *testing.T) {
			buf := make([]byte, 8192)
			out, err := d.fn(buf, frame)
			if err != nil {
				t.Fatalf("decompress: %v", err)
			}
			if !bytes.Equal(out, payload) {
				t.Fatalf("round-trip mismatch")
			}
			if !sameBuffer(out, buf) {
				t.Fatalf("caller buffer should have been reused")
			}
		})
	}
}
