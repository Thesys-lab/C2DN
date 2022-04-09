package main

import (
	"bufio"
	"fmt"
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
	"io"
	"io/ioutil"
	"math"
	"math/rand"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func ProcessHeader(headerStr string) (cacheHitType int) {
	if headerStr == "" {
		slogger.Errorf("empty via string %v", headerStr)
		clientStateMetric.WithLabelValues("errViaString").Inc()
		return -1
	}

	// 0 RAM, 1 ats disk Hit/Frontend noDecode Hit, 2 ats miss/frontend noDecode miss
	// 3 frontend decode hit, 4 frontend decode miss
	cacheHitType = 0
	idx := strings.LastIndex(headerStr, "[")
	var idx2 int
	viaStr := headerStr[idx+1:]
	idx2 = strings.Index(viaStr, "c")
	cacheOp := viaStr[idx2+1]

	switch cacheOp {
	case 'R':
		cacheHitType = myconst.RamHit
		//atomic.AddInt64(&nRAM, 1)
	case 'H': // ats hit or noDecode Hit
		cacheHitType = myconst.Hit
		//atomic.AddInt64(&nHit, 1)

	case 'A', 'M', 'S': // ats miss or noDecode Miss
		cacheHitType = myconst.Miss
		//atomic.AddInt64(&nMiss, 1)
	//case 'I': // Decode Hit
	//	cacheHitType = myconst.DecodeHit
	//	//atomic.AddInt64(&nDecodeHit, 1)
	//case 'N': // Decode Miss
	//	cacheHitType = myconst.DecodeMiss
	//	//atomic.AddInt64(&nDecodeMiss, 1)
	//
	//case 'X': // ARCH1 full obj Hit
	//	cacheHitType = myconst.FullObjHit
	//	//atomic.AddInt64(&nFullObjHit, 1)
	//case 'Y': // ARCH1 full obj Miss
	//	cacheHitType = myconst.FullObjMiss
	//	//atomic.AddInt64(&nFullObjMiss, 1)
	case ' ':
		slogger.DPanicf("unknown cache status, via header %s %v\n", headerStr, cacheOp)
	default:
		slogger.DPanicf("unknown cache status, via header %s %v\n", headerStr, cacheOp)
	}

	return cacheHitType
}

func HandleFailedReq(req *FullRequest, startTs time.Time, errStr string, err error) {
	clientReqMetric.WithLabelValues("failed").Inc()

	nFailed := myutils.GetCounterValue(clientReqMetric, []string{"failed"})
	nReq := myutils.GetCounterValue(clientReqMetric, []string{"all"})

	s := fmt.Sprintf("client %v: %v req %v, %v, reqStartTime %v, elapsed time %v, failed req %v/%v",
		cParam.ClientID, errStr, req, err, startTs.Unix(), time.Since(startTs), nFailed, nReq)
	slogger.Errorf(s)

	if nFailed/nReq > 0.1 && nReq > 2000 {
		ReportClientFinish()
		slogger.Panicf("client %v stops, %v failed requests/%v requests", cParam.ClientID, nFailed, nReq)
	}
}

func RequesterHTTP(goID string,
	reqChan chan FullRequest,
	latChans *ClientLatencyChans,
	wg *sync.WaitGroup) {

	if wg != nil {
		wg.Add(1)
		defer (*wg).Done()
	}

	//atomic.AddInt64(nRunRequesters, 1)
	clientStateMetric.WithLabelValues("running_worker").Inc()

	//localCounter := 0
	var err error
	var nByteFirstRead = 0
	var httpReq *http.Request
	var resp *http.Response

	var startTs time.Time
	var curTsSec uint32
	var latencyFirstByte, latencyFullResp float64 = 0, 0
	var bodySize uint32

	var latRes LatencyResult
	//var bFB *BucketFBData
	//var bFR *BucketFRData

	var hostIdx int = -1
	// we don't need true random
	//rg := rand.New(rand.NewSource(time.Now().UnixNano()))

	var host, requestStr, respHeaderStr string

	buf := make([]byte, 1024)

	requesterStartTs := time.Now()
	lastLogPrintTs = time.Now().Unix()
	for {
		req, ok := <-reqChan
		if !ok {
			break
		}

		traceTime = req.Timestamp
		latRes.Req = fmt.Sprintf("%d_%d", req.ID, req.Size)

		if cParam.Mode == "replayCloseloop" {
			// this must be warmup client
			slogger.Fatal("requester only supports real time, throughput workload is supported by " +
				"re-writing request timestamp at trace loading")
		} else {
			curTsSec = uint32(time.Since(requesterStartTs).Seconds())
			if req.Timestamp > curTsSec {
				x := int(req.Timestamp - curTsSec)
				// add some jitter
				x = 1000*x - rand.Intn(500) // sleep time randomization
				if myconst.DebugLevel >= myconst.PerReqDetailedLogging {
					slogger.Debugf("curr ts %v, req %v, wait %v ms", curTsSec, req, x)
				}
				time.Sleep(time.Millisecond * time.Duration(x))
			} else if req.Timestamp == 0 {
				// avoid starting tons of connections in a short time
				time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
			}
		}

		// because we move the load balancer and hashing into frontend and add frontend for both CDN and C2DN
		// the local client will always talk to local frontend
		// for remote client, it cannot talk to local frontend (because it does not have one and it can't have one)
		// so the remote client will use built-in load balancer
		if cParam.ClientID == -1 {
			if !req.Remote {
				slogger.DPanic("remote client gets non-remote requests")
				req.Remote = true
			}
			peers := lb.GetNodesFromMapping(int64(time.Since(requesterStartTs).Seconds()), int(req.Bucket))
			//peers := lbWithNoFailure.GetNodesFromMapping(int64(time.Since(requesterStartTs).Seconds()), int(req.Bucket))
			if cParam.RandomRoute {
				hostIdx = peers[rand.Intn(2)]
			} else {
				hostIdx = peers[0]
			}
			host = cParam.Hosts[hostIdx] // port is included
		} else {
			host = "127.0.0.1:" + myconst.FEPort
		}

		if myconst.UseLocalATSAsFE {
			host = "127.0.0.1:" + myconst.ATSPort
		}

		if req.Remote {
			requestStr = fmt.Sprintf("http://%s/remote/akamai/%d_%d", host, req.ID, req.Size)
		} else {
			requestStr = fmt.Sprintf("http://%s/akamai/%d_%d", host, req.ID, req.Size)
		}

		if myconst.DebugLevel >= myconst.PerReqDetailedLogging {
			slogger.Debugf("client %v %v read %v, req %v, %v requests in chan",
				cParam.ClientID, goID, req, requestStr, len(reqChan))
		}

		httpReq, err = http.NewRequest(http.MethodGet, requestStr, nil)
		if err != nil {
			slogger.DPanicf("req %v: failed to create new requests %v",
				myutils.GetCounterValue(clientReqMetric, []string{"all"}), err)
			continue
		}
		httpReq.Header.Add("Host", host)
		httpReq.Header.Add("Original-Host", strconv.Itoa(req.OriginalHostIdx))
		httpReq.Header.Add("Bucket", strconv.Itoa(int(req.Bucket)))

		startTs = time.Now()

		resp, err = HttpClient.Do(httpReq)

		if err != nil {

			HandleFailedReq(&req, startTs, fmt.Sprintf("txn %v: err sending requests",
				myutils.GetCounterValue(clientReqMetric, []string{"all"})), err)
			continue
		}

		bodySize, latencyFirstByte, latencyFullResp = 0, 0, 0
		respHeaderStr = resp.Header.Get("Via")

		//slogger.Debugf("response via header %v", respHeaderStr)

		hitType := ProcessHeader(respHeaderStr)
		bodySize = 0
		reader := bufio.NewReader(resp.Body)
		for {
			readSize, err := reader.Read(buf)
			bodySize += uint32(readSize)
			if latencyFirstByte == 0 {
				latencyFirstByte = float64(time.Since(startTs).Nanoseconds()) / 1000000.0
				latRes.Latency = latencyFirstByte
				nByteFirstRead = readSize
				if latChans != nil {
					switch hitType {
					case myconst.RamHit:
						latChans.FirstByteRAM <- latRes
					case myconst.Hit:
						latChans.FirstByteHit <- latRes
					case myconst.Miss:
						latChans.FirstByteMiss <- latRes
					case -1:
						slogger.DPanicf("req %v, unknown hitType %v during parsing response header %v, req %v",
							myutils.GetCounterValue(clientReqMetric, []string{"all"}), hitType,
							respHeaderStr, requestStr)
					}
				}

				//bucketFBDataMapMtx.Lock()
				//if bFB, ok = bucketFBDataMap[int(req.Bucket)]; !ok {
				//	bFB = &BucketFBData{}
				//	bucketFBDataMap[int(req.Bucket)] = bFB
				//}
				//bucketFBDataMapMtx.Unlock()
				//bFB.Mtx.Lock()
				//bFB.IntervalLatencySum += latencyFirstByte
				//bFB.IntervalReqCnt++
				//bFB.IntervalReqBytes += int64(req.Size)
				//if hitType == myconst.Miss || hitType == myconst.DecodeMiss || hitType == myconst.FullObjMiss {
				//	bFB.IntervalMissCnt++
				//	bFB.IntervalMissBytes += int64(req.Size)
				//}
				//bFB.Mtx.Unlock()
			}

			if err == io.EOF {
				latencyFullResp = float64(time.Since(startTs).Nanoseconds()) / 1000000.0
				latRes.Latency = latencyFullResp
				if latChans != nil {
					switch hitType {
					case myconst.RamHit:
						latChans.FullRespRAM <- latRes
						clientReqMetric.WithLabelValues("ramHit").Inc()
						clientTrafficMetric.WithLabelValues("ramHit").Add(float64(bodySize))
					case myconst.Hit:
						latChans.FullRespHit <- latRes
						clientReqMetric.WithLabelValues("hit").Inc()
						clientTrafficMetric.WithLabelValues("hit").Add(float64(bodySize))
					case myconst.Miss:
						latChans.FullRespMiss <- latRes
						clientReqMetric.WithLabelValues("miss").Inc()
						clientTrafficMetric.WithLabelValues("miss").Add(float64(bodySize))
					case -1:
						slogger.DPanicf("req %v, unknown hitType %v during parsing response header %v, req %v",
							myutils.GetCounterValue(clientReqMetric, []string{"all"}), hitType,
							respHeaderStr, requestStr)
					}
					clientReqMetric.WithLabelValues("all").Inc()
					clientTrafficMetric.WithLabelValues("all").Add(float64(bodySize))
				}

				//bucketFRDataMapMtx.Lock()
				//if bFR, ok = bucketFRDataMap[int(req.Bucket)]; !ok {
				//	bFR = &BucketFRData{}
				//	bucketFRDataMap[int(req.Bucket)] = bFR
				//}
				//bucketFRDataMapMtx.Unlock()
				//bFR.Mtx.Lock()
				//bFR.IntervalLatencySum += latencyFirstByte
				//bFR.IntervalReqCnt++
				//bFR.IntervalReqBytes += int64(req.Size)
				//if hitType == myconst.Miss || hitType == myconst.DecodeMiss || hitType == myconst.FullObjMiss {
				//	bFR.IntervalMissCnt++
				//	bFR.IntervalMissBytes += int64(req.Size)
				//}
				//bFR.Mtx.Unlock()

				//atomic.AddInt64(&totalInTrafficByte, int64(bodySize))
				_ = resp.Body.Close()
				break
			} else if err != nil {
				HandleFailedReq(&req, startTs, "err Reading response from "+host, err)
				_, _ = io.Copy(ioutil.Discard, resp.Body)
				_ = resp.Body.Close()
				break
			}

			if readSize == 0 {
				slogger.DPanicf("req %v: readSize = 0 %v", requestStr, err)
			}
		}

		if math.Abs(float64(bodySize-req.Size)) > 16 {
			HandleFailedReq(&req, startTs, fmt.Sprintf("req %v, req %v, err response size different %v/%v, respHeader %v",
				myutils.GetCounterValue(clientReqMetric, []string{"all"}),
				requestStr, bodySize, req.Size, respHeaderStr), nil)
		}

		nowSec := time.Now().Unix()
		if nowSec-lastLogPrintTs >= myconst.ClientReportInterval && nowSec != lastLogPrintTs {
			atomic.StoreInt64(&lastLogPrintTs, time.Now().Unix())
			slogger.Infof("Client %d req %v: %s: %v, first byte/ full resp: %.2f/%.2f ms, "+
				"firstReadBytes %v, %d request left, traffic %.2f GB, %d goroutines, %d error, last report ts %v/%v",
				cParam.ClientID, myutils.GetCounterValue(clientReqMetric, []string{"all"}), goID, req,
				latencyFirstByte, latencyFullResp, nByteFirstRead, len(reqChan),
				myutils.GetCounterValue(clientTrafficMetric, []string{"all"})/myconst.GB,
				runtime.NumGoroutine(),
				int(myutils.GetCounterValue(clientReqMetric, []string{"failed"})),
				nowSec-requesterStartTs.Unix(), lastLogPrintTs-requesterStartTs.Unix())
		}

		if cParam.Mode != "replayOpenloop" && cParam.ClientID != -1 && latencyFirstByte > 20 {
			slogger.Warnf("Client %d, latency %.2f/%.2f, req %v, hosts %v (%v)",
				cParam.ClientID, latencyFirstByte, latencyFullResp, req, hostIdx, host)
		}
	}

	//atomic.AddInt64(nRunRequesters, -1)
	clientStateMetric.WithLabelValues("running_worker").Dec()

	if int(myutils.GetGaugeValue(clientStateMetric, []string{"running_worker"})) == 0 {
		//if atomic.LoadInt64(nRunRequesters) == 0 {
		// all requesters have finished, we need to wait HTTP timeout
		// this causes problem with throughput calculation
		//time.Sleep(myconst.HttpTimeOut * time.Second)
		if latChans != nil {
			close(latChans.FirstByteRAM)
			close(latChans.FirstByteHit)
			close(latChans.FirstByteMiss)
			close(latChans.FullRespRAM)
			close(latChans.FullRespHit)
			close(latChans.FullRespMiss)
		}
		// this is used to tell dumpStat go routine that I have finished
		//totalInTrafficByte = -totalInTrafficByte
		atomic.StoreInt32(&replayFinished, 1)
	}

	slogger.Infof("client %d, finishes replay, %s startTime %d, elapsed time %v, finished %d requests (%.2f GiB), %d errors, %d requesters left",
		cParam.ClientID, goID, requesterStartTs.Unix(), time.Since(requesterStartTs),
		int(myutils.GetCounterValue(clientReqMetric, []string{"all"})),
		myutils.GetCounterValue(clientTrafficMetric, []string{"all"})/myconst.GiB,
		int(myutils.GetCounterValue(clientReqMetric, []string{"failed"})),
		int(myutils.GetGaugeValue(clientStateMetric, []string{"running_worker"})))
}

func Requester(requesterID string,
	reqChan chan FullRequest,
	latChans *ClientLatencyChans,
	wg *sync.WaitGroup) {

	//defer func() {
	//	if err := recover(); err != nil {
	//		slogger.DPanicf("panic: %v", err)
	//	}
	//}()

	RequesterHTTP(requesterID, reqChan, latChans, wg)
}
