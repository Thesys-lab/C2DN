package main

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"math"
	"runtime/debug"

	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
	"github.com/klauspost/reedsolomon"
	"github.com/valyala/fasthttp"
	"io"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

const (
	statName = "origin.stat"
)

type OriginParam struct {
	SetNoCacheHeader bool
	SleepTimeMs      time.Duration
}

const (
	Fixed_DATA_SIZE   = 4 * myconst.GB
	Fixed_PARITY_SIZE = 2 * myconst.GB
)

var (
	OParam                OriginParam
	LargeFixedDataBlock   = bytes.Repeat(append([]byte{'*'}), Fixed_DATA_SIZE)
	LargeFixedParityBlock map[string][]byte

	ParityByteMap = map[string]byte{
		"4_3": byte('\xd2'), "3_2": byte('\xf8'),
		"5_4": byte('\x29'), "6_4": byte('\x29'),
		"6_5": byte('\x03'), "8_7": byte('\xfb'),
		"12_11": byte('\x9c')}
)

var (
	nReq uint64 = 0
	//totalByte uint64 = 0
	//intvlByte uint64 = 0

	myEncoder reedsolomon.Encoder = nil
)

var (
	metricOriginByte = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "origin_byte",
		Help: "The number of bytes served by the origin",
	}, []string{"reqType"})

	metricOriginReq = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "origin_nReq",
		Help: "The number of requests served by the origin",
	}, []string{"reqType"})
)

func init() {
	LargeFixedParityBlock = make(map[string][]byte)
	LargeFixedParityBlock["4_3"] = bytes.Repeat(append([]byte{'\xd2'}), Fixed_PARITY_SIZE)
}

