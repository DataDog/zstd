# Zstd Go Wrapper

[Homepage](https://github.com/Cyan4973/zstd)
DD Maintainer: Vianney Tran
The current headers and C files are from *v0.5.0* (Commit [201433a](https://github.com/Cyan4973/zstd/commits/201433a7f713af056cc7ea32624eddefb55e10c8)).
This version has been tested on staging/prod data and is safe for use.

## Usage

There is two main API: simple compress/decompress and a streming API (reader/writer)

### Simple `Compress/Decompress`


```go
// Compress compresses the byte array given in src and writes it to dst.
// If you already have a buffer allocated, you can pass it to prevent allocation
// If not, you can pass nil as dst.
// If the buffer is too small, it will be reallocated, resized, and returned bu the function
// If dst is nil, this will allocate the worst case size (CompressBound(src))
Compress(dst, src []byte) ([]byte, error)
```

```go
// CompressLevel is the same as Compress but you can pass another compression level
CompressLevel(dst, src []byte, level int) ([]byte, error)
```

```go
// Decompress will decompress your payload into dst.
// If you already have a buffer allocated, you can pass it to prevent allocation
// If not, you can pass nil as dst (allocates a 4*src size as default).
// If the buffer is too small, it will retry 3 times by doubling the dst size
// After max retries, it will switch to the slower stream API to be sure to be able
// to decompress. Currently switches if compression ratio > 4*2**3=32.
Decompress(dst, src []byte) ([]byte, error)
```

### Stream API

```go
// NewWriter creates a new object that can optionally be initialized with
// a precomputed dictionary. If dict is nil, compress without a dictionary.
// The dictionary array should not be changed during the use of this object.
// You MUST CALL Close() to write the last bytes of a zstd stream and free C objects.
NewWriter(writer io.Writer, dict []byte, compressionLevel int)

// Write compresses the input data and write it to the underlying writer
(w *Writer) Write(p []byte) (int, error)

// Close flushes the buffer and frees everything
(w *Writer) Close() error
```

```go
// NewReader creates a new object, that can optionnaly be initialized with
// a precomputed dictionnary. If dict is nil, compress without a dictionnary
// the underlying byte array should not be changed during the use of the object.
// The dictionary array should not be changed during the use of this object.
// You MUST CALL Close() to free C objects.
NewReader(reader io.Reader, dict []byte) *Reader

// Close frees the allocated C objects
(r *Reader) Close() error

// Read satifies the io.Reader interface
(r *Reader) Read(p []byte) (int, error)
```

### Benchmarks

The guy behind Zstd is the guy behind LZ4. It's a pretty new algorithm supposed to replace
the spot of Zlib.
So far, the ratio is always better than Zlib, it compresses somewhat faster but it decompress at 3-4x the speed of Zlib

Compression of a 7Mb pdf zstd (thsi wrapper) vs czlib:
```
BenchmarkCompression           5     221056624 ns/op      67.34 MB/s
BenchmarkDecompression       100      18370416 ns/op     810.32 MB/s

BenchmarkFzlibCompress         2     610156603 ns/op      24.40 MB/s
BenchmarkFzlibDecompress          20      81195246 ns/op     183.33 MB/s
```

Ratio is also better by a margin of ~20%.
Compression speed is always better than zlib on all the payloads we tested;
However with the current version, czlib has a faster decompression for small payloads (it's highly optimized for it):
```
Testing with size: 11... czlib: 8.97 MB/s, zstd: 3.26 MB/s
Testing with size: 27... czlib: 23.3 MB/s, zstd: 8.22 MB/s
Testing with size: 62... czlib: 31.6 MB/s, zstd: 19.49 MB/s
Testing with size: 141... czlib: 74.54 MB/s, zstd: 42.55 MB/s
Testing with size: 323... czlib: 155.14 MB/s, zstd: 99.39 MB/s
Testing with size: 739... czlib: 235.9 MB/s, zstd: 216.45 MB/s
Testing with size: 1689... czlib: 116.45 MB/s, zstd: 345.64 MB/s
Testing with size: 3858... czlib: 176.39 MB/s, zstd: 617.56 MB/s
Testing with size: 8811... czlib: 254.11 MB/s, zstd: 824.34 MB/s
Testing with size: 20121... czlib: 197.43 MB/s, zstd: 1339.11 MB/s
Testing with size: 45951... czlib: 201.62 MB/s, zstd: 1951.57 MB/s
```

zstd starts tos hine with payloads > 1KB

### Stability - Current state: STABLE

The C library seems to be pretty stable and according to the author has been tested and fuzzed.

For the Go wrapper, the test cover most usual cases and we have succesfully tested it on all (soon)
staging and prod data.
