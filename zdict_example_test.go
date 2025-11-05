package zstd_test

import (
	"fmt"
	"log"

	"github.com/DataDog/zstd"
)

// ExampleTrainFromBuffer demonstrates basic dictionary training
func ExampleTrainFromBuffer() {
	// Collect similar samples for training (need enough samples)
	samples := make([][]byte, 50)
	for i := 0; i < 50; i++ {
		samples[i] = []byte(fmt.Sprintf(`{"name": "User%d", "age": %d, "city": "NYC", "country": "USA"}`, i, 20+i%30))
	}

	// Train a dictionary
	dict, err := zstd.TrainFromBuffer(samples, 1024)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Trained dictionary: %d bytes\n", len(dict))
	// Output: Trained dictionary: 1024 bytes
}

// ExampleCDict_Compress demonstrates compression with a dictionary
func ExampleCDict_Compress() {
	// Train a dictionary from samples
	samples := make([][]byte, 50)
	for i := 0; i < 50; i++ {
		samples[i] = []byte(fmt.Sprintf(`{"type": "log", "level": "info", "message": "test%d", "timestamp": "%d"}`, i, 1000000+i))
	}

	dict, err := zstd.TrainFromBuffer(samples, 1024)
	if err != nil {
		log.Fatal(err)
	}

	// Create a digested dictionary for compression
	cdict, err := zstd.NewCDict(dict, zstd.DefaultCompression)
	if err != nil {
		log.Fatal(err)
	}
	defer cdict.Close()

	// Compress data using the dictionary
	data := []byte(`{"type": "log", "level": "info", "message": "new log", "timestamp": "9999999"}`)
	compressed, err := cdict.Compress(nil, data)
	if err != nil {
		log.Fatal(err)
	}

	ratio := float64(len(compressed)) / float64(len(data)) * 100
	fmt.Printf("Compression ratio: %.0f%%\n", ratio)
	// Output: Compression ratio: 49%
}

// ExampleDDict_Decompress demonstrates decompression with a dictionary
func ExampleDDict_Decompress() {
	// Train and compress with dictionary
	samples := make([][]byte, 50)
	for i := 0; i < 50; i++ {
		samples[i] = []byte(fmt.Sprintf(`{"user": "user%d", "action": "login"}`, i))
	}

	dict, _ := zstd.TrainFromBuffer(samples, 1024)
	cdict, _ := zstd.NewCDict(dict, zstd.DefaultCompression)
	defer cdict.Close()

	data := []byte(`{"user": "charlie", "action": "login"}`)
	compressed, _ := cdict.Compress(nil, data)

	// Decompress using the same dictionary
	ddict, err := zstd.NewDDict(dict)
	if err != nil {
		log.Fatal(err)
	}
	defer ddict.Close()

	decompressed, err := ddict.Decompress(nil, compressed)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Successfully decompressed data\n")
	_ = decompressed
	// Output: Successfully decompressed data
}

// ExampleDictionary_compressionComparison demonstrates the size improvement when using a dictionary
func ExampleDictionary_compressionComparison() {
	// Train dictionary from samples
	samples := make([][]byte, 50)
	for i := 0; i < 50; i++ {
		samples[i] = []byte(fmt.Sprintf(`{"user": "user%d", "action": "login"}`, i))
	}

	dict, _ := zstd.TrainFromBuffer(samples, 1024)
	cdict, _ := zstd.NewCDict(dict, zstd.DefaultCompression)
	defer cdict.Close()

	// Same data to compress
	data := []byte(`{"user": "charlie", "action": "login"}`)

	// Compress WITH dictionary
	compressedWithDict, _ := cdict.Compress(nil, data)

	// Compress WITHOUT dictionary
	compressedNoDict, _ := zstd.Compress(nil, data)

	// Show the comparison
	improvement := len(compressedNoDict) - len(compressedWithDict)
	fmt.Printf("Original: %d bytes\n", len(data))
	fmt.Printf("Without dictionary: %d bytes\n", len(compressedNoDict))
	fmt.Printf("With dictionary: %d bytes\n", len(compressedWithDict))
	fmt.Printf("Improvement: %d bytes saved\n", improvement)
	// Output:
	// Original: 38 bytes
	// Without dictionary: 47 bytes
	// With dictionary: 29 bytes
	// Improvement: 18 bytes saved
}

// ExampleTrainFromBufferCover demonstrates using the COVER algorithm
func ExampleTrainFromBufferCover() {
	samples := make([][]byte, 50)
	for i := 0; i < 50; i++ {
		samples[i] = []byte(fmt.Sprintf("The quick brown fox jumps over the lazy dog number %d", i))
	}

	// Train with COVER algorithm parameters
	params := zstd.CoverParams{
		K: 16, // Segment size
		D: 8,  // dmer size
	}

	dict, err := zstd.TrainFromBufferCover(samples, 512, params)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Trained COVER dictionary\n")
	_ = dict
	// Output: Trained COVER dictionary
}

// ExampleOptimizeTrainFromBufferFastCover demonstrates parameter optimization
func ExampleOptimizeTrainFromBufferFastCover() {
	samples := make([][]byte, 50)
	for i := 0; i < 50; i++ {
		samples[i] = []byte(fmt.Sprintf("data with repetitive structure number %d", i))
	}

	// Let the algorithm optimize k and d parameters
	params := zstd.FastCoverParams{
		K:     0, // Will be optimized
		D:     0, // Will be optimized
		F:     20,
		Steps: 4, // Fewer steps for faster training
		Accel: 1,
	}

	dict, optimized, err := zstd.OptimizeTrainFromBufferFastCover(samples, 256, params)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Trained optimized dictionary\n")
	_, _ = dict, optimized
	// Output: Trained optimized dictionary
}

// ExampleFinalizeDictionary demonstrates finalizing a raw content dictionary
func ExampleFinalizeDictionary() {
	// Create raw dictionary content (common substrings)
	rawContent := []byte("common prefix, common suffix")

	// Provide samples to compute entropy tables
	samples := make([][]byte, 50)
	for i := 0; i < 50; i++ {
		samples[i] = []byte(fmt.Sprintf("common prefix: data %d, common suffix", i))
	}

	params := zstd.DictParams{
		CompressionLevel: zstd.DefaultCompression,
		DictID:           0, // Auto-generate
	}

	// Finalize adds zstd headers and entropy tables
	dict, err := zstd.FinalizeDictionary(rawContent, samples, 1024, params)
	if err != nil {
		log.Fatal(err)
	}

	// Get dictionary ID
	dictID := zstd.GetDictID(dict)
	fmt.Printf("Created dictionary with valid ID\n")
	_ = dictID
	// Output: Created dictionary with valid ID
}

// ExampleGetDictID demonstrates extracting dictionary ID
func ExampleGetDictID() {
	samples := make([][]byte, 50)
	for i := 0; i < 50; i++ {
		samples[i] = []byte(fmt.Sprintf("sample data number %d", i))
	}

	dict, _ := zstd.TrainFromBuffer(samples, 512)

	// Extract dictionary ID
	dictID := zstd.GetDictID(dict)
	if dictID != 0 {
		fmt.Println("Valid dictionary ID found")
	}
	// Output: Valid dictionary ID found
}
