package main

import (
	"log"
	"math/rand"
	"sync"
	"time"
)

func GenRandomWorkload(n int64, missRatio float64) (workload []string) {
	for i := int64(0); i < n; i++ {
		workload = append(workload, "a")
	}
	return workload
}

func LoadRandomData(n int, workingSetSize int, reqSizeMax int, c chan FullRequest, wg *sync.WaitGroup) {

	if wg != nil {
		wg.Add(1)
		defer (*wg).Done()
	}

	generatedReq := make(map[uint32]uint32)
	rand.Seed(time.Now().Unix())
	//randomSize := int32(float64(workingSetSize)*(1-missRatio))
	randomSize := int32(workingSetSize)

	req := FullRequest{}
	req.Timestamp = 0
	for i := 0; i < n; i++ {
		req.ID = uint32(rand.Int31n(randomSize-1) + 1)
		if sz, _ := generatedReq[req.ID]; sz != 0 {
			req.Size = sz
		} else {
			req.Size = uint32(rand.Int31n(int32(reqSizeMax-1)) + 1) // avoid 0, make it at least 8
			generatedReq[req.ID] = req.Size
		}
		if req.Size == 0 {
			log.Fatal("size 0", req)
		}
		c <- req
	}

	close(c)
	slogger.Info("all requests are generated")
}

func LoadFixedData(n int, workingSetSize int, reqSizeMax int, c chan FullRequest, wg *sync.WaitGroup) {

	if wg != nil {
		wg.Add(1)
		defer (*wg).Done()
	}

	generatedReq := make(map[uint32]uint32)
	rand.Seed(time.Now().Unix())
	var reqID uint32 = 0

	req := FullRequest{}
	req.Timestamp = 0
	for i := 0; i < n; i++ {
		reqID = (reqID + 1) % uint32(workingSetSize)
		req.ID = reqID
		if sz, _ := generatedReq[req.ID]; sz != 0 {
			req.Size = sz
		} else {
			req.Size = uint32(rand.Int31n(int32(reqSizeMax-1)) + 8) // avoid 0, make it at least 8
			generatedReq[req.ID] = req.Size
		}
		if req.Size == 0 {
			log.Fatal("size 0", req)
		}
		c <- req
	}

	close(c)
	slogger.Info("all requests are generated")
}

func LoadAllMiss(n int, startReqID int, clientID, nClients int, reqSize, reqSizeMax int, c chan FullRequest, wg *sync.WaitGroup) {

	if wg != nil {
		wg.Add(1)
		defer (*wg).Done()
	}

	var reqTs uint32 = 0
	if (reqSize <= 0 && reqSizeMax <= 0) || (reqSize > 0 && reqSizeMax > 0) {
		log.Fatal("please provide only reqSize or reqSizeMax, the other one should be 0")
	}

	req := FullRequest{}
	req.Timestamp = 0
	for i := 0; i < n; i++ {
		req.Timestamp = reqTs
		reqTs++
		req.ID = uint32(startReqID + i*nClients + clientID)
		if reqSize <= 0 {
			req.Size = uint32(rand.Int31n(int32(reqSizeMax-1)) + 8) // avoid 0, make it at least 8
		} else {
			req.Size = uint32(reqSize)
		}

		if req.Size == 0 {
			log.Fatal("size 0", req)
		}
		c <- req
	}

	close(c)
	slogger.Info("all requests are generated")
}

func LoadAllHit(n int, workingSetSize uint32, reqSize, reqSizeMax int, c chan FullRequest, wg *sync.WaitGroup) {

	if wg != nil {
		wg.Add(1)
		defer (*wg).Done()
	}

	rand.Seed(time.Now().UnixNano())
	var reqTs uint32 = 0
	if (reqSize <= 0 && reqSizeMax <= 0) || (reqSize > 0 && reqSizeMax > 0) {
		log.Fatal("please provide only reqSize or reqSizeMax, the other one should be 0")
	}

	req := FullRequest{}
	req.Timestamp = 0
	for i := 0; i < n; i++ {
		req.Timestamp = reqTs
		reqTs++
		req.ID = rand.Uint32()%workingSetSize + 1
		if reqSize <= 0 {
			req.Size = uint32(rand.Int31n(int32(reqSizeMax-1)) + 8) // avoid 0, make it at least 8
		} else {
			req.Size = uint32(reqSize)
		}
		if req.Size <= 0 {
			log.Fatal("size 0", req)
		}
		c <- req
	}

	close(c)
	slogger.Info("all requests are generated")
}
