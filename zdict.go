package zstd

/*
#cgo CFLAGS: -DZDICT_STATIC_LINKING_ONLY
#include "zstd.h"
#include "zdict.h"
*/
import "C"
import (
	"errors"
	"runtime"
	"unsafe"
)

var (
	// ErrNotEnoughSamples is returned when dictionary training fails due to insufficient samples
	ErrNotEnoughSamples = errors.New("not enough samples for dictionary training")
)

// DictParams are parameters for dictionary building
type DictParams struct {
	CompressionLevel  int  // optimize for a specific zstd compression level; 0 means default
	NotificationLevel uint // log verbosity; 0=none, 1=errors, 2=progression, 3=details, 4=debug
	DictID            uint // force dictID value; 0 means auto mode (32-bits random value)
}

func (p DictParams) toCParams() C.ZDICT_params_t {
	return C.ZDICT_params_t{
		compressionLevel:  C.int(p.CompressionLevel),
		notificationLevel: C.uint(p.NotificationLevel),
		dictID:            C.uint(p.DictID),
	}
}

// CoverParams are parameters for the COVER dictionary training algorithm
type CoverParams struct {
	K                       uint    // Segment size : constraint: 0 < k : Reasonable range [16, 2048+]
	D                       uint    // dmer size : constraint: 0 < d <= k : Reasonable range [6, 16]
	Steps                   uint    // Number of steps : Only used for optimization : 0 means default (40)
	NBThreads               uint    // Number of threads : constraint: 0 < nbThreads : 1 means single-threaded
	SplitPoint              float64 // Percentage of samples used for training: 0 means default (1.0)
	ShrinkDict              uint    // Train dictionaries to shrink in size: 0 means no shrinking, 1 means shrinking
	ShrinkDictMaxRegression uint    // Max regression percentage for shrinking
	CompressionLevel        int     // optimize for a specific zstd compression level; 0 means default
	NotificationLevel       uint    // log verbosity; 0=none, 1=errors, 2=progression, 3=details, 4=debug
	DictID                  uint    // force dictID value; 0 means auto mode (32-bits random value)
}

func (p CoverParams) toCParams() C.ZDICT_cover_params_t {
	return C.ZDICT_cover_params_t{
		k:                       C.uint(p.K),
		d:                       C.uint(p.D),
		steps:                   C.uint(p.Steps),
		nbThreads:               C.uint(p.NBThreads),
		splitPoint:              C.double(p.SplitPoint),
		shrinkDict:              C.uint(p.ShrinkDict),
		shrinkDictMaxRegression: C.uint(p.ShrinkDictMaxRegression),
		zParams: C.ZDICT_params_t{
			compressionLevel:  C.int(p.CompressionLevel),
			notificationLevel: C.uint(p.NotificationLevel),
			dictID:            C.uint(p.DictID),
		},
	}
}

func cParamsToCoverParams(cp C.ZDICT_cover_params_t) CoverParams {
	return CoverParams{
		K:                       uint(cp.k),
		D:                       uint(cp.d),
		Steps:                   uint(cp.steps),
		NBThreads:               uint(cp.nbThreads),
		SplitPoint:              float64(cp.splitPoint),
		ShrinkDict:              uint(cp.shrinkDict),
		ShrinkDictMaxRegression: uint(cp.shrinkDictMaxRegression),
		CompressionLevel:        int(cp.zParams.compressionLevel),
		NotificationLevel:       uint(cp.zParams.notificationLevel),
		DictID:                  uint(cp.zParams.dictID),
	}
}

// FastCoverParams are parameters for the fastCover dictionary training algorithm
type FastCoverParams struct {
	K                       uint    // Segment size : constraint: 0 < k : Reasonable range [16, 2048+]
	D                       uint    // dmer size : constraint: 0 < d <= k : Reasonable range [6, 16]
	F                       uint    // log of size of frequency array : constraint: 0 < f <= 31 : 1 means default(20)
	Steps                   uint    // Number of steps : Only used for optimization : 0 means default (40)
	NBThreads               uint    // Number of threads : constraint: 0 < nbThreads : 1 means single-threaded
	SplitPoint              float64 // Percentage of samples used for training: 0 means default (0.75)
	Accel                   uint    // Acceleration level: constraint: 0 < accel <= 10, higher means faster and less accurate, 0 means default(1)
	ShrinkDict              uint    // Train dictionaries to shrink in size: 0 means no shrinking, 1 means shrinking
	ShrinkDictMaxRegression uint    // Max regression percentage for shrinking
	CompressionLevel        int     // optimize for a specific zstd compression level; 0 means default
	NotificationLevel       uint    // log verbosity; 0=none, 1=errors, 2=progression, 3=details, 4=debug
	DictID                  uint    // force dictID value; 0 means auto mode (32-bits random value)
}

