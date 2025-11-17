package zstd

import (
	"bytes"
	"fmt"
	"os"
	"strings"
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

func TestPerf(t *testing.T) {

	trainingRecordsCount := 1000
	testRecordsCount := 1000

	// Large word pool for more realistic variable content
	words := []string{
		"system", "error", "warning", "info", "debug", "critical", "fatal", "trace",
		"process", "thread", "memory", "disk", "network", "cpu", "cache", "queue",
		"request", "response", "timeout", "retry", "failed", "success", "pending", "complete",
		"user", "session", "authentication", "authorization", "permission", "access", "denied",
		"database", "query", "transaction", "commit", "rollback", "connection", "pool",
		"service", "endpoint", "handler", "middleware", "controller", "model", "view",
		"started", "stopped", "restarted", "initializing", "terminated", "crashed", "recovered",
		"upstream", "downstream", "latency", "throughput", "bandwidth", "packet", "frame",
		"encrypted", "decrypted", "signed", "verified", "hashed", "encoded", "decoded",
		"allocated", "deallocated", "garbage", "collected", "leaked", "freed", "reserved",
		"synchronized", "locked", "unlocked", "blocked", "waiting", "running", "sleeping",
		"registered", "unregistered", "subscribed", "unsubscribed", "published", "consumed",
		"validated", "rejected", "accepted", "processed", "queued", "dispatched", "delivered",
	}

	// Helper to generate pseudo-random UUIDs
	randomUUID := func(seed int) string {
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			seed*12345, (seed*678)%65536, (seed*910)%65536,
			(seed*1112)%65536, seed*131415)
	}

	// Helper to generate pseudo-random hex hash
	randomHash := func(seed int) string {
		return fmt.Sprintf("%064x", seed*271828182845904523)
	}

	// Helper to generate pseudo-random base64
	randomBase64 := func(seed int) string {
		// Simulate base64 encoded data
		chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
		result := ""
		for i := 0; i < 32; i++ {
			result += string(chars[(seed*i*7919)%len(chars)])
		}
		return result
	}

	// Helper to create a record
	makeRecord := func(id int, seedOffset int) string {
		// Static fields (1/3 of size) - these compress extremely well with dictionary
		staticPart := fmt.Sprintf(`{
  "user_id": %d,
  "action": "page_view",
  "timestamp": "2024-01-01T%02d:00:00Z",
  "session_id": "abc123def456ghi789",
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
  "referrer": "https://example.com/landing",
  "page_url": "https://example.com/product/12345",
  "page_title": "Product Details - Example Store",
  "country": "United States",
  "region": "California",
  "city": "San Francisco",
  "device_type": "desktop",
  "browser": "Chrome",
  "os": "Windows",`, id, id%24)

		// Variable message field (2/3 of size) - random words, UUIDs, hashes
		var messageParts []string
		for j := 0; j < 10; j++ {
			messageParts = append(messageParts, words[(id+seedOffset+j*7)%len(words)])
		}
		message := fmt.Sprintf("%s uuid=%s hash=%s data=%s",
			strings.Join(messageParts, " "),
			randomUUID(id+seedOffset),
			randomHash(id+seedOffset),
			randomBase64(id+seedOffset))

		return staticPart + fmt.Sprintf(`
  "message": "%s",
  "request_id": "%s",
  "trace_id": "%s",
  "status": "success"
}`, message, randomUUID(id+seedOffset+1), randomUUID(id+seedOffset+2))
	}

	// Training set: 10k records with seed offset 0
	trainingRecords := make([]string, trainingRecordsCount)
	for i := 0; i < trainingRecordsCount; i++ {
		trainingRecords[i] = makeRecord(i, 0)
	}

	// Test set: Different 1k records with seed offset 50000 (completely different UUIDs/hashes)
	testRecords := make([]string, testRecordsCount)
	for i := 0; i < testRecordsCount; i++ {
		testRecords[i] = makeRecord(i, 50000) // Different seed = different UUIDs/hashes
	}

	// Step 1: Train dictionary on TRAINING set
	trainingSamples := make([][]byte, len(trainingRecords))
	for i, record := range trainingRecords {
		trainingSamples[i] = []byte(record)
	}
	dict, _ := TrainFromBuffer(trainingSamples, 100*1024) // Larger dict for more patterns

	// Step 2: Test on DIFFERENT test set to prove dictionary generalizes
	// Create test array
	var testArray string
	testArray = "["
	for i, record := range testRecords {
		if i > 0 {
			testArray += ","
		}
		testArray += record
	}
	testArray += "]"
	testArrayBytes := []byte(testArray)

	// Step 3: Compress test data with dictionary (never seen this data before!)
	cdict, _ := NewCDict(dict, DefaultCompression)
	defer cdict.Close()
	compressedWithDict, _ := cdict.Compress(nil, testArrayBytes)

	// Compare with no dictionary on same test data
	compressedNoDict, _ := Compress(nil, testArrayBytes)

	// Calculate metrics
	originalSize := len(testArrayBytes)
	noDictSize := len(compressedNoDict)
	withDictSize := len(compressedWithDict)
	savings := noDictSize - withDictSize

	noDictRatio := float64(originalSize) / float64(noDictSize)
	withDictRatio := float64(originalSize) / float64(withDictSize)
	improvement := float64(savings) / float64(noDictSize) * 100

	// Show results
	t.Logf("\n=== Dictionary Compression Test ===")
	t.Logf("Training: %d records | Testing: %d records (unseen data)", len(trainingRecords), len(testRecords))
	t.Logf("Dictionary size: %d KB", len(dict)/1024)
	t.Logf("\nOriginal size:           %10d bytes", originalSize)
	t.Logf("Without dictionary:      %10d bytes  (%.2fx compression)", noDictSize, noDictRatio)
	t.Logf("With dictionary:         %10d bytes  (%.2fx compression)", withDictSize, withDictRatio)
	t.Logf("\nImprovement:             %10d bytes saved (%.1f%% smaller)", savings, improvement)

	// Step 4: Decompress and verify
	ddict, _ := NewDDict(dict)
	defer ddict.Close()
	decompressed, _ := ddict.Decompress(nil, compressedWithDict)

	if string(decompressed) != testArray {
		t.Fatal("Round-trip failed - decompressed data doesn't match original")
	}
	t.Logf("\n✓ Round-trip verified on unseen data")

}

