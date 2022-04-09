package main

import (
	"bufio"
	"fmt"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"

	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/montanaflynn/stats"
	"log"
	"os"
	"os/user"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

func prepareLatChan(wg *sync.WaitGroup) (latChans *ClientLatencyChans) {

	latChans = &ClientLatencyChans{}

	latChans.FirstByteRAM = make(chan LatencyResult, myconst.ChanBufLen)
	latChans.FullRespRAM = make(chan LatencyResult, myconst.ChanBufLen)
	latChans.FirstByteHit = make(chan LatencyResult, myconst.ChanBufLen)
	latChans.FullRespHit = make(chan LatencyResult, myconst.ChanBufLen)
	latChans.FirstByteMiss = make(chan LatencyResult, myconst.ChanBufLen)
	latChans.FullRespMiss = make(chan LatencyResult, myconst.ChanBufLen)

	//latChans.FirstByteNoDecodeHit = make(chan LatencyResult, myconst.ChanBufLen)
	//latChans.FullRespNoDecodeHit = make(chan LatencyResult, myconst.ChanBufLen)
	//latChans.FirstByteNoDecodeMiss = make(chan LatencyResult, myconst.ChanBufLen)
	//latChans.FullRespNoDecodeMiss = make(chan LatencyResult, myconst.ChanBufLen)
	//latChans.FirstByteDecodeHit = make(chan LatencyResult, myconst.ChanBufLen)
	//latChans.FullRespDecodeHit = make(chan LatencyResult, myconst.ChanBufLen)
	//latChans.FirstByteDecodeMiss = make(chan LatencyResult, myconst.ChanBufLen)
	//latChans.FullRespDecodeMiss = make(chan LatencyResult, myconst.ChanBufLen)
	//latChans.FirstByteFullObjHit = make(chan LatencyResult, myconst.ChanBufLen)
	//latChans.FullRespFullObjHit = make(chan LatencyResult, myconst.ChanBufLen)
	//latChans.FirstByteFullObjMiss = make(chan LatencyResult, myconst.ChanBufLen)
	//latChans.FullRespFullObjMiss = make(chan LatencyResult, myconst.ChanBufLen)

	go DumpLatencyResultChan(latChans.FirstByteRAM, "client.latency.firstByte.RAM", wg)
	go DumpLatencyResultChan(latChans.FullRespRAM, "client.latency.fullResp.RAM", wg)

	go DumpLatencyResultChan(latChans.FirstByteHit, "client.latency.firstByte.Hit", wg)
	go DumpLatencyResultChan(latChans.FullRespHit, "client.latency.fullResp.Hit", wg)

	go DumpLatencyResultChan(latChans.FirstByteMiss, "client.latency.firstByte.Miss", wg)
	go DumpLatencyResultChan(latChans.FullRespMiss, "client.latency.fullResp.Miss", wg)

	return latChans
}

//func dumpBucketData(wg *sync.WaitGroup) {
//	if wg != nil {
//		wg.Add(1)
//		defer wg.Done()
//	}
//
//	slogger.Debugf("dump bucket data starts")
//
//	_ = os.Remove("client.bucket.firstByte")
//	if _, err := os.Create("client.bucket.firstByte"); err != nil {
//		slogger.Fatal(err)
//	}
//	_ = os.Remove("client.bucket.fullResp")
//	if _, err := os.Create("client.bucket.fullResp"); err != nil {
//		slogger.Fatal(err)
//	}
//
//	fileFB, err := os.OpenFile("client.bucket.firstByte", os.O_WRONLY, 0644)
//	fileFR, err := os.OpenFile("client.bucket.fullResp", os.O_WRONLY, 0644)
//	writerFB := bufio.NewWriter(fileFB)
//	writerFR := bufio.NewWriter(fileFR)
//	if err != nil {
//		slogger.Fatal(err)
//	}
//	defer fileFB.Close()
//	defer fileFR.Close()
//
//	var s = "#ts, bucketID(avgLatency, missCnt/ReqCnt, missByte/reqByte)"
//	if _, err := writerFB.WriteString(s + "\n"); err != nil {
//		slogger.Fatal("Error writing bucket data", err.Error())
//	}
//
//	if _, err := writerFR.WriteString(s + "\n"); err != nil {
//		slogger.Fatal("Error writing bucket data", err.Error())
//	}
//
//	time.Sleep(120 * time.Second)
//	for atomic.LoadInt64(&totalInTrafficByte) > 0 {
//		for time.Now().Unix()%100 != 0 {
//			time.Sleep(200 * time.Millisecond)
//		}
//
//		s = fmt.Sprintf("%v", time.Now().Unix())
//		bucketFBDataMapMtx.Lock()
//		for bucketID := range bucketFBDataMap {
//			bFB := bucketFBDataMap[bucketID]
//			s = fmt.Sprintf("%s, %v(%.2f, %v/%v, %v/%v)", s, bucketID, bFB.IntervalLatencySum/float64(bFB.IntervalReqCnt),
//				bFB.IntervalMissCnt, bFB.IntervalReqCnt, bFB.IntervalMissBytes, bFB.IntervalReqBytes)
//			bFB.IntervalLatencySum = 0
//			bFB.IntervalMissCnt = 0
//			bFB.IntervalReqCnt = 0
//			bFB.IntervalMissBytes = 0
//			bFB.IntervalReqBytes = 0
//		}
//		bucketFBDataMapMtx.Unlock()
//		if _, err := writerFB.WriteString(s + "\n"); err != nil {
//			slogger.Fatal("Error writing bucket data", err.Error())
//		}
//
//		s = fmt.Sprintf("%v", time.Now().Unix())
//		bucketFRDataMapMtx.Lock()
//		for bucketID := range bucketFRDataMap {
//			bFR := bucketFRDataMap[bucketID]
//			s = fmt.Sprintf("%s, %v(%.2f, %v/%v, %v/%v)", s, bucketID, bFR.IntervalLatencySum/float64(bFR.IntervalReqCnt),
//				bFR.IntervalMissCnt, bFR.IntervalReqCnt, bFR.IntervalMissBytes, bFR.IntervalReqBytes)
//			bFR.IntervalLatencySum = 0
//			bFR.IntervalMissCnt = 0
//			bFR.IntervalReqCnt = 0
//			bFR.IntervalMissBytes = 0
//			bFR.IntervalReqBytes = 0
//		}
//		bucketFRDataMapMtx.Unlock()
//		if _, err := writerFR.WriteString(s + "\n"); err != nil {
//			slogger.Fatal("Error writing bucket data", err.Error())
//		}
//		time.Sleep(time.Second * 95)
//	}
//
//	slogger.Debugf("Dump bucket data finishes")
//}

func DumpLatencyResultChan(c chan LatencyResult, filename string, wg *sync.WaitGroup) {
	if wg != nil {
		wg.Add(1)
		defer (*wg).Done()
	}

	filePath := myconst.OutputDir + "/" + filename
	_ = os.Remove(filePath)
	if _, err := os.Create(filePath); err != nil {
		slogger.Panic(err)
	}

	file, err := os.OpenFile(filePath, os.O_WRONLY, 0644)
	writer := bufio.NewWriter(file)
	if err != nil {
		fmt.Println(err)
		slogger.Panic(err)
		return
	}
	defer file.Close()

	for {
		e, ok := <-c
		if !ok {
			break
		}
		t := time.Now().Unix()
		s := fmt.Sprintf("%v %v %v\n", t, e.Req, e.Latency)
		if _, err := writer.WriteString(s); err != nil {
			slogger.Fatal("Error writing to file", filePath, err.Error())
		}

		if t%10 == 0 {
			_ = writer.Flush()
			_ = file.Sync()
		}
	}

	s := fmt.Sprintf("# %v %v\n", time.Now().Unix(), "end of output")
	if _, err := writer.WriteString(s); err != nil {
		slogger.Fatal("Error writing to file", filePath, err.Error())
	}

	if myconst.DebugLevel > 1 {
		slogger.Debugf("dumpFloatChan done for " + filePath)
	}
}

func calOutputStat(chanResult chan ClientStat, clientID int, workload string) (aggregatedResult ClientStat) {
	var latencyFirstByteSlice []float64
	var latencyFullRespSlice []float64
	var trafficInByte int64 = 0
	var runtime time.Duration = 0
	var throughput float64 = 0

	for {
		result, ok := <-chanResult
		if !ok {
			break
		} else {
			aggregatedResult.Concurrency += 1
			latencyFirstByteSlice = append(latencyFirstByteSlice, result.LatencyFirstByteSlice...)
			latencyFullRespSlice = append(latencyFullRespSlice, result.LatencyFullRespSlice...)
			trafficInByte += result.ReceivedTrafficInByte
			throughput += result.AchievedThroughput
			if result.FinishTs.Sub(result.StartTs) > runtime {
				runtime = result.FinishTs.Sub(result.StartTs)
			}
		}
	}

	if len(latencyFirstByteSlice) < 1000 {
		log.Fatal("not enough data points")
	}

	aggregatedResult.ClientID = clientID
	aggregatedResult.Workload = workload
	aggregatedResult.ReceivedTrafficInByte = trafficInByte
	aggregatedResult.AchievedThroughput = throughput
	aggregatedResult.Runtime = runtime

	sort.Float64s(latencyFirstByteSlice)
	sort.Float64s(latencyFullRespSlice)
	aggregatedResult.LatencyFirstByteMin, _ = stats.Min(latencyFirstByteSlice)
	aggregatedResult.LatencyFirstByteMax, _ = stats.Max(latencyFirstByteSlice)
	aggregatedResult.LatencyFirstByteMean, _ = stats.Mean(latencyFirstByteSlice)
	aggregatedResult.LatencyFirstByteMedian, _ = stats.Median(latencyFirstByteSlice)
	aggregatedResult.LatencyFirstByteP90 = latencyFirstByteSlice[int(float64(len(latencyFirstByteSlice))*0.9)]
	aggregatedResult.LatencyFirstByteP95 = latencyFirstByteSlice[int(float64(len(latencyFirstByteSlice))*0.95)]
	aggregatedResult.LatencyFirstByteP99 = latencyFirstByteSlice[int(float64(len(latencyFirstByteSlice))*0.99)]
	aggregatedResult.LatencyFirstByteP999 = latencyFirstByteSlice[int(float64(len(latencyFirstByteSlice))*0.999)]

	aggregatedResult.LatencyFullRespMin, _ = stats.Min(latencyFullRespSlice)
	aggregatedResult.LatencyFullRespMax, _ = stats.Max(latencyFullRespSlice)
	aggregatedResult.LatencyFullRespMean, _ = stats.Mean(latencyFullRespSlice)
	aggregatedResult.LatencyFullRespMedian, _ = stats.Median(latencyFullRespSlice)
	aggregatedResult.LatencyFullRespP90 = latencyFullRespSlice[int(float64(len(latencyFullRespSlice))*0.9)]
	aggregatedResult.LatencyFullRespP95 = latencyFullRespSlice[int(float64(len(latencyFullRespSlice))*0.95)]
	aggregatedResult.LatencyFullRespP99 = latencyFullRespSlice[int(float64(len(latencyFullRespSlice))*0.99)]
	aggregatedResult.LatencyFullRespP999 = latencyFullRespSlice[int(float64(len(latencyFullRespSlice))*0.999)]

	output := fmt.Sprintf("workload %s, client%d %d concurrency, finish using %.2f seconds, thrpt %f Gbps, latency mean %.2f/%.2f, median %.2f/%.2f, P90 %.2f/%.2f, P99 %.2f/%.2f, P999 %.2f/%.2f, %v failed requests",
		aggregatedResult.Workload, aggregatedResult.ClientID, aggregatedResult.Concurrency, float64(aggregatedResult.Runtime.Nanoseconds())/1000000000,
		aggregatedResult.AchievedThroughput, aggregatedResult.LatencyFirstByteMean, aggregatedResult.LatencyFullRespMean,
		aggregatedResult.LatencyFirstByteMedian, aggregatedResult.LatencyFullRespMedian,
		aggregatedResult.LatencyFirstByteP90, aggregatedResult.LatencyFullRespP90,
		aggregatedResult.LatencyFirstByteP99, aggregatedResult.LatencyFullRespP99,
		aggregatedResult.LatencyFirstByteP999, aggregatedResult.LatencyFullRespP999,
		myutils.GetCounterValue(clientReqMetric, []string{"failed"}),
	)
	slogger.Info(output)

	usr, _ := user.Current()
	dir := usr.HomeDir
	_, err := os.Stat(dir + "/client.result")
	if os.IsNotExist(err) {
		if _, err := os.Create(dir + "/client.result"); err != nil {
			slogger.DPanic(err)
		}
	}
	file, err := os.OpenFile(dir+"/client.result", os.O_APPEND|os.O_WRONLY, 0644)
	writer := bufio.NewWriter(file)
	if err != nil {
		slogger.DPanic(err)
		return
	}
	defer file.Close()
	if _, err := writer.WriteString(output + "\n"); err != nil {
		slogger.DPanic(err)
	}
	if err = writer.Flush(); err != nil {
		slogger.DPanic(err)
	}
	return aggregatedResult
}

func dumpClientStat(filename string, wg *sync.WaitGroup) {
	if wg != nil {
		wg.Add(1)
		defer (*wg).Done()
	}

	filePath := myconst.OutputDir + "/" + filename
	_ = os.Remove(filePath)
	if _, err := os.Create(filePath); err != nil {
		slogger.Fatal(err)
	}

	file, err := os.OpenFile(filePath, os.O_WRONLY, 0644)
	writer := bufio.NewWriter(file)
	if err != nil {
		slogger.Fatal(err)
		return
	}
	defer file.Close()

	startTs := time.Now()
	s := fmt.Sprintf("#client ts, trace ts: nReq/nRAM/nHit/nMiss, nByte/nRAM/nHit/nMiss" +
		"trafficInterval trafficTotal (GB), err\n")
	if _, err := writer.WriteString(s); err != nil {
		slogger.Fatal("Error writing to file", filePath, err.Error())
	}

	var lastBytes int64 = 0
	for atomic.LoadInt32(&replayFinished) == 0 {
		for time.Now().Unix()%10 != 0 {
			time.Sleep(60 * time.Millisecond)
		}
		totalBytes := int64(myutils.GetCounterValue(clientTrafficMetric, []string{"all"}))
		intvlBytes := totalBytes - lastBytes
		lastBytes = totalBytes
		s := fmt.Sprintf("%.0f, %d: %v/%v/%v/%v, %.2f/%.2f/%.2f/%.2f, %.2f/%.2f GiB, err %v\n",
			time.Since(startTs).Seconds(), traceTime,
			int(myutils.GetCounterValue(clientReqMetric, []string{"all"})),
			int(myutils.GetCounterValue(clientReqMetric, []string{"ramHit"})),
			int(myutils.GetCounterValue(clientReqMetric, []string{"hit"})),
			int(myutils.GetCounterValue(clientReqMetric, []string{"miss"})),
			myutils.GetCounterValue(clientTrafficMetric, []string{"all"})/myconst.GiB,
			myutils.GetCounterValue(clientTrafficMetric, []string{"ramHit"})/myconst.GiB,
			myutils.GetCounterValue(clientTrafficMetric, []string{"hit"})/myconst.GiB,
			myutils.GetCounterValue(clientTrafficMetric, []string{"miss"})/myconst.GiB,
			float64(intvlBytes)/myconst.GiB, float64(totalBytes)/myconst.GiB,
			int(myutils.GetCounterValue(clientReqMetric, []string{"failed"})))

		if _, err := writer.WriteString(s); err != nil {
			slogger.Fatal("Error writing to file", filePath, err.Error())
		}
		_ = writer.Flush()
		time.Sleep(8 * time.Second)
	}
	slogger.Debugf("client stat output finishes")
}

func ReportClientFinish() {
	_, err := os.Stat("./client.status")
	if os.IsNotExist(err) {
		if _, err := os.Create("./client.status"); err != nil {
			slogger.DPanic(err)
		}
	}

	file, err := os.OpenFile("./client.status", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		slogger.DPanic(err)
		return
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	s := fmt.Sprintf("client %v finishes trace_replay, "+
		"%d requesters, replay speed %v, start ts %v, end ts %v, "+
		"remoteOrigin %v, requestRate %v, dumpStat %t, elapsedTime %v, %.2f GB traffic, %v failedReq\n"+
		"%v req, %v RAM, %v hit, %v miss, %.2f GiB, %.2f RAM, %.2f hit %.2f miss",
		cParam.ClientID, cParam.Concurrency, cParam.ReplaySpeedup,
		cParam.ReplayStartTs, cParam.ReplayEndTs, cParam.RemoteOrigin,
		cParam.RequestRateMbps, myconst.DumpStat, time.Since(clientStartTs),
		myutils.GetCounterValue(clientTrafficMetric, []string{"all"})/float64(myconst.GiB),
		int64(myutils.GetCounterValue(clientTrafficMetric, []string{"failed"})),
		int(myutils.GetCounterValue(clientReqMetric, []string{"all"})),
		int(myutils.GetCounterValue(clientReqMetric, []string{"ramHit"})),
		int(myutils.GetCounterValue(clientReqMetric, []string{"hit"})),
		int(myutils.GetCounterValue(clientReqMetric, []string{"miss"})),
		myutils.GetCounterValue(clientTrafficMetric, []string{"all"})/myconst.GiB,
		myutils.GetCounterValue(clientTrafficMetric, []string{"ramHit"})/myconst.GiB,
		myutils.GetCounterValue(clientTrafficMetric, []string{"hit"})/myconst.GiB,
		myutils.GetCounterValue(clientTrafficMetric, []string{"miss"})/myconst.GiB)
	slogger.Info(s)
	if _, err := writer.WriteString(s + "\n"); err != nil {
		slogger.DPanic(err)
	}
	if err = writer.Flush(); err != nil {
		slogger.DPanic(err)
	}
}
