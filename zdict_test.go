package zstd

import (
	"bytes"
	"testing"
)

// Generate sample data that has repeated patterns suitable for dictionary training
func generateSamples() [][]byte {
	samples := make([][]byte, 100)
	patterns := []string{
		`{"name": "user_`, `", "email": "user`, `@example.com", "age": `,
		`, "city": "New York", "country": "USA"}`,
		`{"type": "log", "level": "info", "message": "`,
		`", "timestamp": "2024-01-01T00:00:00Z"}`,
		`{"product": "item_`, `", "price": `, `, "quantity": `,
		`, "category": "electronics"}`,
	}

	for i := 0; i < len(samples); i++ {
		// Create samples with repetitive structure
		var buf bytes.Buffer
		patternIdx := i % 3
		switch patternIdx {
		case 0:
			buf.WriteString(patterns[0])
			buf.WriteString(string(rune('A' + (i % 26))))
			buf.WriteString(patterns[1])
			buf.WriteString(string(rune('A' + (i % 26))))
			buf.WriteString(patterns[2])
			buf.WriteString(string(rune('0' + (i % 10))))
			buf.WriteString(patterns[3])
		case 1:
			buf.WriteString(patterns[4])
			buf.WriteString("Test message ")
			buf.WriteString(string(rune('0' + (i % 10))))
			buf.WriteString(patterns[5])
		case 2:
			buf.WriteString(patterns[6])
			buf.WriteString(string(rune('A' + (i % 26))))
			buf.WriteString(patterns[7])
			buf.WriteString(string(rune('0' + (i % 10))))
			buf.WriteString(patterns[8])
			buf.WriteString(string(rune('0' + (i % 10))))
			buf.WriteString(patterns[9])
		}
		samples[i] = buf.Bytes()
	}

	return samples
}

func TestTrainFromBuffer(t *testing.T) {
	samples := generateSamples()

	dict, err := TrainFromBuffer(samples, 4096)
	if err != nil {
		t.Fatalf("Failed to train dictionary: %v", err)
	}

	if len(dict) == 0 {
		t.Fatal("Dictionary is empty")
	}

	t.Logf("Trained dictionary size: %d bytes", len(dict))

	// Verify dictionary can be used for compression/decompression
	testDictionary(t, dict, samples[0])
}

func TestTrainFromBufferEmptySamples(t *testing.T) {
	_, err := TrainFromBuffer([][]byte{}, 4096)
	if err != ErrNotEnoughSamples {
		t.Fatalf("Expected ErrNotEnoughSamples, got: %v", err)
	}
}

func TestTrainFromBufferCover(t *testing.T) {
	samples := generateSamples()

	params := CoverParams{
		K: 32,
		D: 8,
	}

	dict, err := TrainFromBufferCover(samples, 4096, params)
	if err != nil {
		t.Fatalf("Failed to train dictionary with COVER: %v", err)
	}

	if len(dict) == 0 {
		t.Fatal("Dictionary is empty")
	}

	t.Logf("Trained COVER dictionary size: %d bytes", len(dict))
	testDictionary(t, dict, samples[0])
}

func TestOptimizeTrainFromBufferCover(t *testing.T) {
	samples := generateSamples()

	params := CoverParams{
		K:     0, // Will be optimized
		D:     0, // Will be optimized
		Steps: 4, // Use fewer steps for faster testing
	}

	dict, optimizedParams, err := OptimizeTrainFromBufferCover(samples, 4096, params)
	if err != nil {
		t.Fatalf("Failed to optimize and train dictionary with COVER: %v", err)
	}

	if len(dict) == 0 {
		t.Fatal("Dictionary is empty")
	}

	t.Logf("Optimized COVER dictionary size: %d bytes", len(dict))
	t.Logf("Optimized parameters: K=%d, D=%d", optimizedParams.K, optimizedParams.D)

	if optimizedParams.K == 0 || optimizedParams.D == 0 {
		t.Fatal("Parameters were not optimized")
	}

	testDictionary(t, dict, samples[0])
}

func TestTrainFromBufferFastCover(t *testing.T) {
	samples := generateSamples()

	params := FastCoverParams{
		K: 32,
		D: 8,
		F: 20,
	}

	dict, err := TrainFromBufferFastCover(samples, 4096, params)
	if err != nil {
		t.Fatalf("Failed to train dictionary with fastCover: %v", err)
	}

	if len(dict) == 0 {
		t.Fatal("Dictionary is empty")
	}

	t.Logf("Trained fastCover dictionary size: %d bytes", len(dict))
	testDictionary(t, dict, samples[0])
}