func TestDictOptimization(t *testing.T) {
	// Reuse data generation from TestPerf
	words := []string{
		"system", "error", "warning", "info", "debug", "critical", "fatal", "trace",
		"process", "thread", "memory", "disk", "network", "cpu", "cache", "queue",
		"request", "response", "timeout", "retry", "failed", "success", "pending", "complete",
		"user", "session", "authentication", "authorization", "permission", "access", "denied",
		"database", "query", "transaction", "commit", "rollback", "connection", "pool",
		"service", "endpoint", "handler", "middleware", "controller", "model", "view",
		"started", "stopped", "restarted", "initializing", "terminated", "crashed", "recovered",
		"upstream", "downstream", "latency", "throughput", "bandwidth", "packet", "frame",
		"encrypted", "decrypted", "signed", "verified", "hashed", "encoded", "decoded",
		"allocated", "deallocated", "garbage", "collected", "leaked", "freed", "reserved",
		"synchronized", "locked", "unlocked", "blocked", "waiting", "running", "sleeping",
		"registered", "unregistered", "subscribed", "unsubscribed", "published", "consumed",
		"validated", "rejected", "accepted", "processed", "queued", "dispatched", "delivered",
	}

	randomUUID := func(seed int) string {
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			seed*12345, (seed*678)%65536, (seed*910)%65536,
			(seed*1112)%65536, seed*131415)
	}

	randomHash := func(seed int) string {
		return fmt.Sprintf("%064x", seed*271828182845904523)
	}

	randomBase64 := func(seed int) string {
		chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
		result := ""
		for i := 0; i < 32; i++ {
			result += string(chars[(seed*i*7919)%len(chars)])
		}
		return result
	}

	makeRecord := func(id int, seedOffset int) string {
		staticPart := fmt.Sprintf(`{
  "user_id": %d,
  "action": "page_view",
  "timestamp": "2024-01-01T%02d:00:00Z",
  "session_id": "abc123def456ghi789",
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
  "referrer": "https://example.com/landing",
  "page_url": "https://example.com/product/12345",
  "page_title": "Product Details - Example Store",
  "country": "United States",
  "region": "California",
  "city": "San Francisco",
  "device_type": "desktop",
  "browser": "Chrome",
  "os": "Windows",`, id, id%24)

		var messageParts []string
		for j := 0; j < 10; j++ {
			messageParts = append(messageParts, words[(id+seedOffset+j*7)%len(words)])
		}
		message := fmt.Sprintf("%s uuid=%s hash=%s data=%s",
			strings.Join(messageParts, " "),
			randomUUID(id+seedOffset),
			randomHash(id+seedOffset),
			randomBase64(id+seedOffset))

		return staticPart + fmt.Sprintf(`
  "message": "%s",
  "request_id": "%s",
  "trace_id": "%s",
  "status": "success"
}`, message, randomUUID(id+seedOffset+1), randomUUID(id+seedOffset+2))
	}

	// Generate training and test data
	trainingRecords := make([]string, 500)
	for i := 0; i < 500; i++ {
		trainingRecords[i] = makeRecord(i, 0)
	}

	testRecords := make([]string, 1000)
	for i := 0; i < 1000; i++ {
		testRecords[i] = makeRecord(i, 50000)
	}

	// Prepare training samples
	trainingSamples := make([][]byte, len(trainingRecords))
	for i, record := range trainingRecords {
		trainingSamples[i] = []byte(record)
	}

	// Create test array
	var testArray string
	testArray = "["
	for i, record := range testRecords {
		if i > 0 {
			testArray += ","
		}
		testArray += record
	}
	testArray += "]"
	testArrayBytes := []byte(testArray)

	// Baseline: no dictionary
	compressedNoDict, _ := Compress(nil, testArrayBytes)
	noDictRatio := float64(len(testArrayBytes)) / float64(len(compressedNoDict))

	t.Logf("\n=== Dictionary Optimization Test ===")
	t.Logf("Training samples: %d | Test data: %d bytes", len(trainingSamples), len(testArrayBytes))
	t.Logf("Baseline (no dictionary): %.2fx compression\n", noDictRatio)

	// Test 1: Dictionary size optimization
	t.Log("=== Test 1: Dictionary Size ===")
	dictSizes := []int{4 * 1024, 8 * 1024, 16 * 1024, 32 * 1024, 64 * 1024, 100 * 1024}
	bestSize := 0
	bestRatio := 0.0
	bestDict := []byte{}

	for _, size := range dictSizes {
		dict, _ := TrainFromBuffer(trainingSamples, size)
		cdict, _ := NewCDict(dict, DefaultCompression)
		compressed, _ := cdict.Compress(nil, testArrayBytes)
		cdict.Close()

		ratio := float64(len(testArrayBytes)) / float64(len(compressed))
		improvement := (ratio - noDictRatio) / noDictRatio * 100

		t.Logf("  %3d KB: %.2fx compression (+%.1f%% vs baseline)", size/1024, ratio, improvement)

		if ratio > bestRatio {
			bestRatio = ratio
			bestSize = size
			bestDict = dict
		}
	}

	t.Logf("\nBest size: %d KB (%.2fx compression)\n", bestSize/1024, bestRatio)

	// Test 2: Algorithm comparison
	t.Log("=== Test 2: Algorithm Comparison ===")

	// Basic TrainFromBuffer (uses fastCover defaults)
	dict1, _ := TrainFromBuffer(trainingSamples, bestSize)
	cdict1, _ := NewCDict(dict1, DefaultCompression)
	compressed1, _ := cdict1.Compress(nil, testArrayBytes)
	ratio1 := float64(len(testArrayBytes)) / float64(len(compressed1))
	cdict1.Close()
	t.Logf("  TrainFromBuffer (default):  %.2fx compression", ratio1)

	// COVER with manual parameters
	coverParams := CoverParams{K: 64, D: 8}
	dict2, _ := TrainFromBufferCover(trainingSamples, bestSize, coverParams)
	cdict2, _ := NewCDict(dict2, DefaultCompression)
	compressed2, _ := cdict2.Compress(nil, testArrayBytes)
	ratio2 := float64(len(testArrayBytes)) / float64(len(compressed2))
	cdict2.Close()
	t.Logf("  COVER (K=64, D=8):          %.2fx compression", ratio2)

	// fastCover with optimization
	fastParams := FastCoverParams{
		K:     0,  // Auto-optimize
		D:     0,  // Auto-optimize
		F:     20, // Default
		Steps: 4,  // Faster for testing
		Accel: 1,
	}
	dict3, optimized, _ := OptimizeTrainFromBufferFastCover(trainingSamples, bestSize, fastParams)
	cdict3, _ := NewCDict(dict3, DefaultCompression)
	compressed3, _ := cdict3.Compress(nil, testArrayBytes)
	ratio3 := float64(len(testArrayBytes)) / float64(len(compressed3))
	cdict3.Close()
	t.Logf("  fastCover optimized:        %.2fx compression (K=%d, D=%d)", ratio3, optimized.K, optimized.D)

	// Test 3: Compression level impact
	t.Log("\n=== Test 3: Compression Level (with best dict) ===")
	levels := []int{1, 3, 5, 7, 9}
	for _, level := range levels {
		cdict, _ := NewCDict(bestDict, level)
		compressed, _ := cdict.Compress(nil, testArrayBytes)
		ratio := float64(len(testArrayBytes)) / float64(len(compressed))
		cdict.Close()
		t.Logf("  Level %d: %.2fx compression", level, ratio)
	}

	// Test 4: Sample size impact
	t.Log("\n=== Test 4: Training Sample Count ===")
	sampleCounts := []int{50, 100, 200, 500}
	for _, count := range sampleCounts {
		if count > len(trainingSamples) {
			continue
		}
		samples := trainingSamples[:count]
		dict, _ := TrainFromBuffer(samples, bestSize)
		cdict, _ := NewCDict(dict, DefaultCompression)
		compressed, _ := cdict.Compress(nil, testArrayBytes)
		ratio := float64(len(testArrayBytes)) / float64(len(compressed))
		cdict.Close()
		t.Logf("  %3d samples: %.2fx compression", count, ratio)
	}

	// Summary
	t.Log("\n=== Optimization Summary ===")
	t.Logf("Baseline (no dict):     %.2fx compression", noDictRatio)
	t.Logf("Best configuration:     %.2fx compression", bestRatio)
	t.Logf("Improvement:            +%.1f%%", (bestRatio-noDictRatio)/noDictRatio*100)
	t.Logf("\nRecommendations:")
	t.Logf("  - Dictionary size: %d KB", bestSize/1024)
	t.Logf("  - Algorithm: fastCover with optimization")
	t.Logf("  - Training samples: 100-200 records")
	t.Logf("  - Compression level: 3-5 (balance speed/ratio)")
}

