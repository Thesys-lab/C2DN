package main

import (
	"bytes"
	"fmt"
	"github.com/1a1a11a/c2dnPrototype/src/loadbalancer"
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
	"github.com/klauspost/reedsolomon"
	"io"
	"net/http"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/coocood/freecache"
	"github.com/valyala/fasthttp"
	"time"
)

const (
	HttpHeaderSize = 0
	//FixSizedHeader = bytes.Repeat([]byte("x"), HttpHeaderSize)
)

var (
	feParam FrontendParam

	feIP       string
	txnCounter uint64 = 0
	/* 	because warmup period is not real time replay, and does not need unavailability
	 *	so unavailTraceReplayStartTs indicates the timestamp when real_time replay (evaluation) starts */
	unavailTraceReplayStartTs int64 = -1

	C2DNRamCache   *freecache.Cache
	TxnFinishCache = freecache.NewCache(8 * 1024 * 1024)
)

var (

	//Stat           = &FrontendStat{}
	//ErrStat        = &FrontendErrStat{}
	LatChans       = &FrontendLatencyChan{}
	LatChansRemote = &FrontendLatencyChan{}
)

var (
	atsGetClient  *http.Client
	atsPushClient *http.Client
	lb            *loadbalancer.ParityBalancer
	//lb            *loadbalancer.Balancer

	myEncoder      reedsolomon.Encoder
	myStreamCoders sync.Pool
)

var (

//CurServingMtx    = sync.RWMutex{}
//CurServingObjBuf = make(map[string][]byte)
//CurServingObjCnt = make(map[string]int)
//CurrentServingObj = &sync.Map{}

)

func init() {
	var err error

	// this will actually find the local IP for EC2, but it's what we need
	// for cloudlab, I am not sure yet, this might find the first (external) ip
	if feIP, err = myutils.GetExternalIP(); err != nil {
		slogger.Panic("I am not able to get my external IP ", err)
	}

	tr := &http.Transport{
		MaxIdleConns:        myconst.MaxIdleConns,
		MaxIdleConnsPerHost: myconst.MaxIdleConnsPerHost,
		IdleConnTimeout:     myconst.HttpTimeOut * time.Second,
		//DisableCompression:  true,
	}

	atsGetClient = &http.Client{Transport: tr, Timeout: myconst.HttpTimeOut - 2*time.Second}
	atsPushClient = &http.Client{Transport: tr, Timeout: myconst.HttpTimeOut - 2*time.Second}

	LatChans.Started = false
	LatChansRemote.Started = false
}

