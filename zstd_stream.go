package zstd

/*
#include <zstd.h>
#cgo LDFLAGS: /usr/local/lib/libzstd.a
*/
import "C"
import (
	"bytes"
	"fmt"
	"io"
	"unsafe"
)

type zstdParams struct {
	SrcSize      uint64
	WindowLog    uint32
	ContentLog   uint32
	HashLog      uint32
	SearchLog    uint32
	SearchLength uint32
	Strategy     uint32
}

// Writer is a zstd object that implements the io.WriteCloser interface
type Writer struct {
	CompressionLevel int

	ctx              *C.ZSTD_CCtx
	dict             []byte
	dstBuffer        []byte
	firstError       error
	underlyingWriter io.Writer
}

func grow(in []byte, addSize int) []byte {
	if in == nil {
		return make([]byte, addSize)
	}
	total := len(in) + addSize
	if cap(in) >= total { // Do not reallocate if not needed
		return in[:total]
	}
	out := make([]byte, len(in)+addSize)
	copy(out, in)
	return out
}

// NewWriter creates a new object, that can optionnaly be initialized with
// a precomputed dictionnary. If dict is nil, compress without a dictionnary
// the underlying byte array should not be changed during the use of the object.
// This will allow you to compress streams
func NewWriter(writer io.Writer, dict []byte, compressionLevel int) *Writer {
	var err error
	ctx := C.ZSTD_createCCtx()
	cCompression := C.int(compressionLevel)

	if dict == nil {
		err = getError(int(C.ZSTD_compressBegin(ctx, cCompression)))
	} else {
		cDict := unsafe.Pointer(&dict[0])
		cDictSize := C.size_t(len(dict))
		err = getError(int(C.ZSTD_compressBegin_usingDict(ctx, cDict, cDictSize, cCompression)))
	}

	return &Writer{
		CompressionLevel: compressionLevel,
		ctx:              ctx,
		dict:             dict,
		firstError:       err,
		underlyingWriter: writer,
	}
}

// Write compress the input data and write it to the underlying writer
func (w *Writer) Write(p []byte) (int, error) {
	if w.firstError != nil {
		return 0, w.firstError
	}
	// Check if dstBuffer is enough
	if len(w.dstBuffer) < CompressBound(len(p)) {
		w.dstBuffer = make([]byte, CompressBound(len(p)))
	}
	cDst := unsafe.Pointer(&w.dstBuffer[0])
	cDstSize := C.size_t(len(w.dstBuffer))
	cSrc := unsafe.Pointer(&p[0])
	cSrcSize := C.size_t(len(p))

	retCode := C.ZSTD_compressContinue(w.ctx, cDst, cDstSize, cSrc, cSrcSize)
	if err := getError(int(retCode)); err != nil {
		return 0, err
	}
	written := int(retCode)

	// Write to underlying buffer
	return w.underlyingWriter.Write(w.dstBuffer[:written])
}

// Close flushes the buffer and frees everything
func (w *Writer) Close() error {
	cDst := unsafe.Pointer(&w.dstBuffer[0])
	cDstSize := C.size_t(len(w.dstBuffer))
	retCode := C.ZSTD_compressEnd(w.ctx, cDst, cDstSize)
	if err := getError(int(retCode)); err != nil {
		return err
	}
	written := int(retCode)
	retCode = C.ZSTD_freeCCtx(w.ctx) // Safely close buffer before writing the end

	if err := getError(int(retCode)); err != nil {
		return err
	}

	_, err := w.underlyingWriter.Write(w.dstBuffer[:written])
	if err != nil {
		return err
	}
	return nil
}

// Reader is a zstd object that implements io.ReadCloser
type Reader struct {
	ctx        *C.ZSTD_DCtx
	dict       []byte
	dstBuffer  []byte
	firstError error

	tempBuffer       bytes.Buffer
	underlyingReader io.Reader
}

// NewReader creates a new object, that can optionnaly be initialized with
// a precomputed dictionnary. If dict is nil, compress without a dictionnary
// the underlying byte array should not be changed during the use of the object.
// This will allow you to decompress streams
func NewReader(reader io.Reader, dict []byte) *Reader {
	var err error
	ctx := C.ZSTD_createDCtx()
	if dict == nil {
		err = getError(int(C.ZSTD_decompressBegin(ctx)))
	} else {
		cDict := unsafe.Pointer(&dict[0])
		cDictSize := C.size_t(len(dict))
		err = getError(int(C.ZSTD_decompressBegin_usingDict(ctx, cDict, cDictSize)))
	}
	return &Reader{
		ctx:              ctx,
		dict:             dict,
		firstError:       err,
		underlyingReader: reader,
	}
}

// Close frees the allocated C objects
func (r *Reader) Close() error {
	return getError(int(C.ZSTD_freeDCtx(r.ctx)))
}

