package zstd

// The lib is going to be compiled in /tmp

/*
#include <zstd.h>
#include <dictBuilder.h>
#cgo LDFLAGS: /usr/local/lib/libzstd.a
*/
import "C"
import (
	"unsafe"
)

type dictParams struct {
	selectivityLevel uint64
	compressionLevel uint64
}

// TrainFromData learns from each point of data and put a precomputed dict in resultDict
func TrainFromData(resultDict []byte, data [][]byte) ([]byte, error) {
	// Serialize all in one big buffer
	sizes := make([]int64, 0, len(data))
	dataBuffer := make([]byte, 0, len(data)*50)
	for _, point := range data {
		sizes = append(sizes, int64(len(point)))
		dataBuffer = append(dataBuffer, point...)
	}

	cDst := unsafe.Pointer(&resultDict[0])
	cDstSize := C.size_t(len(resultDict))
	cSrc := unsafe.Pointer(&dataBuffer[0])
	cSampleSizes := (*C.size_t)(unsafe.Pointer(&sizes[0]))
	cNbSamples := C.uint(len(sizes))
	//params := dictParams{}
	cParams := C.DiB_params_t{}
	//cParams := *C.DiB_params_t((unsafe.Pointer(&params[0])))

	cWritten := C.DiB_trainFromBuffer(cDst, cDstSize, cSrc, cSampleSizes, cNbSamples, cParams)
	written := int(cWritten)
	if getError(written) != nil {
		return nil, getError(written)
	}
	return resultDict[:written], nil
}