func (p FastCoverParams) toCParams() C.ZDICT_fastCover_params_t {
	return C.ZDICT_fastCover_params_t{
		k:                       C.uint(p.K),
		d:                       C.uint(p.D),
		f:                       C.uint(p.F),
		steps:                   C.uint(p.Steps),
		nbThreads:               C.uint(p.NBThreads),
		splitPoint:              C.double(p.SplitPoint),
		accel:                   C.uint(p.Accel),
		shrinkDict:              C.uint(p.ShrinkDict),
		shrinkDictMaxRegression: C.uint(p.ShrinkDictMaxRegression),
		zParams: C.ZDICT_params_t{
			compressionLevel:  C.int(p.CompressionLevel),
			notificationLevel: C.uint(p.NotificationLevel),
			dictID:            C.uint(p.DictID),
		},
	}
}

func cParamsToFastCoverParams(cp C.ZDICT_fastCover_params_t) FastCoverParams {
	return FastCoverParams{
		K:                       uint(cp.k),
		D:                       uint(cp.d),
		F:                       uint(cp.f),
		Steps:                   uint(cp.steps),
		NBThreads:               uint(cp.nbThreads),
		SplitPoint:              float64(cp.splitPoint),
		Accel:                   uint(cp.accel),
		ShrinkDict:              uint(cp.shrinkDict),
		ShrinkDictMaxRegression: uint(cp.shrinkDictMaxRegression),
		CompressionLevel:        int(cp.zParams.compressionLevel),
		NotificationLevel:       uint(cp.zParams.notificationLevel),
		DictID:                  uint(cp.zParams.dictID),
	}
}

// getDictError returns an error for the return code, or nil if it's not an error
func getDictError(code int) error {
	if code < 0 && int(C.ZDICT_isError(C.size_t(code))) != 0 {
		return ErrorCode(code)
	}
	return nil
}

// TrainFromBuffer trains a dictionary from an array of samples.
// It returns the dictionary buffer or an error.
// This is equivalent to calling TrainFromBufferFastCover with default parameters (d=8, steps=4, f=20, accel=1).
func TrainFromBuffer(samples [][]byte, dictSize int) ([]byte, error) {
	if len(samples) == 0 {
		return nil, ErrNotEnoughSamples
	}

	// Flatten samples into a single buffer and collect sizes
	samplesBuffer, samplesSizes := flattenSamples(samples)
	dictBuffer := make([]byte, dictSize)

	var samplesPtr unsafe.Pointer
	if len(samplesBuffer) > 0 {
		samplesPtr = unsafe.Pointer(&samplesBuffer[0])
	}

	written := C.ZDICT_trainFromBuffer(
		unsafe.Pointer(&dictBuffer[0]),
		C.size_t(len(dictBuffer)),
		samplesPtr,
		&samplesSizes[0],
		C.uint(len(samplesSizes)))

	writtenSize := int(written)
	if err := getDictError(writtenSize); err != nil {
		return nil, err
	}

	return dictBuffer[:writtenSize], nil
}

// TrainFromBufferCover trains a dictionary using the COVER algorithm.
func TrainFromBufferCover(samples [][]byte, dictSize int, params CoverParams) ([]byte, error) {
	if len(samples) == 0 {
		return nil, ErrNotEnoughSamples
	}

	samplesBuffer, samplesSizes := flattenSamples(samples)
	dictBuffer := make([]byte, dictSize)

	var samplesPtr unsafe.Pointer
	if len(samplesBuffer) > 0 {
		samplesPtr = unsafe.Pointer(&samplesBuffer[0])
	}

	cParams := params.toCParams()
	written := C.ZDICT_trainFromBuffer_cover(
		unsafe.Pointer(&dictBuffer[0]),
		C.size_t(len(dictBuffer)),
		samplesPtr,
		&samplesSizes[0],
		C.uint(len(samplesSizes)),
		cParams)

	writtenSize := int(written)
	if err := getDictError(writtenSize); err != nil {
		return nil, err
	}

	return dictBuffer[:writtenSize], nil
}

// OptimizeTrainFromBufferCover trains a dictionary using the COVER algorithm with optimization.
// It tries many parameter combinations and picks the best parameters.
// The optimized parameters are returned along with the dictionary.
func OptimizeTrainFromBufferCover(samples [][]byte, dictSize int, params CoverParams) ([]byte, CoverParams, error) {
	if len(samples) == 0 {
		return nil, params, ErrNotEnoughSamples
	}

	samplesBuffer, samplesSizes := flattenSamples(samples)
	dictBuffer := make([]byte, dictSize)

	var samplesPtr unsafe.Pointer
	if len(samplesBuffer) > 0 {
		samplesPtr = unsafe.Pointer(&samplesBuffer[0])
	}

	cParams := params.toCParams()
	written := C.ZDICT_optimizeTrainFromBuffer_cover(
		unsafe.Pointer(&dictBuffer[0]),
		C.size_t(len(dictBuffer)),
		samplesPtr,
		&samplesSizes[0],
		C.uint(len(samplesSizes)),
		&cParams)

	writtenSize := int(written)
	if err := getDictError(writtenSize); err != nil {
		return nil, params, err
	}

	return dictBuffer[:writtenSize], cParamsToCoverParams(cParams), nil
}

