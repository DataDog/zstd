package zstd

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

var raw []byte
var (
	ErrNoPayloadEnv = errors.New("PAYLOAD env was not set")
)

func init() {
	var err error
	payload := os.Getenv("PAYLOAD")
	if len(payload) > 0 {
		raw, err = ioutil.ReadFile(payload)
		if err != nil {
			fmt.Printf("Error opening payload: %s\n", err)
		}
	}
}

// Test our version of compress bound vs C implementation
func TestCompressBound(t *testing.T) {
	tests := []int{0, 1, 2, 10, 456, 15468, 1313, 512, 2147483632}
	for _, test := range tests {
		if CompressBound(test) != cCompressBound(test) {
			t.Fatalf("For %v, results are different: %v (actual) != %v (expected)", test,
				CompressBound(test), cCompressBound(test))
		}
	}
}

// Test error code
func TestErrorCode(t *testing.T) {
	tests := make([]int, 211)
	for i := 0; i < len(tests); i++ {
		tests[i] = i - 105
	}
	for _, test := range tests {
		err := getError(test)
		if err == nil && cIsError(test) {
			t.Fatalf("C function returned error for %v but ours did not", test)
		} else if err != nil && !cIsError(test) {
			t.Fatalf("Ours function returned error for %v but C one did not", test)
		}
	}

}

// Test compression
func TestCompressDecompress(t *testing.T) {
	input := []byte("Hello World!")
	out, err := Compress(nil, input)
	if err != nil {
		t.Fatalf("Error while compressing: %v", err)
	}
	out2 := make([]byte, 1000)
	out2, err = Compress(out2, input)
	if err != nil {
		t.Fatalf("Error while compressing: %v", err)
	}
	t.Logf("Compressed: %v", out)
	rein, err := Decompress(nil, out)
	if err != nil {
		t.Fatalf("Error while decompressing: %v", err)
	}
	rein2 := make([]byte, 10)
	rein2, err = Decompress(rein2, out2)
	if err != nil {
		t.Fatalf("Error while decompressing: %v", err)
	}

	if string(input) != string(rein) {
		t.Fatalf("Cannot compress and decompress: %s != %s", input, rein)
	}
	if string(input) != string(rein2) {
		t.Fatalf("Cannot compress and decompress: %s != %s", input, rein)
	}
}

func TestCompressLevel(t *testing.T) {
	inputs := [][]byte{
		nil, {}, {0}, []byte("Hello World!"),
	}

	for _, input := range inputs {
		for level := BestSpeed; level <= BestCompression; level++ {
			out, err := CompressLevel(nil, input, level)
			if err != nil {
				t.Errorf("input=%#v level=%d CompressLevel failed err=%s", string(input), level, err.Error())
				continue
			}

			orig, err := Decompress(nil, out)
			if err != nil {
				t.Errorf("input=%#v level=%d Decompress failed err=%s", string(input), level, err.Error())
				continue
			}
			if !bytes.Equal(orig, input) {
				t.Errorf("input=%#v level=%d orig does not match: %#v", string(input), level, string(orig))
			}
		}
	}
}

func doCompressLevel(payload []byte, out []byte) error {
	out, err := CompressLevel(out, payload, DefaultCompression)
	if err != nil {
		return fmt.Errorf("failed calling CompressLevel: %w", err)
	}
	if len(out) == 0 {
		return errors.New("CompressLevel must return non-empty bytes")
	}
	return nil
}

func useStackSpaceCompressLevel(payload []byte, out []byte, level int) error {
	if level == 0 {
		return doCompressLevel(payload, out)
	}
	return useStackSpaceCompressLevel(payload, out, level-1)
}

