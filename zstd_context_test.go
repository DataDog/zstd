package zstd

import "testing"

func TestCompressDecompressObj(t *testing.T) {
	obj := NewZstd(nil, 5)
	input := []byte("Hello World!")
	out, err := obj.Compress(nil, input)
	if err != nil {
		t.Fatalf("Error while compressing: %v", err)
	}
	rein, err := obj.Decompress(nil, out)
	if err != nil {
		t.Fatalf("Error while decompressing: %v", err)
	}

	if string(input) != string(rein) {
		t.Fatalf("Cannot compress and decompress: %s != %s", input, rein)
	}
}

func TestWithDict(t *testing.T) {
	// Build dict
	dict := make([]byte, 32*1024*1024) // 32 KB dicionnary, way overkill
	dict, err := TrainFromData(dict, [][]byte{
		[]byte("Hello nice world!"),
		[]byte("It's a very nice world!"),
	})
	if err != nil {
		t.Fatalf("Failed creating dict: %s", err)
	}
	// Compress with this dictionnary
	obj := NewZstd(dict, 5)
	defer func() {
		err := obj.Close()
		if err != nil {
			t.Fatalf("Error while closing object: %s", err)
		}
	}()

	input := []byte("Hello World!")
	out, err := obj.Compress(nil, input)
	if err != nil {
		t.Fatalf("Error while compressing: %v", err)
	}
	rein, err := obj.Decompress(nil, out)
	if err != nil {
		t.Fatalf("Error while decompressing: %v", err)
	}

	if string(input) != string(rein) {
		t.Fatalf("Cannot compress and decompress: %s != %s", input, rein)
	}
}
