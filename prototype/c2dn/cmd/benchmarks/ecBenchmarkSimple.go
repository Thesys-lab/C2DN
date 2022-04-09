package main

import (
	"github.com/klauspost/reedsolomon"
	"log"
	"math"
	"math/rand"
	"time"
)

/*
One process 4,3,1 Decoding
128	798.1498
256	1547.8161
512	2930.5602
1024	5378.8803
2048	9177.6614
4096	13663.1550
8192	18094.2275
16384	20980.7475
32768	22870.7009
65536	24541.7116
131072	25326.8853
262144	24199.4564
524288	15059.2291
1048576	12562.7793
2097152	12562.7793
4194304	11596.4117
8388608	9019.4313


one process 8,7,1 Decoding
128	1403.5976
256	2675.3499
512	4816.7125
1024	8016.4159
2048	13060.1189
4096	16846.8414
8192	20510.9854
16384	22795.2034
32768	23629.0310
65536	23417.6381
131072	23347.1738
262144	19166.2916
524288	15408.1952
1048576	12777.5277
2097152	12025.9084
4194304	9566.0635
8388608	7842.9838


one process 5,3,2 Decoding
128	590.9643
256	1140.2871
512	2119.7488
1024	3713.5319
2048	5940.7073
4096	8270.1189
8192	10238.7155
16384	11470.5826
32768	11933.6337
65536	12492.3150
131072	12623.1773
262144	10750.8400
524288	7489.3492
1048576	6442.4509
2097152	6442.4509
4194304	5798.2058
8388608	4123.1686




one process 9,7,2 Decoding
128	870.5278
256	1671.0500
512	3056.0223
1024	5188.3016
2048	8188.5397
4096	10904.3515
8192	13529.1470
16384	15279.0106
32768	15913.1894
65536	15619.5881
131072	14186.8138
262144	12777.5277
524288	12213.8132
1048576	11274.2892
2097152	10522.6699
4194304	9150.1477
8388608	7842.9838




*/

func BenchmarkDecodeSimple(n, k, testTime int) {
	log.Printf("benchmark start\n")
	nRand := 100000000
	rand.Seed(time.Now().UnixNano())
	rd := rand.New(rand.NewSource(rand.Int63()))
	var randIntSlice []int
	for i := 0; i < nRand; i++ {
		randIntSlice = append(randIntSlice, int(rd.Int31n(int32(n))))
	}
	log.Printf("all random number generated\n")

	// coder, _ := reedsolomon.New(k, n-k)
	coder, _ := reedsolomon.New(k, n-k, reedsolomon.WithMaxGoroutines(1))
	// coder, _ := reedsolomon.New(k, n-k, reedsolomon.WithMaxGoroutines(1), reedsolomon.WithCauchyMatrix())
	// coder, _ := reedsolomon.New(n-k, k, reedsolomon.WithMaxGoroutines(4), reedsolomon.WithCauchyMatrix(), reedsolomon.WithMinSplitSize(1024))

	for chunkSize := 128; chunkSize < int(math.Pow(2, 24)); chunkSize *= 2 {
		encodedData := encodeSimple(coder, k, chunkSize)

		startTs := time.Now()
		turnAroundBytes := 0
		var idx int64 = 0

		for int(time.Since(startTs).Seconds()) < testTime {
			for i := 0; i < 1024; i++ {
				loseDataSimple2(n, k, encodedData, randIntSlice[idx])
				decodeSimple(coder, n, k, encodedData)
				turnAroundBytes += chunkSize * k
				idx = (idx + 1) % int64(nRand)
			}
		}
		thrpt := float64(turnAroundBytes) / 1000000 / float64(time.Since(startTs).Nanoseconds()/1000000000)

		log.Printf("%d\t%d\t%d\t%.4f\n", n, k, chunkSize, thrpt)
	}
}

func BenchmarkEncodeSimple(n, k, testTime int) {
	log.Printf("benchmark start\n")
	rand.Seed(time.Now().UnixNano())
	nData := 20000

	// coder, _ := reedsolomon.New(k, n-k)
	coder, _ := reedsolomon.New(k, n-k, reedsolomon.WithMaxGoroutines(1))
	// coder, _ := reedsolomon.New(k, n-k, reedsolomon.WithMaxGoroutines(1), reedsolomon.WithCauchyMatrix())
	// coder, _ := reedsolomon.New(n-k, k, reedsolomon.WithMaxGoroutines(4), reedsolomon.WithCauchyMatrix(), reedsolomon.WithMinSplitSize(1024))

	for chunkSize := 128; chunkSize < int(math.Pow(2, 24)); chunkSize *= 2 {
		var dataSlice [][]byte
		for i := 0; i < nData; i++ {
			data := make([]byte, chunkSize*k)
			rand.Read(data)
			dataSlice = append(dataSlice, data)
		}

		startTs := time.Now()
		turnAroundBytes := 0
		var idx int64 = 0

		for int(time.Since(startTs).Seconds()) < testTime {
			for i := 0; i < 1024; i++ {
				var data = dataSlice[idx]
				shards, _ := coder.Split(data)
				_ = coder.Encode(shards)
				turnAroundBytes += chunkSize * k
				idx = (idx + 1) % int64(nData)
			}
		}
		thrpt := float64(turnAroundBytes) / 1000000 / float64(time.Since(startTs).Nanoseconds()/1000000000)

		log.Printf("%d\t%d\t%d\t%.4f\n", n, k, chunkSize, thrpt)
	}
}