// initBuffer must be called at first call of Read to initialize dst buffer,
// it will read a bit of data from the reader while reading headers and returns
// that too so you can also decompress that piece of data
func (r *Reader) initBuffer() ([]byte, error) {
	readBuf := make([]byte, 100) // We will try to read max 100 bytes of data first
	params := zstdParams{}
	cParams := (*C.ZSTD_parameters)(unsafe.Pointer(&params))
	read, err := r.underlyingReader.Read(readBuf)
	if err != nil && err != io.EOF {
		return []byte{}, fmt.Errorf("failed to read from underlying reader: %s", err)
	}

	success := false
	// Try 3 times to read the params buffer, readjusting the size when necessary
	for i := 0; i < 3; i++ {
		cSrc := unsafe.Pointer(&readBuf[0])
		cSrcSize := C.size_t(read)
		// retCode: == 0 if the write to params was successful
		// > 0 indicates the number of additional bytes it needs to successfully populate params
		// < 0 is an error code
		retCode := int(C.ZSTD_getFrameParams(cParams, cSrc, cSrcSize))
		if retCode == 0 {
			success = true
			break
		} else if retCode < 0 {
			return readBuf[:read], getError(retCode)
		}
		// Reread with more data
		readBuf = grow(readBuf, retCode)
		read2, err := r.underlyingReader.Read(readBuf[read:]) // Read the remaining bytes
		read += read2
		if err != nil && err != io.EOF {
			return readBuf[:read], fmt.Errorf("failed to read from underlying reader: %s", err)
		}
		if err == io.EOF && read2 == 0 {
			return readBuf[:read], fmt.Errorf("Missing headers ?")
		}
	}
	if !success {
		return readBuf[:read], fmt.Errorf("failed to allocate buffer to read params")
	}
	// Allocate buffer
	toAllocate := 1 << params.WindowLog
	r.dstBuffer = make([]byte, toAllocate)
	return readBuf[:read], nil
}

// Called the first time ever of Read() to init dst buffer and process as few data as
// possible from that header. Return 0 if it did not have enough data to write to the
// caller
func (r *Reader) readInit(p []byte) (int, error) {
	if r.dstBuffer != nil {
		panic(fmt.Errorf("ReadInit should only be called once !"))
	}
	readBuf, err := r.initBuffer()
	if err != nil {
		return 0, fmt.Errorf("failed to allocate buffer: %s", err)
	}
	for len(readBuf) > 0 { // Decompress until we are synced with input
		cToRead := C.ZSTD_nextSrcSizeToDecompress(r.ctx)
		toRead := int(cToRead)
		if len(readBuf) < toRead { // Read the few more needed bytes
			previousSize := len(readBuf)
			readBuf = grow(readBuf, toRead-len(readBuf))
			n, err := r.underlyingReader.Read(readBuf[previousSize:])
			if err != nil {
				return 0, fmt.Errorf("failed to read from underlying reader: %s", err)
			}
			if n != (toRead - previousSize) {
				return 0, fmt.Errorf("failed to read enough bytes from reader: %v != %v", n, (toRead - previousSize))
			}
		}
		// Decompress and append to tempBuffer
		cDst := unsafe.Pointer(&r.dstBuffer[0])
		cDstSize := C.size_t(len(r.dstBuffer))
		cSrc := unsafe.Pointer(&readBuf[0])
		retCode := int(C.ZSTD_decompressContinue(r.ctx, cDst, cDstSize, cSrc, cToRead))
		if err := getError(retCode); err != nil {
			return 0, err
		}
		_, err := r.tempBuffer.Write(r.dstBuffer[:retCode])
		if err != nil {
			return 0, fmt.Errorf("failed to write to temporary buffer: %s", err)
		}
		readBuf = readBuf[toRead:]
	}
	if r.tempBuffer.Len() >= len(p) { // We can directly copy from buffer
		return r.tempBuffer.Read(p)
	}
	return 0, nil // Populated tempBuffer with initial data but it's still not enough
}

// Read satifies the io.Reader interface
func (r *Reader) Read(p []byte) (int, error) {
	if r.firstError != nil {
		return 0, r.firstError
	}
	if r.tempBuffer.Len() >= len(p) { // We can directly copy from buffer
		return r.tempBuffer.Read(p)
	}

	if r.dstBuffer == nil { // Not inited yet, we need to know how much space we need to decode
		n, err := r.readInit(p)
		if err != nil {
			return 0, fmt.Errorf("failed to init: %s", err)
		}
		if n > 0 {
			return n, err
		}
	}
	// We need to read more
	for r.tempBuffer.Len() < len(p) {
		cToRead := C.ZSTD_nextSrcSizeToDecompress(r.ctx)
		if int(cToRead) == 0 { // End of stream
			return r.tempBuffer.Read(p)
		}
		src := make([]byte, int(cToRead))
		n, err := r.underlyingReader.Read(src)
		if err == io.EOF { // We do not have anything to process anymore
			if n == 0 { // End of input, return eveything we have
				return r.tempBuffer.Read(p)
			} // Else do nothing, we still need to process the remaining bytes
		} else if err != nil {
			return 0, fmt.Errorf("failed to read underlying buffer: %s", err)
		}
		// Decompress and append to tempBuffer
		cDst := unsafe.Pointer(&r.dstBuffer[0])
		cDstSize := C.size_t(len(r.dstBuffer))
		cSrc := unsafe.Pointer(&src[0])
		retCode := int(C.ZSTD_decompressContinue(r.ctx, cDst, cDstSize, cSrc, cToRead))
		if err := getError(retCode); err != nil {
			return 0, err
		}
		_, err = r.tempBuffer.Write(r.dstBuffer[:retCode])
		if err != nil {
			return 0, fmt.Errorf("failed to write to temporary buffer: %s", err)
		}
	}

	return r.tempBuffer.Read(p)
}