func TestOptimizeTrainFromBufferFastCover(t *testing.T) {
	samples := generateSamples()

	params := FastCoverParams{
		K:     0, // Will be optimized
		D:     0, // Will be optimized
		F:     20,
		Steps: 4, // Use fewer steps for faster testing
		Accel: 1,
	}

	dict, optimizedParams, err := OptimizeTrainFromBufferFastCover(samples, 4096, params)
	if err != nil {
		t.Fatalf("Failed to optimize and train dictionary with fastCover: %v", err)
	}

	if len(dict) == 0 {
		t.Fatal("Dictionary is empty")
	}

	t.Logf("Optimized fastCover dictionary size: %d bytes", len(dict))
	t.Logf("Optimized parameters: K=%d, D=%d, F=%d", optimizedParams.K, optimizedParams.D, optimizedParams.F)

	if optimizedParams.K == 0 || optimizedParams.D == 0 {
		t.Fatal("Parameters were not optimized")
	}

	testDictionary(t, dict, samples[0])
}

func TestFinalizeDictionary(t *testing.T) {
	samples := generateSamples()

	// Create a raw content dictionary (just some repeated bytes)
	rawContent := []byte("user_@example.com{\"name\": \"email\": \"age\": \"city\": \"country\": \"type\": \"log\"")

	params := DictParams{
		CompressionLevel: DefaultCompression,
	}

	dict, err := FinalizeDictionary(rawContent, samples, 4096, params)
	if err != nil {
		t.Fatalf("Failed to finalize dictionary: %v", err)
	}

	if len(dict) == 0 {
		t.Fatal("Dictionary is empty")
	}

	t.Logf("Finalized dictionary size: %d bytes", len(dict))
	testDictionary(t, dict, samples[0])
}

func TestFinalizeDictionaryEmptyContent(t *testing.T) {
	samples := generateSamples()

	params := DictParams{
		CompressionLevel: DefaultCompression,
	}

	// Empty dict content should work - it will just create entropy tables from samples
	dict, err := FinalizeDictionary([]byte{}, samples, 4096, params)
	if err != nil {
		t.Fatalf("Failed to finalize dictionary with empty content: %v", err)
	}

	if len(dict) == 0 {
		t.Fatal("Dictionary is empty")
	}

	t.Logf("Finalized dictionary (empty content) size: %d bytes", len(dict))
}

func TestGetDictID(t *testing.T) {
	samples := generateSamples()

	// Train a dictionary
	dict, err := TrainFromBuffer(samples, 4096)
	if err != nil {
		t.Fatalf("Failed to train dictionary: %v", err)
	}

	// Get the dictionary ID
	dictID := GetDictID(dict)
	if dictID == 0 {
		t.Fatal("Dictionary ID is 0 (invalid)")
	}

	t.Logf("Dictionary ID: %d", dictID)

	// Test with invalid dictionary
	invalidDictID := GetDictID([]byte("not a dictionary"))
	if invalidDictID != 0 {
		t.Fatalf("Expected 0 for invalid dictionary, got: %d", invalidDictID)
	}

	// Test with empty buffer
	emptyDictID := GetDictID([]byte{})
	if emptyDictID != 0 {
		t.Fatalf("Expected 0 for empty buffer, got: %d", emptyDictID)
	}
}

func TestGetDictHeaderSize(t *testing.T) {
	samples := generateSamples()

	dict, err := TrainFromBuffer(samples, 4096)
	if err != nil {
		t.Fatalf("Failed to train dictionary: %v", err)
	}

	headerSize, err := GetDictHeaderSize(dict)
	if err != nil {
		t.Fatalf("Failed to get dictionary header size: %v", err)
	}

	if headerSize <= 0 || headerSize > len(dict) {
		t.Fatalf("Invalid header size: %d (dict size: %d)", headerSize, len(dict))
	}

	t.Logf("Dictionary header size: %d bytes", headerSize)

	// Test with invalid dictionary
	_, err = GetDictHeaderSize([]byte("not a dictionary"))
	if err == nil {
		t.Fatal("Expected error for invalid dictionary")
	}

	// Test with empty buffer
	_, err = GetDictHeaderSize([]byte{})
	if err == nil {
		t.Fatal("Expected error for empty buffer")
	}
}

