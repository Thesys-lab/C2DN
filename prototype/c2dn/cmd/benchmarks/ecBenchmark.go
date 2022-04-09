package main

import (
	"fmt"
	"github.com/klauspost/reedsolomon"
	"math"
	"math/rand"
	"runtime"
	"sort"
	"sync"
	"time"
)

/*
One process
128 	 724.1073
512 	 2681.7331
1024 	 4938.7930
4096 	 3177.1853
16384 	 6744.4408
131072 	 18119.3933


128 	 1471.0211
512 	 7730.6266
1024 	 14769.1930
4096 	 7323.2548
16384 	 23932.6986
131072 	 67645.7349
*/

func BenchmarkDecode(n, k int, thrptMap map[int]float64, mtx *sync.Mutex, wg *sync.WaitGroup) {
	defer (*wg).Done()
	rand.Seed(time.Now().UnixNano())

	//fmt.Println("WithMaxGoroutines(1) WithCauchyMatrix()")
	// coder, _ := reedsolomon.New(k, n-k)
	coder, _ := reedsolomon.New(k, n-k, reedsolomon.WithMaxGoroutines(1))
	// coder, _ := reedsolomon.New(k, n-k, reedsolomon.WithMaxGoroutines(1), reedsolomon.WithCauchyMatrix())
	// coder, _ := reedsolomon.New(n-k, k, reedsolomon.WithMaxGoroutines(4), reedsolomon.WithCauchyMatrix(), reedsolomon.WithMinSplitSize(1024))

	// for chunkSize := 16; chunkSize < int(math.Pow(2, 24)); chunkSize *= 4 {
	for chunkSize := 128; chunkSize < int(math.Pow(2, 20)); chunkSize *= 2 {
		//coder, _ := reedsolomon.New(k, n-k, reedsolomon.WithAutoGoroutines(chunkSize), reedsolomon.WithCauchyMatrix())
		encodedData := encodeSimple(coder, k, chunkSize)
		rd := rand.New(rand.NewSource(rand.Int63()))
		//loseData(n, k, encodedData)

		startTs := time.Now()
		turnAroundBytes := 0

		for int(time.Since(startTs).Seconds()) < 2 {
			for i := 0; i < 1024; i++ {
				loseDataSimple(n, k, encodedData, rd)
				decodeSimple(coder, n, k, encodedData)
				turnAroundBytes += chunkSize * k
			}
		}
		thrpt := float64(turnAroundBytes) / 1000000 / float64(time.Since(startTs).Nanoseconds()/1000000000)

		mtx.Lock()
		if v, ok := thrptMap[chunkSize]; ok {
			thrptMap[chunkSize] = v + thrpt
		} else {
			thrptMap[chunkSize] = thrpt
		}
		mtx.Unlock()

		// fmt.Printf("%d\t%d\t%d\t%.4f\n", n, k, chunkSize, thrpt)
	}
}

func BenchmarkParallel(n, k, nThreads int) {
	runtime.GOMAXPROCS(nThreads)
	fmt.Println(n, k, nThreads)
	thrptMap := make(map[int]float64)
	mtx := &sync.Mutex{}
	var wg = sync.WaitGroup{}

	for i := 0; i < nThreads; i++ {
		wg.Add(1)
		go BenchmarkDecode(n, k, thrptMap, mtx, &wg)
	}
	wg.Wait()

	keys := make([]int, 0)
	for k, _ := range thrptMap {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	for _, k := range keys {
		fmt.Printf("%d \t %.4f\n", k, thrptMap[k])
	}
}

func test() {

	coder, _ := reedsolomon.New(3, 1, reedsolomon.WithMaxGoroutines(1), reedsolomon.WithCauchyMatrix())
	var data = make([]byte, 1*3)
	data[0] = 'a'
	data[1] = 'b'
	data[2] = 'c'

	shards, _ := coder.Split(data)
	fmt.Println(shards)
	_ = coder.Encode(shards)
	fmt.Println(shards)
}

func main() {
	numcpu := runtime.NumCPU()
	runtime.GOMAXPROCS(numcpu)

	test()
	BenchmarkParallel(4, 3, 1)
	BenchmarkParallel(4, 3, 8)
	// BenchmarkParallel(4 ,3, 2)
	// BenchmarkParallel(4 ,3, 4)
	// BenchmarkParallel(4 ,3, 8)
	// BenchmarkParallel(4 ,3, 16)
}