func Run() {
	defer myutils.CatchPanic(slogger)
	go dumpStat(statName)
	debug.SetGCPercent(10)
	slogger.Info("origin ready to serve")
	if err := fasthttp.ListenAndServe(":"+myconst.OriginPort, requestHandler); err != nil {
		slogger.Fatalf("Error in ListenAndServe: %s", err)
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {

	reqID := atomic.AddUint64(&nReq, 1)

	urlSplit := bytes.Split(ctx.Path(), []byte{'/'})
	objType := string(ctx.Request.Header.Peek("Obj-Type"))
	fakePush := bytes.Equal(ctx.Request.Header.Peek("Fake-Push"), []byte("true"))

	if myconst.DebugLevel >= myconst.PerReqDetailedLogging {
		slogger.Debugf("req %v, Obj-Type %s, noCacheSet %v, fakePush %v",
			string(ctx.Path()), objType, OParam.SetNoCacheHeader, fakePush)
	}

	if fakePush && !myconst.UseFakePush {
		slogger.DPanic("fake push is not enabled, but origin request has fakePush header")
	}

	if len(urlSplit) < 2 {
		rootHandler(ctx)
	} else {
		if fakePush {
			metricOriginReq.WithLabelValues("fakePush_" + string(urlSplit[1])).Inc()
		} else {
			metricOriginReq.WithLabelValues(string(urlSplit[1])).Inc()
		}

		if bytes.Equal(urlSplit[1], []byte("akamai")) {
			if !fakePush && OParam.SleepTimeMs > 0 {
				slogger.Debugf("sleep %v", OParam.SleepTimeMs)
				time.Sleep(OParam.SleepTimeMs)
				ctx.Response.Header.Set("Sleep-Time", fmt.Sprintf("%v", OParam.SleepTimeMs))
			}
			akamaiHandler(ctx, objType, fakePush)
		} else if bytes.Equal(urlSplit[1], []byte("stat")) {
			statHandler(ctx, reqID)
		} else if bytes.Equal(urlSplit[1], []byte("reset")) {
			statHandler(ctx, reqID)
			atomic.StoreUint64(&nReq, 0)
			metricOriginReq.Reset()
			metricOriginByte.Reset()
		} else if bytes.Equal(urlSplit[1], []byte("setSleepTime")) {
			if len(urlSplit) < 3 {
				slogger.Panic("setSleepTime cannot find time %v", urlSplit)
			}
			v, _ := strconv.Atoi(string(urlSplit[2]))
			OParam.SleepTimeMs = time.Duration(v) * time.Millisecond
			msg := fmt.Sprintf("set sleep time to %v\n", OParam.SleepTimeMs)
			_, _ = fmt.Fprintf(ctx, msg)
			slogger.Infof(msg)
		} else if bytes.Equal(urlSplit[1], []byte("setNoCache")) {
			OParam.SetNoCacheHeader = true
			var msg = "will add no cache header to each origin full object request\n"
			_, _ = fmt.Fprintf(ctx, msg)
			slogger.Infof(msg)
		} else if bytes.Equal(urlSplit[1], []byte("setCache")) {
			OParam.SetNoCacheHeader = false
			var msg = "will NOT add no cache header to each origin request\n"
			_, _ = fmt.Fprintf(ctx, msg)
			slogger.Info(msg)
		} else {
			rootHandler(ctx)
		}
	}
}

func akamaiHandler(ctx *fasthttp.RequestCtx, objType string, fakePush bool) {
	var objSize int
	var err1, err2, err3 error

	// uriSplit[0] is empty, uriSplit[1] == "akamai", uriSplit[2] == "coded" || objID_objSize
	// if uriSplit[2] == "coded", then uriSplit[3] == ECN_ECK, uriSplit[4]=chunkID, uriSplit[5]=objID-objSize
	uriSplit := strings.Split(string(ctx.Path()), "/")
	req := uriSplit[2]
	objInfo := strings.Split(req, "_")
	objSize, _ = strconv.Atoi(objInfo[1])
	// used Objsize not objSize because fastHTTP server does not support camelCase and change
	// to lower case with first letter being uppercase
	ctx.Response.Header.Set("Obj-Size", strconv.Itoa(objSize))

	if myconst.DebugLevel >= myconst.PerReqLogging {
		slogger.Debugf("req %v, objSize %v objType %v, noCacheSet %v, code %v, fakePush %v",
			uriSplit, objSize, objType, OParam.SetNoCacheHeader, myutils.CheckCodingPolicy(int64(objSize)), fakePush)
	}

	if objType == "chunk" {
		/* this is used when frontend sends a data chunk request to peer ATS, but the peer ATS does not have the chunk
		 * in other words, this is a partial hit (with some chunks being hit and some chunks being miss)
		 * or if fakePush is enabled,
		 * this is used to fake the push request
		 */
		var k, n, chunkID int
		chunkInfo := strings.Split(string(ctx.Request.Header.Peek("Ec-Chunk")), "_")
		n, err1 = strconv.Atoi(chunkInfo[0])
		k, err2 = strconv.Atoi(chunkInfo[1])
		chunkID, err3 = strconv.Atoi(chunkInfo[2])

		if err1 != nil || err2 != nil || err3 != nil {
			slogger.DPanicf("unknown url %v with ecChunk header %v",
				string(ctx.Path()), string(ctx.Request.Header.Peek("Ec-Chunk")))
		}

		ctx.Response.Header.Set("Obj-Type", "chunk")
		ctx.Response.Header.Set("Ec-Chunk", string(ctx.Request.Header.Peek("Ec-Chunk")))
		akamaiCodedHandler(ctx, n, k, chunkID, objSize, fakePush)
	} else {
		/* request full object, this is used when frontend sends requests to local ATS and local ATS does not have the object
		or fakePush is enabled and frontend sends a fakePush request to remote ATS peer to fake the full object is pushed to the peer
		*/

		/* this is to cooperate with frontend,
		 * if the object needs to be coded,
		 * we add no-cache header to avoid ATS storing it */
		if !fakePush && OParam.SetNoCacheHeader && myutils.CheckCodingPolicy(int64(objSize)) {
			ctx.Response.Header.Set("Cache-Control", "no-cache, no-store")
		}

		if fakePush {
			/* do not expose this metric, because it is not accurate, if ATS has the object, the fake push
			 * will not arrive at origin */
			//metricOriginByte.WithLabelValues("fakePush_fullObj").Add(float64(objSize))
		} else {
			metricOriginByte.WithLabelValues("fullObj").Add(float64(objSize))
		}
		ctx.Response.Header.Set("Obj-Type", "full")
		ctx.SetBody(LargeFixedDataBlock[:objSize])

		//if objSize < myconst.OriginStreamingChunkSize {
		//	ctx.SetBody(bytes.Repeat([]byte("*"), objSize))
		//} else {
		//	pr, pw := io.Pipe()
		//	ctx.SetBodyStream(pr, -1)
		//	go streamResp(pw, FixedDataBlock, '*', objSize, string(ctx.Path()))
		//}
	}
}

func akamaiCodedHandler(ctx *fasthttp.RequestCtx, n, k, chunkID, reqObjSize int, fakePush bool) {

	// encoder thread-safe
	//if myEncoder == nil {
	//	myEncoder, _ = reedsolomon.New(k, n-k, reedsolomon.WithMinSplitSize(131072), reedsolomon.WithCauchyMatrix())
	//}

	chunkSize := int(math.Ceil(float64(reqObjSize) / float64(k)))

	if fakePush {
		/* do not expose this metric, because it is not accurate, if ATS has the object, the fake push
		 * will not arrive at origin */
		//metricOriginByte.WithLabelValues("fakePush_chunkObj").Add(float64(chunkSize))
	} else {
		metricOriginByte.WithLabelValues("chunkObj").Add(float64(chunkSize))
	}

	scheme := fmt.Sprintf("%d_%d", n, k)

	if chunkID < k {
		ctx.SetBody(LargeFixedDataBlock[:chunkSize])
	} else {
		ctx.SetBody(LargeFixedParityBlock[scheme][:chunkSize])
	}

	//var blockData []byte
	//var singleByte byte

	//if chunkID < k {
	//	// data chunk
	//	blockData = FixedDataBlock
	//	singleByte = '*'
	//} else {
	//	// parity chunk
	//	blockData = FixedParityBlock[scheme]
	//	singleByte = ParityByteMap[scheme]
	//}

	//if chunkSize < int(myconst.OriginStreamingChunkSize) {
	//	ctx.SetBody(bytes.Repeat([]byte{singleByte}, chunkSize))
	//} else {
	//	pr, pw := io.Pipe()
	//	ctx.SetBodyStream(pr, -1)
	//	//ctx.SetBodyStream(pr, chunkSize)
	//	go streamResp(pw, blockData, singleByte, chunkSize, string(ctx.Path()))
	//}
}

func statHandler(ctx *fasthttp.RequestCtx, txnID uint64) {
	statStr := fmt.Sprintf("txn %v, traffic full/chunk %v/%v bytes, noCache %v, sleepTime %v",
		txnID,
		myutils.GetCounterValue(metricOriginByte, []string{"fullObj"}),
		myutils.GetCounterValue(metricOriginByte, []string{"chunkObj"}),
		OParam.SetNoCacheHeader, OParam.SleepTimeMs)
	slogger.Info(statStr)
	_, _ = fmt.Fprintf(ctx, statStr+"\n")
}

func rootHandler(ctx *fasthttp.RequestCtx) {
	_, _ = fmt.Fprintf(ctx, "Hello, world! This is origin")
}

func streamResp(writer *io.PipeWriter, blockData []byte, c byte, reqObjSize int, req string) {
	sizeLeft := reqObjSize
	for sizeLeft > int(myconst.OriginStreamingChunkSize) {
		if n, err := writer.Write(blockData); err != nil || n != int(myconst.OriginStreamingChunkSize) {
			slogger.Errorf("txn %v: write to client err %v, req %v, written %v bytes, should write %v",
				atomic.LoadUint64(&nReq), err, req, n, myconst.OriginStreamingChunkSize)
		}
		sizeLeft -= int(myconst.OriginStreamingChunkSize)
	}
	if n, err := writer.Write(bytes.Repeat([]byte{c}, sizeLeft)); err != nil || n != sizeLeft {
		slogger.Errorf("txn %v: write to client err %v, req %v, written %v bytes, should write %v",
			atomic.LoadUint64(&nReq), err, req, n, myconst.OriginStreamingChunkSize)
	}
	if err := writer.Close(); err != nil {
		slogger.DPanicf("txn %v: close pipewriter err %v, req %v", atomic.LoadUint64(&nReq), err, req)
	}
}

func dumpStat(filename string) {
	filepath := myconst.OutputDir + "/" + filename

	slogger.Infof("origin traffic monitor starts %v", filepath)
	_ = os.Remove(filepath)
	if _, err := os.Create(filepath); err != nil {
		slogger.DPanic(err)
	}

	file, err := os.OpenFile(filepath, os.O_WRONLY, 0644)
	writer := bufio.NewWriter(file)
	if err != nil {
		slogger.DPanic(err)
		return
	}
	defer file.Close()

	if _, err := writer.WriteString("ts: req, full/chunk_trafficAccu(MB)\n"); err != nil {
		slogger.DPanic("Error writing to file", filepath, err.Error())
	}

	for {
		for time.Now().Unix()%20 != 0 {
			time.Sleep(100 * time.Millisecond)
		}
		t1 := myutils.GetCounterValue(metricOriginByte, []string{"fullObj"}) / myconst.MiB
		t2 := myutils.GetCounterValue(metricOriginByte, []string{"chunkObj"}) / myconst.MiB
		s := fmt.Sprintf("%v: %v, %.2f/%.2f\n", time.Now().Unix(), nReq, t1, t2)
		if _, err := writer.WriteString(s); err != nil {
			slogger.DPanic("Error writing to file", filepath, err.Error())
		}
		_ = writer.Flush()
		time.Sleep(19 * time.Second)
	}
}