// generateSampleLogsFile creates a file with N sample JSON log records
func generateSampleLogsFile(filename string, count int) error {
	words := []string{
		"system", "error", "warning", "info", "debug", "critical", "fatal", "trace",
		"process", "thread", "memory", "disk", "network", "cpu", "cache", "queue",
		"request", "response", "timeout", "retry", "failed", "success", "pending", "complete",
		"user", "session", "authentication", "authorization", "permission", "access", "denied",
		"database", "query", "transaction", "commit", "rollback", "connection", "pool",
		"service", "endpoint", "handler", "middleware", "controller", "model", "view",
		"started", "stopped", "restarted", "initializing", "terminated", "crashed", "recovered",
		"upstream", "downstream", "latency", "throughput", "bandwidth", "packet", "frame",
		"encrypted", "decrypted", "signed", "verified", "hashed", "encoded", "decoded",
		"allocated", "deallocated", "garbage", "collected", "leaked", "freed", "reserved",
		"synchronized", "locked", "unlocked", "blocked", "waiting", "running", "sleeping",
		"registered", "unregistered", "subscribed", "unsubscribed", "published", "consumed",
		"validated", "rejected", "accepted", "processed", "queued", "dispatched", "delivered",
	}

	randomUUID := func(seed int) string {
		return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			seed*12345, (seed*678)%65536, (seed*910)%65536,
			(seed*1112)%65536, seed*131415)
	}

	randomHash := func(seed int) string {
		return fmt.Sprintf("%064x", seed*271828182845904523)
	}

	randomBase64 := func(seed int) string {
		chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
		result := ""
		for i := 0; i < 32; i++ {
			result += string(chars[(seed*i*7919)%len(chars)])
		}
		return result
	}

	makeRecord := func(id int) string {
		var messageParts []string
		for j := 0; j < 10; j++ {
			messageParts = append(messageParts, words[(id+j*7)%len(words)])
		}
		message := fmt.Sprintf("%s uuid=%s hash=%s data=%s",
			strings.Join(messageParts, " "),
			randomUUID(id),
			randomHash(id),
			randomBase64(id))

		// Single line JSON (JSONL format)
		return fmt.Sprintf(`{"user_id":%d,"action":"page_view","timestamp":"2024-01-01T%02d:00:00Z","session_id":"abc123def456ghi789","ip_address":"192.168.1.100","user_agent":"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36","referrer":"https://example.com/landing","page_url":"https://example.com/product/12345","page_title":"Product Details - Example Store","country":"United States","region":"California","city":"San Francisco","device_type":"desktop","browser":"Chrome","os":"Windows","message":"%s","request_id":"%s","trace_id":"%s","status":"success"}`,
			id, id%24, message, randomUUID(id+1), randomUUID(id+2))
	}

	// Create file
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Write JSONL format (one JSON object per line)
	for i := 0; i < count; i++ {
		f.WriteString(makeRecord(i))
		f.WriteString("\n")
	}

	return nil
}

