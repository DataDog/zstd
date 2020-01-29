package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"hash/fnv"
	"io"
	"math/rand"
	"sync"

	"github.com/DataDog/zstd"
	"github.com/dustin/randbo"
)

const (
	kB = 1024
	mB = 1024 * kB
	gB = 1024 * mB
)

var (
	ErrMisMatch = errors.New("Missmatch in checksums")
)

var options struct {
	totalMemoryMB  int
	totalWorkers   int
	workMultiplier int
}

/*
Benchmark:
- x goroutines
*/

func getRandomSize(source rand.Source, averageSize int) int {
	min := int(float64(averageSize) * 0.5)
	max := int(float64(averageSize) * 2.0)
	r := rand.New(source)
	return r.Intn(max-min) + min
}

func runBenchmark(workerID, fileSize, chunkSize int) error {
	totalInput := 0
	totalOutput := 0

	name := fmt.Sprintf("worker-%v", workerID)
	randSeed := rand.NewSource(int64(workerID))

	inputHash := fnv.New128a()
	r := randbo.NewFrom(randSeed)
	r = io.TeeReader(r, inputHash)

	outputHash := fnv.New128a()
	randboDataSize := getRandomSize(randSeed, chunkSize)
	randboData := make([]byte, randboDataSize)
	exitDataSize := getRandomSize(randSeed, chunkSize)
	exitData := make([]byte, exitDataSize)

	compressed := new(bytes.Buffer)
	writer := zstd.NewWriter(compressed)
	reader := zstd.NewReader(compressed)
	for i := 0; i < fileSize; i += chunkSize {
		r.Read(randboData)
		totalInput += len(randboData)
		writer.Write(randboData)

		// Then reread
		n, _ := reader.Read(exitData)
		totalOutput += n
		outputHash.Write(exitData[:n])
	}
	// Then finish
	writer.Close()
	for {
		outBuf := make([]byte, 1024)
		n, _ := reader.Read(outBuf)
		outputHash.Write(outBuf[:n])
		totalOutput += n
		if n < 1024 {
			break
		}
	}
	reader.Close()

	a := inputHash.Sum(nil)
	b := outputHash.Sum(nil)
	equal := bytes.Equal(a, b)
	equalDataLength := totalInput == totalOutput

	//fmt.Printf("[%s] Input hash:  %x\n[%s] Output hash: %x\n[%s] Equal: %t\n", name, a, name, b, name, equal)
	fmt.Printf("[%s] same_hash=%t, same_length=%t\n", name, equal, equalDataLength)
	if !equal {
		return ErrMisMatch
	}
	return nil
}

func init() {
	// Define default flags
	flag.IntVar(&options.totalMemoryMB, "total_memory_mb", 1, "How much memory you have on your machine")
	flag.IntVar(&options.totalWorkers, "workers", 8, "How many parallel workers you want")
	flag.IntVar(&options.workMultiplier, "work_multiplier", 1000, "How many loops of writer, this mainly impact duration of the test")
	flag.Parse()
}

func main() {
	totalMemory := options.totalMemoryMB * 1024 * 1024
	chunkSize := totalMemory / 2 / options.totalWorkers
	workToGo := chunkSize * options.workMultiplier

	var wg sync.WaitGroup
	errs := make(chan error, options.totalWorkers)
	for i := 0; i < options.totalWorkers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- runBenchmark(i, workToGo, chunkSize)
		}(i)
	}
	wg.Wait()
	close(errs)
	errored := false
	// Check if any errors
	for err := range errs {
		if err != nil {
			errored = true
		}
	}
	if errored {
		fmt.Println("Test failed :'(")
	} else {
		fmt.Println("All good!")
	}
}
