package main

import (
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
	"log"
	"strconv"
	"sync"
	"time"
)

func RunAkamai() {
	defer myutils.CatchPanic(slogger)
	wg := sync.WaitGroup{}

	if cParam.ClientID == -100 {
		slogger.Fatalf("negative clientID #{CParam.ClientID}")
	}

	chanReq := make(chan FullRequest, myconst.ChanBufLen)

	clientStartTs = time.Now()

	var latChans *ClientLatencyChans = nil
	if myconst.DumpStat {
		latChans = prepareLatChan(&wg)
	}

	//go dumpBucketData(&wg)

	// for monitoring
	go dumpClientStat("client.stat", &wg)

	for i := 0; i < cParam.Concurrency; i++ {
		go Requester("goroutine "+strconv.Itoa(i), chanReq, latChans, &wg)
		time.Sleep(50 * time.Millisecond)
	}

	go LoadAkamaiBinData(cParam.Trace, chanReq, &wg)
	// this is needed in case the goroutine has not started, so wg.Add is not executed
	time.Sleep(2 * time.Second)

	wg.Wait()

	//time.Sleep(8 * time.Second)
	time.Sleep(2 * time.Second)
	ReportClientFinish()
}

func RunSyntheticWorkload(nRequester int, clientID int, servers []string, workloadType string, dumpStat bool, workloadParam int) {
	wg := sync.WaitGroup{}

	chanReq := make(chan FullRequest, myconst.ChanBufLen)
	var latChans *ClientLatencyChans = nil

	if dumpStat {
		latChans = prepareLatChan(&wg)
		//go myutils.DumpThrpt(nRequester, throughputs, "HttpClient.traffic", &wg)
	}
	go dumpClientStat("client.stat", &wg)

	clientStartTs = time.Now()
	//wg.Add(nRequester + 1)
	switch workloadType {
	case "fixed":
		go LoadFixedData(2000000, 20, 128000, chanReq, &wg)
	case "random":
		go LoadRandomData(2000000, 20000, 128000, chanReq, &wg)
	case "allHit":
		go LoadAllHit(1000000, myconst.AllHitWorksetSize, myconst.SynWorkloadReqSize, 0, chanReq, &wg)
	case "allMiss":
		go LoadAllMiss(1000000, workloadParam, 0, 1, myconst.SynWorkloadReqSize, 0, chanReq, &wg)
	default:
		log.Fatal("unknown synthetic workload type ", workloadType)
	}
	for i := 0; i < nRequester; i++ {
		go Requester("goroutine "+strconv.Itoa(i), chanReq, latChans, &wg)
		time.Sleep(50 * time.Millisecond)
	}

	// this is needed in case the goroutine has started, so wg.Add is not executed
	time.Sleep(2 * time.Second)

	wg.Wait()
	ReportClientFinish()
}

func ThroughputClient(servers []string, workload string, workloadParam int, concurrency int, clientID, nClient int, reqSize int) {

	if reqSize <= 0 {
		reqSize = myconst.SynWorkloadReqSize
	}
	if nClient <= 0 {
		nClient = myconst.NumClients
	}

	wg := sync.WaitGroup{}
	chanReq := make(chan FullRequest, myconst.ChanBufLen)
	chanResult := make(chan ClientStat, concurrency)
	//throughputs := make([]int64, concurrency)

	if workload == "allHit" {
		n := 1000000
		go LoadAllHit(n, uint32(workloadParam), int(reqSize), 0, chanReq, &wg)

	} else if workload == "ramHit" {
		n := 1000000
		go LoadAllHit(n, uint32(workloadParam), int(reqSize), 0, chanReq, &wg)

	} else if workload == "diskHitWarmup" {
		go LoadAllMiss(workloadParam, 1, 0, 1, int(reqSize), 0, chanReq, &wg)

	} else if workload == "ramHitFetchDecode" || workload == "ramHitFetchNoDecode" || workload == "diskHitTest" {
		n := 100000
		go LoadAllHit(n, uint32(workloadParam), int(reqSize), 0, chanReq, &wg)

	} else if workload == "allHitRAM0" || workload == "diskHitFetchDecode" {
		n := 100000
		go LoadAllHit(n, uint32(workloadParam), int(reqSize), 0, chanReq, &wg)

	} else if workload == "allMiss" {
		n := 100000
		go LoadAllMiss(n, workloadParam, int(clientID), nClient, int(reqSize), 0, chanReq, &wg)

	} else {
		log.Fatal("unknown workload " + workload)
	}

	for i := 0; i < concurrency; i++ {
		go Requester("goroutine "+strconv.Itoa(i), chanReq, nil, &wg)
		time.Sleep(5 * time.Millisecond)
	}
	// this is needed in case the goroutine has started, so wg.Add is not executed
	time.Sleep(2 * time.Second)

	wg.Wait()
	close(chanResult)

	// Now calculate stat
	_ = calOutputStat(chanResult, clientID, workload)
	slogger.Infof("HttpClient %d: workload %s concurrency %d ThroughputClient has finished all the tests", clientID, workload, concurrency)
}