// TrainFromBufferFastCover trains a dictionary using the fastCover algorithm.
func TrainFromBufferFastCover(samples [][]byte, dictSize int, params FastCoverParams) ([]byte, error) {
	if len(samples) == 0 {
		return nil, ErrNotEnoughSamples
	}

	samplesBuffer, samplesSizes := flattenSamples(samples)
	dictBuffer := make([]byte, dictSize)

	var samplesPtr unsafe.Pointer
	if len(samplesBuffer) > 0 {
		samplesPtr = unsafe.Pointer(&samplesBuffer[0])
	}

	cParams := params.toCParams()
	written := C.ZDICT_trainFromBuffer_fastCover(
		unsafe.Pointer(&dictBuffer[0]),
		C.size_t(len(dictBuffer)),
		samplesPtr,
		&samplesSizes[0],
		C.uint(len(samplesSizes)),
		cParams)

	writtenSize := int(written)
	if err := getDictError(writtenSize); err != nil {
		return nil, err
	}

	return dictBuffer[:writtenSize], nil
}

// OptimizeTrainFromBufferFastCover trains a dictionary using the fastCover algorithm with optimization.
// It tries many parameter combinations and picks the best parameters.
// The optimized parameters are returned along with the dictionary.
func OptimizeTrainFromBufferFastCover(samples [][]byte, dictSize int, params FastCoverParams) ([]byte, FastCoverParams, error) {
	if len(samples) == 0 {
		return nil, params, ErrNotEnoughSamples
	}

	samplesBuffer, samplesSizes := flattenSamples(samples)
	dictBuffer := make([]byte, dictSize)

	var samplesPtr unsafe.Pointer
	if len(samplesBuffer) > 0 {
		samplesPtr = unsafe.Pointer(&samplesBuffer[0])
	}

	cParams := params.toCParams()
	written := C.ZDICT_optimizeTrainFromBuffer_fastCover(
		unsafe.Pointer(&dictBuffer[0]),
		C.size_t(len(dictBuffer)),
		samplesPtr,
		&samplesSizes[0],
		C.uint(len(samplesSizes)),
		&cParams)

	writtenSize := int(written)
	if err := getDictError(writtenSize); err != nil {
		return nil, params, err
	}

	return dictBuffer[:writtenSize], cParamsToFastCoverParams(cParams), nil
}

// FinalizeDictionary finalizes a raw content dictionary by adding zstd headers and statistics.
// The samples are used to compute statistics. The dictContent can be empty.
func FinalizeDictionary(dictContent []byte, samples [][]byte, maxDictSize int, params DictParams) ([]byte, error) {
	if len(samples) == 0 {
		return nil, ErrNotEnoughSamples
	}

	samplesBuffer, samplesSizes := flattenSamples(samples)
	dictBuffer := make([]byte, maxDictSize)

	var dictContentPtr unsafe.Pointer
	if len(dictContent) > 0 {
		dictContentPtr = unsafe.Pointer(&dictContent[0])
	}

	var samplesPtr unsafe.Pointer
	if len(samplesBuffer) > 0 {
		samplesPtr = unsafe.Pointer(&samplesBuffer[0])
	}

	cParams := params.toCParams()
	written := C.ZDICT_finalizeDictionary(
		unsafe.Pointer(&dictBuffer[0]),
		C.size_t(len(dictBuffer)),
		dictContentPtr,
		C.size_t(len(dictContent)),
		samplesPtr,
		&samplesSizes[0],
		C.uint(len(samplesSizes)),
		cParams)

	writtenSize := int(written)
	if err := getDictError(writtenSize); err != nil {
		return nil, err
	}

	return dictBuffer[:writtenSize], nil
}

// GetDictID extracts the dictionary ID from a dictionary buffer.
// Returns 0 if the buffer is not a valid dictionary.
func GetDictID(dict []byte) uint {
	if len(dict) == 0 {
		return 0
	}
	return uint(C.ZDICT_getDictID(unsafe.Pointer(&dict[0]), C.size_t(len(dict))))
}

// GetDictHeaderSize returns the size of the dictionary header.
// Returns an error if the dictionary is not valid.
func GetDictHeaderSize(dict []byte) (int, error) {
	if len(dict) == 0 {
		return 0, errors.New("empty dictionary")
	}
	size := int(C.ZDICT_getDictHeaderSize(unsafe.Pointer(&dict[0]), C.size_t(len(dict))))
	if err := getError(size); err != nil {
		return 0, err
	}
	return size, nil
}