func TestCDict(t *testing.T) {
	samples := generateSamples()

	dict, err := TrainFromBuffer(samples, 4096)
	if err != nil {
		t.Fatalf("Failed to train dictionary: %v", err)
	}

	// Create CDict
	cdict, err := NewCDict(dict, DefaultCompression)
	if err != nil {
		t.Fatalf("Failed to create CDict: %v", err)
	}
	defer cdict.Close()

	// Test compression with dictionary
	input := samples[0]
	compressed, err := cdict.Compress(nil, input)
	if err != nil {
		t.Fatalf("Failed to compress with CDict: %v", err)
	}

	if len(compressed) == 0 {
		t.Fatal("Compressed data is empty")
	}

	t.Logf("Original size: %d, Compressed size: %d", len(input), len(compressed))

	// Verify decompression works
	ddict, err := NewDDict(dict)
	if err != nil {
		t.Fatalf("Failed to create DDict: %v", err)
	}
	defer ddict.Close()

	decompressed, err := ddict.Decompress(nil, compressed)
	if err != nil {
		t.Fatalf("Failed to decompress with DDict: %v", err)
	}

	if !bytes.Equal(input, decompressed) {
		t.Fatal("Decompressed data does not match original")
	}
}

func TestCDictWithBuffer(t *testing.T) {
	samples := generateSamples()

	dict, err := TrainFromBuffer(samples, 4096)
	if err != nil {
		t.Fatalf("Failed to train dictionary: %v", err)
	}

	cdict, err := NewCDict(dict, DefaultCompression)
	if err != nil {
		t.Fatalf("Failed to create CDict: %v", err)
	}
	defer cdict.Close()

	input := samples[0]

	// Test with pre-allocated buffer
	buf := make([]byte, CompressBound(len(input)))
	compressed, err := cdict.Compress(buf, input)
	if err != nil {
		t.Fatalf("Failed to compress with CDict: %v", err)
	}

	if len(compressed) == 0 {
		t.Fatal("Compressed data is empty")
	}

	// Verify buffer was reused (same underlying array)
	if cap(compressed) != cap(buf) {
		t.Fatal("Buffer was not reused")
	}
}

func TestCDictEmpty(t *testing.T) {
	_, err := NewCDict([]byte{}, DefaultCompression)
	if err == nil {
		t.Fatal("Expected error for empty dictionary")
	}
}

func TestDDict(t *testing.T) {
	samples := generateSamples()

	dict, err := TrainFromBuffer(samples, 4096)
	if err != nil {
		t.Fatalf("Failed to train dictionary: %v", err)
	}

	cdict, err := NewCDict(dict, DefaultCompression)
	if err != nil {
		t.Fatalf("Failed to create CDict: %v", err)
	}
	defer cdict.Close()

	ddict, err := NewDDict(dict)
	if err != nil {
		t.Fatalf("Failed to create DDict: %v", err)
	}
	defer ddict.Close()

	// Compress and decompress multiple samples
	for i, sample := range samples[:10] {
		compressed, err := cdict.Compress(nil, sample)
		if err != nil {
			t.Fatalf("Failed to compress sample %d: %v", i, err)
		}

		decompressed, err := ddict.Decompress(nil, compressed)
		if err != nil {
			t.Fatalf("Failed to decompress sample %d: %v", i, err)
		}

		if !bytes.Equal(sample, decompressed) {
			t.Fatalf("Sample %d: decompressed data does not match original", i)
		}
	}
}

func TestDDictEmpty(t *testing.T) {
	_, err := NewDDict([]byte{})
	if err == nil {
		t.Fatal("Expected error for empty dictionary")
	}
}

func TestDDictEmptyInput(t *testing.T) {
	samples := generateSamples()

	dict, err := TrainFromBuffer(samples, 4096)
	if err != nil {
		t.Fatalf("Failed to train dictionary: %v", err)
	}

	ddict, err := NewDDict(dict)
	if err != nil {
		t.Fatalf("Failed to create DDict: %v", err)
	}
	defer ddict.Close()

	_, err = ddict.Decompress(nil, []byte{})
	if err != ErrEmptySlice {
		t.Fatalf("Expected ErrEmptySlice, got: %v", err)
	}
}

func TestDictionaryCompressionRatio(t *testing.T) {
	samples := generateSamples()

	// Train dictionary
	dict, err := TrainFromBuffer(samples, 4096)
	if err != nil {
		t.Fatalf("Failed to train dictionary: %v", err)
	}

	cdict, err := NewCDict(dict, DefaultCompression)
	if err != nil {
		t.Fatalf("Failed to create CDict: %v", err)
	}
	defer cdict.Close()

	// Compare compression with and without dictionary
	testData := samples[0]

	// Without dictionary
	compressedNormal, err := Compress(nil, testData)
	if err != nil {
		t.Fatalf("Failed to compress without dictionary: %v", err)
	}

	// With dictionary
	compressedWithDict, err := cdict.Compress(nil, testData)
	if err != nil {
		t.Fatalf("Failed to compress with dictionary: %v", err)
	}

	t.Logf("Original size: %d", len(testData))
	t.Logf("Compressed without dictionary: %d", len(compressedNormal))
	t.Logf("Compressed with dictionary: %d", len(compressedWithDict))
	t.Logf("Dictionary improvement: %d bytes (%.1f%%)",
		len(compressedNormal)-len(compressedWithDict),
		100.0*float64(len(compressedNormal)-len(compressedWithDict))/float64(len(compressedNormal)))

	// Dictionary should typically improve compression for similar data
	// (though not guaranteed for all cases)
}

