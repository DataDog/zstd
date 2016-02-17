package zstd

/*
#include <zstd.h>
#include <zstd_buffered.h>
#cgo LDFLAGS: /usr/local/lib/libzstd.a
*/
import "C"
import (
	"bytes"
	"fmt"
	"io"
	"unsafe"
)

// Writer is a zstd object that implements the io.WriteCloser interface
type Writer struct {
	CompressionLevel int

	ctx              *C.ZSTD_CCtx
	dict             []byte
	dstBuffer        []byte
	firstError       error
	underlyingWriter io.Writer
}

func resize(in []byte, newSize int) []byte {
	if in == nil {
		return make([]byte, newSize)
	}
	if newSize <= cap(in) {
		return in[:newSize]
	}
	toAdd := newSize - len(in)
	return append(in, make([]byte, toAdd)...)
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

// NewWriter creates a new object, that can optionally be initialized with
// a precomputed dictionary. If dict is nil, compress without a dictionary
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
		dstBuffer:        make([]byte, CompressBound(1024)),
		firstError:       err,
		underlyingWriter: writer,
	}
}

// Write compress the input data and write it to the underlying writer
func (w *Writer) Write(p []byte) (int, error) {
	if w.firstError != nil {
		return 0, w.firstError
	}
	if len(p) == 0 {
		return 0, nil
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
	ctx                 *C.ZBUFF_DCtx
	compressionBuffer   []byte
	decompressionBuffer []byte
	dict                []byte
	firstError          error
	// Reuse previous bytes from source that were not consumed
	// Hopefully because we use recommended size, we will never need to use that
	srcBuffer          bytes.Buffer
	dstBuffer          bytes.Buffer
	recommendedSrcSize int
	underlyingReader   io.Reader
}

// NewReader creates a new object, that can optionnaly be initialized with
// a precomputed dictionnary. If dict is nil, compress without a dictionnary
// the underlying byte array should not be changed during the use of the object.
// This will allow you to decompress streams
func NewReader(reader io.Reader, dict []byte) *Reader {
	var err error
	ctx := C.ZBUFF_createDCtx()
	if dict == nil {
		err = getError(int(C.ZBUFF_decompressInit(ctx)))
	} else {
		cDict := unsafe.Pointer(&dict[0])
		cDictSize := C.size_t(len(dict))
		err = getError(int(C.ZBUFF_decompressInitDictionary(ctx, cDict, cDictSize)))
	}
	cSize := int(C.ZBUFF_recommendedDInSize())
	dSize := int(C.ZBUFF_recommendedDOutSize())
	if cSize <= 0 {
		panic(fmt.Errorf("C function ZBUFF_recommendedDInSize() returned a wrong size: %v", cSize))
	}
	if dSize <= 0 {
		panic(fmt.Errorf("C function ZBUFF_recommendedDOutSize() returned a weird size: %v", dSize))
	}

	compressionBuffer := make([]byte, cSize)
	decompressionBuffer := make([]byte, dSize)
	return &Reader{
		ctx:                 ctx,
		dict:                dict,
		compressionBuffer:   compressionBuffer,
		decompressionBuffer: decompressionBuffer,
		firstError:          err,
		recommendedSrcSize:  cSize,
		underlyingReader:    reader,
	}
}

// Close frees the allocated C objects
func (r *Reader) Close() error {
	return getError(int(C.ZBUFF_freeDCtx(r.ctx)))
}

// Read satifies the io.Reader interface
func (r *Reader) Read(p []byte) (int, error) {

	// If we already have enough bytes, return
	if r.dstBuffer.Len() >= len(p) {
		return r.dstBuffer.Read(p)
	}

	for r.dstBuffer.Len() < len(p) {
		// Populate src
		src := r.compressionBuffer
		reader := r.underlyingReader
		if r.srcBuffer.Len() != 0 {
			reader = io.MultiReader(&r.srcBuffer, r.underlyingReader)
		}
		n, err := io.ReadFull(reader, src)
		if err == io.EOF {
			break
		} else if err != nil && err != io.ErrUnexpectedEOF {
			return 0, fmt.Errorf("failed to read from underlying reader: %s", err)
		}
		src = src[:n]

		// C code
		cSrc := unsafe.Pointer(&src[0])
		cSrcSize := C.size_t(len(src))
		cDst := unsafe.Pointer(&r.decompressionBuffer[0])
		cDstSize := C.size_t(len(r.decompressionBuffer))
		retCode := int(C.ZBUFF_decompressContinue(r.ctx, cDst, &cDstSize, cSrc, &cSrcSize))
		if err = getError(retCode); err != nil {
			return 0, fmt.Errorf("failed to decompress: %s", err)
		}

		// Put everything in buffer
		if int(cSrcSize) < len(src) { // We did not read everything, put in buffer
			toSave := src[int(cSrcSize):]
			_, err = r.srcBuffer.Write(toSave)
			if err != nil {
				return 0, fmt.Errorf("failed to store temporary src buffer: %s", err)
			}
		}
		_, err = r.dstBuffer.Write(r.decompressionBuffer[:int(cDstSize)])
		if err != nil {
			return 0, fmt.Errorf("failed to store temporary result: %s", err)
		}

		// Resize buffers
		if retCode > 0 { // Hint for next src buffer size
			r.compressionBuffer = resize(r.compressionBuffer, retCode)
		} else { // Reset to recommended size
			r.compressionBuffer = resize(r.compressionBuffer, r.recommendedSrcSize)
		}
	}
	// Write to dst
	return r.dstBuffer.Read(p)
}