func Run() {
	defer myutils.CatchPanic(slogger)
	var err error

	//Stat.NumMissXChunk = make([]int64, feParam.ECN+1)

	// stream decoder
	myStreamCoders = sync.Pool{
		New: func() interface{} {
			return NewStreamCoder(feParam.ECN, feParam.ECK, int(myconst.CodingSubChunkSize))
		}}

	// encoder thread-safe
	if feParam.Mode == "C2DN" || feParam.Mode == "naiveCoding" {
		//myEncoder, err = reedsolomon.New(feParam.ECK, feParam.ECN-feParam.ECK,
		//	reedsolomon.WithMinSplitSize(131072), reedsolomon.WithCauchyMatrix())
		myEncoder, err = reedsolomon.New(feParam.ECK, feParam.ECN-feParam.ECK, reedsolomon.WithCauchyMatrix())
		if myEncoder == nil || err != nil {
			slogger.DPanicf("fail to create encoder %v", err)
		}
	}

	//initialize a RAM cache
	C2DNRamCache = freecache.NewCache(int(feParam.RamCacheSize))

	// to avoid frequent GC, set a low threshold to have GC run normally
	debug.SetGCPercent(100)

	slogger.Debug("frontend started")
	if err := fasthttp.ListenAndServe(":"+myconst.FEPort, requestHandler); err != nil {
		slogger.DPanicf("Error in start frontend server: %v", err)
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	txnID := atomic.AddUint64(&txnCounter, 1)
	if txnID%200000 == 1 {
		slogger.Infof("txn %v %v", txnID, getRAMCacheStatStr())
		slogger.Infof("txn %v %v", txnID, getFrontendStatStr())
		slogger.Infof("txn %v %v", txnID, getFrontendErrStatStr())
	}

	urlSplit := bytes.Split(ctx.Path(), []byte{'/'})
	// urlSplit[0] is empty
	if len(urlSplit) <= 2 {
		rootHandler(ctx)
	} else {
		if bytes.Equal(urlSplit[1], []byte("akamai")) || bytes.Equal(urlSplit[1], []byte("remote")) {
			akamaiHandler(ctx, txnID)
		} else if bytes.Equal(urlSplit[1], []byte("test")) {
			statHandler(ctx, txnID)
		} else if bytes.Equal(urlSplit[1], []byte("stat")) {
			statHandler(ctx, txnID)
		} else if bytes.Equal(urlSplit[1], []byte("setRecordLat")) {
			startLatRecord()
			msg := "frontend starts to record latency\n"
			_, _ = fmt.Fprintf(ctx, msg)
			slogger.Info(msg)
		} else if bytes.Equal(urlSplit[1], []byte("startUnavailReplay")) {
			unavailTraceReplayStartTs = time.Now().Unix()
			lb.Reset()
			msg := "frontend starts to replay unavailability trace, this indicates warmup has finished\n"
			slogger.Info(msg)
			_, _ = fmt.Fprintf(ctx, msg)
		} else if bytes.Equal(urlSplit[1], []byte("reset")) {
			resetFEStat()
			statHandler(ctx, txnID)
			msg := "reset frontend stat"
			slogger.Info(msg)
			_, _ = fmt.Fprintf(ctx, msg)
		} else if bytes.Equal(urlSplit[1], []byte("gc")) {
			runtime.GC()
		} else {
			rootHandler(ctx)
		}
	}
}

func akamaiHandler(ctx *fasthttp.RequestCtx, txnID uint64) {
	startTs := time.Now()
	url := string(ctx.Path())
	var remote bool

	var route string
	var req string

	urlSplit := bytes.Split(ctx.Path(), []byte{'/'})
	if bytes.Equal(urlSplit[1], []byte("remote")) {
		if len(urlSplit) < 4 || !bytes.Equal(urlSplit[2], []byte("akamai")) {
			slogger.DPanicf("err in route %s", url)
			rootHandler(ctx)
			return
		}
		route = "remote/akamai"
		req = string(urlSplit[3])
		remote = true
	} else {
		if len(urlSplit) < 3 || !bytes.Equal(urlSplit[1], []byte("akamai")) {
			slogger.DPanicf("err in route %s", url)
			rootHandler(ctx)
			return
		}

		route = "akamai"
		req = string(urlSplit[2])
		remote = false
	}

	reqSplit := strings.Split(req, "_")
	sz, _ := strconv.Atoi(reqSplit[1])

	metricByteClient.WithLabelValues("allToClient").Add(float64(sz))
	metricReqClient.WithLabelValues("allToClient").Inc()

	if !lb.IsUnavailable(feParam.NodeIdx) && serverFromDRAMCache(ctx, startTs, remote, sz) {
		/* served from DRAM cache */
		return
	}

	piper, pipew := io.Pipe()
	ctx.SetBodyStream(piper, -1)

	feTxn := &FrontendTxnData{
		TxnID: txnID, Route: route, ReqContent: req, Ctx: ctx, ObjSize: int64(sz),
		ObjType: "", NFrontendMain: 0,
		StartTs: startTs, Pipew: pipew, PeerIdx: nil, SendSize: 0, Remote: remote,
		RAMCacheKey: append([]byte(nil), ctx.RequestURI()...),
	}
	feTxn.Cond = sync.NewCond(&(feTxn.Mtx))

	// get peers
	feTxn.Bucket = string(ctx.Request.Header.Peek("Bucket"))
	if len(feTxn.Bucket) == 0 && feTxn.ReqContent[:1] != "a" {
		slogger.DPanicf("client does not provide bucket information %v %v", ctx.Request.Header.String(), feTxn)
		_ = pipew.Close()
		return
	}
	if myconst.DebugLevel > myconst.PerReqDetailedLogging {
		slogger.Debugf("akamaiHandler: start req %v: %v - bucket %v", txnID, url, feTxn.Bucket)
	}

	//var hashKey = bucket
	//var peers []string
	//if unavailTraceReplayStartTs <= 0 {
	//	/* this is warm up */
	//	peers = lb.GetNodes(0, hashKey, feParam.ECN)
	//} else {
	//	peers = lb.GetNodes(time.Now().Unix()-unavailTraceReplayStartTs, hashKey, feParam.ECN)
	//}
	//
	//for _, hostIdxStr := range peers {
	//	hostIdx, _ := strconv.Atoi(hostIdxStr)
	//	feTxn.PeerIdx = append(feTxn.PeerIdx, hostIdx)
	//}

	bucketInt, err := strconv.Atoi(feTxn.Bucket)
	if err != nil {
		slogger.DPanicf("fail to convert bucket \"%s\"", feTxn.Bucket)
		return
	}
	if unavailTraceReplayStartTs <= 0 {
		/* this is warm up */
		feTxn.PeerIdx = lb.GetNodesFromMapping(0, bucketInt)
	} else {
		feTxn.PeerIdx = lb.GetNodesFromMapping(time.Now().Unix()-unavailTraceReplayStartTs, bucketInt)
	}
	appendDebugString(feTxn, fmt.Sprintf("bucket%v-peers-%v", feTxn.Bucket, feTxn.PeerIdx))

	FrontendMain(feTxn)
}

func statHandler(ctx *fasthttp.RequestCtx, txnID uint64) {
	slogger.Info("txn ", txnID, " ", getRAMCacheStatStr())
	slogger.Info("txn ", txnID, " ", getFrontendStatStr())
	slogger.Info("txn ", txnID, " ", getFrontendErrStatStr())

	_, _ = fmt.Fprintf(ctx, "txn %d %s\n%s\n%s\n", txnID, getRAMCacheStatStr(),
		getFrontendStatStr(), getFrontendErrStatStr())
}

func rootHandler(ctx *fasthttp.RequestCtx) {
	_, _ = fmt.Fprintf(ctx, "Hello, world I am Frontend!\n\n")
}

func resetFEStat() {
	C2DNRamCache.ResetStatistics()
	metricByteClient.Reset()
	metricReqClient.Reset()
	metricTraffic.Reset()
	metricReq.Reset()
	metricTraffic.Reset()
	metricEC.Reset()
	metricErr.Reset()

	//for i := 0; i < len(Stat.NumMissXChunk); i++ {
	//	Stat.NumMissXChunk[i] = 0
	//}
	//Stat.TrafficFromOrigin = 0
	//Stat.NumFullObjMiss = 0
	//Stat.NumFullObjHit = 0
	//Stat.NumAllMiss = 0
	//Stat.NumAllHit = 0
	//Stat.NumPartialHitGood = 0
	//Stat.NumPartialHitBad = 0
	//
	//ErrStat.NumStatusIncorrectErr = 0
	////ErrStat.NumFromATSErr = 0
	////ErrStat.NumToClientErr = 0
	//ErrStat.NumDataTransferErr = 0
	//ErrStat.NumLocalReadErr = 0
	//ErrStat.NumRAMCacheErr = 0
	//ErrStat.NumCodingErr = 0
	//ErrStat.NumFirstPushFail = 0
	//ErrStat.NumFinalPushFail = 0
	//ErrStat.NumPushFailureBytes = 0
}