func TestCompressLevelStackCgoBug(t *testing.T) {
	// CompressLevel previously had a bug where it would access the wrong pointer
	// This test would crash when run with CGODEBUG=efence=1 go test .
	const maxStackLevels = 100

	payload := []byte("Hello World!")
	// allocate the output buffer so CompressLevel does not allocate it
	out := make([]byte, CompressBound(len(payload)))

	for level := 0; level < maxStackLevels; level++ {
		err := useStackSpaceCompressLevel(payload, out, level)
		if err != nil {
			t.Fatal("CompressLevel failed:", err)
		}
	}
}

func TestEmptySliceCompress(t *testing.T) {
	compressed, err := Compress(nil, []byte{})
	if err != nil {
		t.Fatalf("Error while compressing: %v", err)
	}
	t.Logf("Compressing empty slice gives 0x%x", compressed)
	decompressed, err := Decompress(nil, compressed)
	if err != nil {
		t.Fatalf("Error while compressing: %v", err)
	}
	if string(decompressed) != "" {
		t.Fatalf("Expected empty slice as decompressed, got %v instead", decompressed)
	}
}

func TestEmptySliceDecompress(t *testing.T) {
	_, err := Decompress(nil, []byte{})
	if err != ErrEmptySlice {
		t.Fatalf("Did not get the correct error: %s", err)
	}
}

func TestDecompressZeroLengthBuf(t *testing.T) {
	input := []byte("Hello World!")
	out, err := Compress(nil, input)
	if err != nil {
		t.Fatalf("Error while compressing: %v", err)
	}

	buf := make([]byte, 0)
	decompressed, err := Decompress(buf, out)
	if err != nil {
		t.Fatalf("Error while decompressing: %v", err)
	}

	if res, exp := string(input), string(decompressed); res != exp {
		t.Fatalf("expected %s but decompressed to %s", exp, res)
	}
}

func TestTooSmall(t *testing.T) {
	var long bytes.Buffer
	for i := 0; i < 10000; i++ {
		long.Write([]byte("Hellow World!"))
	}
	input := long.Bytes()
	out, err := Compress(nil, input)
	if err != nil {
		t.Fatalf("Error while compressing: %v", err)
	}
	rein := make([]byte, 1)
	// This should switch to the decompression stream to handle too small dst
	rein, err = Decompress(rein, out)
	if err != nil {
		t.Fatalf("Failed decompressing: %s", err)
	}
	if string(input) != string(rein) {
		t.Fatalf("Cannot compress and decompress: %s != %s", input, rein)
	}
}

func TestRealPayload(t *testing.T) {
	if raw == nil {
		t.Skip(ErrNoPayloadEnv)
	}
	dst, err := Compress(nil, raw)
	if err != nil {
		t.Fatalf("Failed to compress: %s", err)
	}
	rein, err := Decompress(nil, dst)
	if err != nil {
		t.Fatalf("Failed to decompress: %s", err)
	}
	if string(raw) != string(rein) {
		t.Fatalf("compressed/decompressed payloads are not the same (lengths: %v & %v)", len(raw), len(rein))
	}
}

func BenchmarkCompression(b *testing.B) {
	if raw == nil {
		b.Fatal(ErrNoPayloadEnv)
	}
	dst := make([]byte, CompressBound(len(raw)))
	b.SetBytes(int64(len(raw)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Compress(dst, raw)
		if err != nil {
			b.Fatalf("Failed compressing: %s", err)
		}
	}
}

func BenchmarkDecompression(b *testing.B) {
	if raw == nil {
		b.Fatal(ErrNoPayloadEnv)
	}
	src := make([]byte, len(raw))
	dst, err := Compress(nil, raw)
	if err != nil {
		b.Fatalf("Failed compressing: %s", err)
	}
	b.Logf("Reduced from %v to %v", len(raw), len(dst))
	b.SetBytes(int64(len(raw)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src2, err := Decompress(src, dst)
		if err != nil {
			b.Fatalf("Failed decompressing: %s", err)
		}
		b.StopTimer()
		if !bytes.Equal(raw, src2) {
			b.Fatalf("Results are not the same: %v != %v", len(raw), len(src2))
		}
		b.StartTimer()
	}
}
