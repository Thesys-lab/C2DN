package main

import (
	"github.com/klauspost/reedsolomon"
	"log"
	"math/rand"
)

func encodeSimple(coder reedsolomon.Encoder, k, chunkSize int) (shards [][]byte) {
	var data = make([]byte, chunkSize*k)
	rand.Read(data)

	shards, _ = coder.Split(data)
	_ = coder.Encode(shards)
	ok, _ := coder.Verify(shards)
	if !ok {
		log.Fatal("error")
	}
	return shards
}

func loseDataSimple(n, k int, shards [][]byte, rd *rand.Rand) {

	if n-k == 1 {
		l := rd.Intn(n)
		shards[l] = shards[l][:0]
	} else {
		for i := 0; i < n-k; i++ {
			l := rd.Intn(n)
			if len(shards[l]) == 0 || shards[l] == nil {
				i--
			} else {
				shards[l] = shards[l][:0]
			}
		}
	}
}

func loseDataSimple2(n, k int, shards [][]byte, l int) {

	if n-k == 1 {
		shards[l] = shards[l][:0]
	} else {
		for i := 0; i < n-k; i++ {
			shards[l] = shards[l][:0]
			l = (l + 1) % n
		}
	}
}

func decodeSimple(coder reedsolomon.Encoder, n, k int, shards [][]byte) (fixedShards [][]byte) {
	_ = coder.Reconstruct(shards)
	return shards
}
