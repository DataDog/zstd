package zstd

import (
	"bytes"
	"encoding/base64"
	"math/rand"
	"regexp"
	"strings"
	"testing"
)

var dictBase64 string = `
	N6Qw7IsuFDIdENCSQjr//////4+QlekuNkmXbUBIkIDiVRX7H4AzAFCgQCFCO9oHAAAEQEuSikaK
	Dg51OYghBYgBAAAAAAAAAAAAAAAAAAAAANQVpmRQGQAAAAAAAAAAAAAAAAABAAAABAAAAAgAAABo
	ZWxwIEpvaW4gZW5naW5lZXJzIGVuZ2luZWVycyBmdXR1cmUgbG92ZSB0aGF0IGFyZWlsZGluZyB1
	c2UgaGVscCBoZWxwIHVzaGVyIEpvaW4gdXNlIGxvdmUgdXMgSm9pbiB1bmQgaW4gdXNoZXIgdXNo
	ZXIgYSBwbGF0Zm9ybSB1c2UgYW5kIGZ1dHVyZQ==`

func getRandomText() string {
	words := []string{"We", "are", "building", "a platform", "that", "engineers", "love", "to", "use", "Join", "us", "and", "help", "usher", "in", "the", "future"}
	wordCount := 10 + rand.Intn(100) // 10 - 109
	result := []string{}
	for i := 0; i < wordCount; i++ {
		result = append(result, words[rand.Intn(len(words))])
	}

	return strings.Join(result, " ")
}

func TestCompressAndDecompress(t *testing.T) {
	var b64 = base64.StdEncoding
	dict, err := b64.DecodeString(regexp.MustCompile(`\s+`).ReplaceAllString(dictBase64, ""))
	if err != nil {
		t.Fatalf("failed to decode the dictionary")
	}

	p, err := NewBulkProcessor(dict, BestSpeed)
	if err != nil {
		t.Fatalf("failed to create a BulkProcessor")
	}

	for i := 0; i < 100; i++ {
		payload := []byte(getRandomText())

		compressed, err := p.Compress(nil, payload)
		if err != nil {
			t.Fatalf("failed to compress")
		}

		uncompressed, err := p.Decompress(nil, compressed)
		if err != nil {
			t.Fatalf("failed to decompress")
		}

		if bytes.Compare(payload, uncompressed) != 0 {
			t.Fatalf("uncompressed payload didn't match")
		}
	}

	p.Cleanup()
}

func TestCompressAndDecompressInReverseOrder(t *testing.T) {
	var b64 = base64.StdEncoding
	dict, err := b64.DecodeString(regexp.MustCompile(`\s+`).ReplaceAllString(dictBase64, ""))
	if err != nil {
		t.Fatalf("failed to decode the dictionary")
	}

	p, err := NewBulkProcessor(dict, BestSpeed)
	if err != nil {
		t.Fatalf("failed to create a BulkProcessor")
	}

	payloads := [][]byte{}
	compressedPayloads := [][]byte{}
	for i := 0; i < 100; i++ {
		payloads = append(payloads, []byte(getRandomText()))

		compressed, err := p.Compress(nil, payloads[i])
		if err != nil {
			t.Fatalf("failed to compress")
		}
		compressedPayloads = append(compressedPayloads, compressed)
	}

	for i := 99; i >= 0; i-- {
		uncompressed, err := p.Decompress(nil, compressedPayloads[i])
		if err != nil {
			t.Fatalf("failed to decompress")
		}

		if bytes.Compare(payloads[i], uncompressed) != 0 {
			t.Fatalf("uncompressed payload didn't match")
		}
	}

	p.Cleanup()
}

// BenchmarkCompress-8   	  715689	      1550 ns/op	  59.37 MB/s	     208 B/op	       5 allocs/op
func BenchmarkCompress(b *testing.B) {
	var b64 = base64.StdEncoding
	dict, err := b64.DecodeString(regexp.MustCompile(`\s+`).ReplaceAllString(dictBase64, ""))
	if err != nil {
		b.Fatalf("failed to decode the dictionary")
	}

	p, err := NewBulkProcessor(dict, BestSpeed)
	if err != nil {
		b.Fatalf("failed to create a BulkProcessor")
	}

	payload := []byte("We're building a platform that engineers love to use. Join us, and help usher in the future.")
	for n := 0; n < b.N; n++ {
		_, err := p.Compress(nil, payload)
		if err != nil {
			b.Fatalf("failed to compress")
		}
		b.SetBytes(int64(len(payload)))
	}

	p.Cleanup()
}

// BenchmarkDecompress-8   	  664922	      1544 ns/op	  36.91 MB/s	     192 B/op	       7 allocs/op
func BenchmarkDecompress(b *testing.B) {
	var b64 = base64.StdEncoding
	dict, err := b64.DecodeString(regexp.MustCompile(`\s+`).ReplaceAllString(dictBase64, ""))
	if err != nil {
		b.Fatalf("failed to decode the dictionary")
	}

	p, err := NewBulkProcessor(dict, BestSpeed)
	if err != nil {
		b.Fatalf("failed to create a BulkProcessor")
	}

	payload, err := p.Compress(nil, []byte("We're building a platform that engineers love to use. Join us, and help usher in the future."))
	if err != nil {
		b.Fatalf("failed to compress")
	}
	for n := 0; n < b.N; n++ {
		_, err := p.Decompress(nil, payload)
		if err != nil {
			b.Fatalf("failed to decompress")
		}
		b.SetBytes(int64(len(payload)))
	}

	p.Cleanup()
}