func TestGenerateSampleLogs(t *testing.T) {
	filename := "sample_logs.jsonl"
	count := 10000

	t.Logf("Generating %d sample log records to %s (JSONL format)...", count, filename)
	err := generateSampleLogsFile(filename, count)
	if err != nil {
		t.Fatalf("Failed to generate logs: %v", err)
	}

	// Check file size
	info, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	t.Logf("✓ Generated %s (%.2f MB, %d records)",
		filename,
		float64(info.Size())/(1024*1024),
		count)

	// Test compression on the file
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Without dictionary
	compressedNoDict, _ := Compress(nil, data)
	noDictRatio := float64(len(data)) / float64(len(compressedNoDict))

	// Train dictionary on subset (parse JSONL - one JSON per line)
	var samples [][]byte
	lines := strings.Split(string(data), "\n")
	for i := 0; i < 100 && i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if len(line) > 0 {
			samples = append(samples, []byte(line))
		}
	}

	dict, _ := TrainFromBuffer(samples, 64*1024)
	cdict, _ := NewCDict(dict, 5)
	defer cdict.Close()

	compressedWithDict, _ := cdict.Compress(nil, data)
	withDictRatio := float64(len(data)) / float64(len(compressedWithDict))

	t.Logf("\nCompression Results:")
	t.Logf("  Original:          %10d bytes", len(data))
	t.Logf("  Without dict:      %10d bytes (%.2fx)", len(compressedNoDict), noDictRatio)
	t.Logf("  With dict (64KB):  %10d bytes (%.2fx)", len(compressedWithDict), withDictRatio)
	t.Logf("  Improvement:       %10d bytes (+%.1f%%)",
		len(compressedNoDict)-len(compressedWithDict),
		(withDictRatio-noDictRatio)/noDictRatio*100)

	t.Logf("\n✓ Test file created: %s", filename)
	t.Logf("  You can now use this file for benchmarking and testing")
}