// flattenSamples converts a slice of byte slices into a single flat buffer and a sizes array
func flattenSamples(samples [][]byte) ([]byte, []C.size_t) {
	totalSize := 0
	for _, sample := range samples {
		totalSize += len(sample)
	}

	buffer := make([]byte, totalSize)
	sizes := make([]C.size_t, len(samples))
	offset := 0

	for i, sample := range samples {
		copy(buffer[offset:], sample)
		sizes[i] = C.size_t(len(sample))
		offset += len(sample)
	}

	return buffer, sizes
}

// CDict is a digested dictionary for compression
type CDict struct {
	cdict *C.ZSTD_CDict
}

// NewCDict creates a digested dictionary for compression.
// The dictionary is digested once, making it faster to use with multiple compressions.
func NewCDict(dict []byte, compressionLevel int) (*CDict, error) {
	if len(dict) == 0 {
		return nil, errors.New("empty dictionary")
	}

	cdict := C.ZSTD_createCDict(
		unsafe.Pointer(&dict[0]),
		C.size_t(len(dict)),
		C.int(compressionLevel))

	if cdict == nil {
		return nil, errors.New("failed to create CDict")
	}

	cd := &CDict{cdict: cdict}
	runtime.SetFinalizer(cd, finalizeCDict)
	return cd, nil
}

// Close frees the CDict resources
func (cd *CDict) Close() {
	if cd.cdict != nil {
		C.ZSTD_freeCDict(cd.cdict)
		cd.cdict = nil
	}
}

func finalizeCDict(cd *CDict) {
	cd.Close()
}

// CompressWithDict compresses src using a dictionary into dst.
// If dst is too small or nil, a new buffer will be allocated and returned.
func (cd *CDict) Compress(dst, src []byte) ([]byte, error) {
	bound := CompressBound(len(src))
	if cap(dst) >= bound {
		dst = dst[0:bound]
	} else {
		dst = make([]byte, bound)
	}

	cctx := C.ZSTD_createCCtx()
	if cctx == nil {
		return nil, errors.New("failed to create compression context")
	}
	defer C.ZSTD_freeCCtx(cctx)

	var srcPtr unsafe.Pointer
	if len(src) > 0 {
		srcPtr = unsafe.Pointer(&src[0])
	}

	written := C.ZSTD_compress_usingCDict(
		cctx,
		unsafe.Pointer(&dst[0]),
		C.size_t(len(dst)),
		srcPtr,
		C.size_t(len(src)),
		cd.cdict)

	writtenSize := int(written)
	if err := getError(writtenSize); err != nil {
		return nil, err
	}

	return dst[:writtenSize], nil
}

// DDict is a digested dictionary for decompression
type DDict struct {
	ddict *C.ZSTD_DDict
}

// NewDDict creates a digested dictionary for decompression.
// The dictionary is digested once, making it faster to use with multiple decompressions.
func NewDDict(dict []byte) (*DDict, error) {
	if len(dict) == 0 {
		return nil, errors.New("empty dictionary")
	}

	ddict := C.ZSTD_createDDict(
		unsafe.Pointer(&dict[0]),
		C.size_t(len(dict)))

	if ddict == nil {
		return nil, errors.New("failed to create DDict")
	}

	dd := &DDict{ddict: ddict}
	runtime.SetFinalizer(dd, finalizeDDict)
	return dd, nil
}

// Close frees the DDict resources
func (dd *DDict) Close() {
	if dd.ddict != nil {
		C.ZSTD_freeDDict(dd.ddict)
		dd.ddict = nil
	}
}

func finalizeDDict(dd *DDict) {
	dd.Close()
}

// Decompress decompresses src using a dictionary into dst.
// If dst is too small or nil, a new buffer will be allocated and returned.
func (dd *DDict) Decompress(dst, src []byte) ([]byte, error) {
	if len(src) == 0 {
		return []byte{}, ErrEmptySlice
	}

	bound := decompressSizeHint(src)
	if cap(dst) >= bound {
		dst = dst[0:cap(dst)]
	} else {
		dst = make([]byte, bound)
	}

	dctx := C.ZSTD_createDCtx()
	if dctx == nil {
		return nil, errors.New("failed to create decompression context")
	}
	defer C.ZSTD_freeDCtx(dctx)

	written := C.ZSTD_decompress_usingDDict(
		dctx,
		unsafe.Pointer(&dst[0]),
		C.size_t(len(dst)),
		unsafe.Pointer(&src[0]),
		C.size_t(len(src)),
		dd.ddict)

	writtenSize := int(written)
	if err := getError(writtenSize); err != nil {
		return nil, err
	}

	return dst[:writtenSize], nil
}