func TestCDictClose(t *testing.T) {
	samples := generateSamples()

	dict, err := TrainFromBuffer(samples, 4096)
	if err != nil {
		t.Fatalf("Failed to train dictionary: %v", err)
	}

	cdict, err := NewCDict(dict, DefaultCompression)
	if err != nil {
		t.Fatalf("Failed to create CDict: %v", err)
	}

	// Close should be idempotent
	cdict.Close()
	cdict.Close()
}

func TestDDictClose(t *testing.T) {
	samples := generateSamples()

	dict, err := TrainFromBuffer(samples, 4096)
	if err != nil {
		t.Fatalf("Failed to train dictionary: %v", err)
	}

	ddict, err := NewDDict(dict)
	if err != nil {
		t.Fatalf("Failed to create DDict: %v", err)
	}

	// Close should be idempotent
	ddict.Close()
	ddict.Close()
}

func TestDictParamsWithCustomID(t *testing.T) {
	samples := generateSamples()

	params := DictParams{
		CompressionLevel: DefaultCompression,
		DictID:           12345,
	}

	rawContent := []byte("test content")
	dict, err := FinalizeDictionary(rawContent, samples, 4096, params)
	if err != nil {
		t.Fatalf("Failed to finalize dictionary: %v", err)
	}

	dictID := GetDictID(dict)
	if dictID != 12345 {
		t.Fatalf("Expected dict ID 12345, got: %d", dictID)
	}
}

// testDictionary is a helper function to verify a dictionary works
func testDictionary(t *testing.T, dict []byte, testData []byte) {
	t.Helper()

	// Create CDict and DDict
	cdict, err := NewCDict(dict, DefaultCompression)
	if err != nil {
		t.Fatalf("Failed to create CDict: %v", err)
	}
	defer cdict.Close()

	ddict, err := NewDDict(dict)
	if err != nil {
		t.Fatalf("Failed to create DDict: %v", err)
	}
	defer ddict.Close()

	// Compress
	compressed, err := cdict.Compress(nil, testData)
	if err != nil {
		t.Fatalf("Failed to compress: %v", err)
	}

	// Decompress
	decompressed, err := ddict.Decompress(nil, compressed)
	if err != nil {
		t.Fatalf("Failed to decompress: %v", err)
	}

	// Verify
	if !bytes.Equal(testData, decompressed) {
		t.Fatal("Decompressed data does not match original")
	}
}

func BenchmarkTrainFromBuffer(b *testing.B) {
	samples := generateSamples()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := TrainFromBuffer(samples, 4096)
		if err != nil {
			b.Fatalf("Failed to train dictionary: %v", err)
		}
	}
}

func BenchmarkCompressWithDict(b *testing.B) {
	samples := generateSamples()
	dict, err := TrainFromBuffer(samples, 4096)
	if err != nil {
		b.Fatalf("Failed to train dictionary: %v", err)
	}

	cdict, err := NewCDict(dict, DefaultCompression)
	if err != nil {
		b.Fatalf("Failed to create CDict: %v", err)
	}
	defer cdict.Close()

	testData := samples[0]
	dst := make([]byte, CompressBound(len(testData)))

	b.SetBytes(int64(len(testData)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := cdict.Compress(dst, testData)
		if err != nil {
			b.Fatalf("Failed to compress: %v", err)
		}
	}
}

func BenchmarkDecompressWithDict(b *testing.B) {
	samples := generateSamples()
	dict, err := TrainFromBuffer(samples, 4096)
	if err != nil {
		b.Fatalf("Failed to train dictionary: %v", err)
	}

	cdict, err := NewCDict(dict, DefaultCompression)
	if err != nil {
		b.Fatalf("Failed to create CDict: %v", err)
	}
	defer cdict.Close()

	ddict, err := NewDDict(dict)
	if err != nil {
		b.Fatalf("Failed to create DDict: %v", err)
	}
	defer ddict.Close()

	testData := samples[0]
	compressed, err := cdict.Compress(nil, testData)
	if err != nil {
		b.Fatalf("Failed to compress: %v", err)
	}

	dst := make([]byte, len(testData))

	b.SetBytes(int64(len(testData)))
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := ddict.Decompress(dst, compressed)
		if err != nil {
			b.Fatalf("Failed to decompress: %v", err)
		}
	}
}
