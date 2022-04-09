package main

import (
	"fmt"

	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
	"io"
	"io/ioutil"
	"math/rand"
	"strings"
	"time"
)

func CheckObjCachePolicy(feTxn *FrontendTxnData) {

	feTxn.SaveToFeRAM = myutils.CheckRamAdmitPolicy(feParam.RamCacheSize, feTxn.ObjSize)
	if feParam.ECN == 2 && feParam.ECK == 1 {
		feTxn.ShouldCode = false
	} else {
		feTxn.ShouldCode = myutils.CheckCodingPolicy(feTxn.ObjSize)
	}
}

func saveToFeRAM(feTxn *FrontendTxnData, dat []byte) {
	//_, _ = buf.Write(FixSizedHeader)
	//if buf.Len() <= HttpHeaderSize {
	//	slogger.DPanicf("resp is %v after writing fixed header when saving to frontend RAM", buf.Len())
	//}

	err := C2DNRamCache.Set(feTxn.RAMCacheKey, dat, 0)
	if err != nil {
		slogger.DPanicf("%v, resp size %v, %v", err, len(dat), feTxn)
	}
}

func appendDebugString(feTxn *FrontendTxnData, s string) {
	if myconst.DebugLevel >= myconst.PerReqLogging {
		feTxn.Mtx.Lock()
		if feTxn.DebugString == "" {
			feTxn.DebugString = fmt.Sprintf("txn%d: %s/%s: ", feTxn.TxnID, feTxn.Route, feTxn.ReqContent)
		}
		feTxn.DebugString += fmt.Sprintf("%s_%vms | ", s, time.Since(feTxn.StartTs).Milliseconds())
		feTxn.Mtx.Unlock()
	}
}

func startLatRecord() {
	//wg := sync.WaitGroup{}
	//wg.Add(8)
	slogger.Infof("frontend start record stat")

	LatChans.Started = true
	LatChans.RamCache = make(chan float64, myconst.ChanBufLen)
	LatChans.NoDecodeHit = make(chan float64, myconst.ChanBufLen)
	LatChans.NoDecodeMiss = make(chan float64, myconst.ChanBufLen)
	LatChans.DecodeHit = make(chan float64, myconst.ChanBufLen)
	LatChans.DecodeMiss = make(chan float64, myconst.ChanBufLen)
	LatChans.FullObjHit = make(chan float64, myconst.ChanBufLen)
	LatChans.FullObjMiss = make(chan float64, myconst.ChanBufLen)
	LatChans.FetchLocalHeaderHit = make(chan float64, myconst.ChanBufLen)
	LatChans.FetchLocalHeaderMiss = make(chan float64, myconst.ChanBufLen)
	LatChans.FetchPeerHeader = make(chan float64, myconst.ChanBufLen)

	go myutils.DumpFloatChan(LatChans.RamCache, "fe.latency.RamCache", nil)
	go myutils.DumpFloatChan(LatChans.NoDecodeHit, "fe.latency.NoDecodeHit", nil)
	go myutils.DumpFloatChan(LatChans.NoDecodeMiss, "fe.latency.NoDecodeMiss", nil)
	go myutils.DumpFloatChan(LatChans.DecodeHit, "fe.latency.DecodeHit", nil)
	go myutils.DumpFloatChan(LatChans.DecodeMiss, "fe.latency.DecodeMiss", nil)
	go myutils.DumpFloatChan(LatChans.FullObjHit, "fe.latency.FullObjHit", nil)
	go myutils.DumpFloatChan(LatChans.FullObjMiss, "fe.latency.FullObjMiss", nil)
	go myutils.DumpFloatChan(LatChans.FetchLocalHeaderHit, "fe.latency.FetchLocalHeaderHit", nil)
	go myutils.DumpFloatChan(LatChans.FetchLocalHeaderMiss, "fe.latency.FetchLocalHeaderMiss", nil)
	go myutils.DumpFloatChan(LatChans.FetchPeerHeader, "fe.latency.FetchPeerHeader", nil)

	LatChans.Temp = make(chan float64, myconst.ChanBufLen)
	go myutils.DumpFloatChan(LatChans.Temp, "fe.latency.temp", nil)
	LatChans.TempS = make(chan interface{}, myconst.ChanBufLen)
	go myutils.DumpChan(LatChans.TempS, "fe.latency.tempS", nil)

	LatChansRemote.Started = true
	LatChansRemote.RamCache = make(chan float64, myconst.ChanBufLen)
	LatChansRemote.NoDecodeHit = make(chan float64, myconst.ChanBufLen)
	LatChansRemote.NoDecodeMiss = make(chan float64, myconst.ChanBufLen)
	LatChansRemote.DecodeHit = make(chan float64, myconst.ChanBufLen)
	LatChansRemote.DecodeMiss = make(chan float64, myconst.ChanBufLen)
	LatChansRemote.FullObjHit = make(chan float64, myconst.ChanBufLen)
	LatChansRemote.FullObjMiss = make(chan float64, myconst.ChanBufLen)
	LatChansRemote.FetchLocalHeaderHit = make(chan float64, myconst.ChanBufLen)
	LatChansRemote.FetchLocalHeaderMiss = make(chan float64, myconst.ChanBufLen)
	LatChansRemote.FetchPeerHeader = make(chan float64, myconst.ChanBufLen)

	go myutils.DumpFloatChan(LatChansRemote.RamCache, "fe.latency.remote.RamCache", nil)
	go myutils.DumpFloatChan(LatChansRemote.NoDecodeHit, "fe.latency.remote.NoDecodeHit", nil)
	go myutils.DumpFloatChan(LatChansRemote.NoDecodeMiss, "fe.latency.remote.NoDecodeMiss", nil)
	go myutils.DumpFloatChan(LatChansRemote.DecodeHit, "fe.latency.remote.DecodeHit", nil)
	go myutils.DumpFloatChan(LatChansRemote.DecodeMiss, "fe.latency.remote.DecodeMiss", nil)
	go myutils.DumpFloatChan(LatChansRemote.FullObjHit, "fe.latency.remote.FullObjHit", nil)
	go myutils.DumpFloatChan(LatChansRemote.FullObjMiss, "fe.latency.remote.FullObjMiss", nil)
	go myutils.DumpFloatChan(LatChansRemote.FetchLocalHeaderHit, "fe.latency.remote.FetchLocalHeaderHit", nil)
	go myutils.DumpFloatChan(LatChansRemote.FetchLocalHeaderMiss, "fe.latency.remote.FetchLocalHeaderMiss", nil)
	go myutils.DumpFloatChan(LatChansRemote.FetchPeerHeader, "fe.latency.remote.FetchPeerHeader", nil)

	//wg.Wait()
}

// get the http host index that are responsible for storing the chunks of current request
// reqID is objID_objSize, does not contain akamai/
// the n here is not the n in EC
// here n is the number of total servers
func GetHostIndexForReq(n, m int, reqID []byte) (indexs []int) {
	slogger.DPanicf("need rewrite with load balancer")
	firstHostID := int(myutils.HashByte(reqID)) % n
	for i := 0; i < m; i++ {
		indexs = append(indexs, (firstHostID+i)%n)
	}
	return indexs
}

func ParseViaHeader(headerStr string) (cacheHit, getFromOrigin bool, cacheOp, serverOp uint8) {
	if headerStr == "" {
		if !myconst.UseOriginAsATS {
			slogger.DPanicf("empty via string %v", headerStr)
		}
		return true, false, 0, 0
	}

	idx := strings.LastIndex(headerStr, "[")
	var idx2 int
	viaStr := headerStr[idx+1:]
	idx2 = strings.Index(viaStr, "c")
	cacheOp = viaStr[idx2+1]
	idx2 = strings.Index(viaStr, "s")
	serverOp = viaStr[idx2+1]
	//idx2 = strings.Index(headerStr[idx+1:], "f")
	//cacheFillOp := headerStr[idx2+1]

	switch cacheOp {
	case 'M':
		cacheHit = false
	case 'H', 'R':
		cacheHit = true
	case 'A', 'S':
		cacheHit = false
		slogger.DPanicf("Cache status should not be A or S, via header %s %v", headerStr, cacheOp)
	case ' ':
		slogger.DPanicf("unknown cache status, via header %s %v", headerStr, cacheOp)
	default:
		slogger.DPanicf("unknown cache status, via header %s %v", headerStr, cacheOp)
	}

	switch serverOp {
	case 'S':
		getFromOrigin = true
	case ' ':
		getFromOrigin = false
	case 'N', 'E':
		slogger.DPanicf("unknown server status, via header %s\n", headerStr)
	default:
		slogger.DPanicf("unknown server status, via header %s\n", headerStr)
	}

	if cacheHit && getFromOrigin {
		slogger.DPanicf("cacheHit && getFromOrigin %v", headerStr)
	}

	//if (!cacheHit) && !getFromOrigin {
	//	/* this could happen when we check whether an object is cached on a server using head request */
	//	slogger.DPanicf("!cacheHit && !getFromOrigin %v", headerStr)
	//}

	return cacheHit, getFromOrigin, cacheOp, serverOp
}

func getRAMCacheStatStr() (statStr string) {
	return fmt.Sprintf(
		"cacheEntry: %v, HitCount %v, HitRate %v, LookupCount: %v, MissCount: %v, OverwriteCount: %v, EvictionCount; %v",
		C2DNRamCache.EntryCount(), C2DNRamCache.HitCount(), C2DNRamCache.HitRate(),
		C2DNRamCache.LookupCount(), C2DNRamCache.MissCount(), C2DNRamCache.OverwriteCount(), C2DNRamCache.EvacuateCount())
}

func getFrontendStatStr() (statStr string) {
	return "need to implement frontend stat str"
	//return fmt.Sprintf("req %v, %v partialHitGood %v partialHitBad %v allHit %v allMiss, nMissXchunks %v, "+
	//	"%v fullObjHit %v fullObjMiss, origin traffic %.2f GB, client traffic %.2f GB, push traffic %.2f GB, peerGet traffic %.2f GB",
	//	txnCounter, Stat.NumPartialHitGood, Stat.NumPartialHitBad, Stat.NumAllHit, Stat.NumAllMiss,
	//	Stat.NumMissXChunk, Stat.NumFullObjHit, Stat.NumFullObjMiss,
	//	float64(Stat.TrafficFromOrigin)/myconst.GB, float64(Stat.TrafficToClient)/myconst.GB,
	//	float64(Stat.PushTraffic)/myconst.GB, float64(Stat.PeerGetTraffic)/myconst.GB)
}

func getFrontendErrStatStr() (errStatStr string) {
	return "need to implement frontend err stat str"
	//return fmt.Sprintf("%v DataTransferErr, %v statusIncorrectErr, %v RAMcacheErr, %v firstPushFail, %v finalPushFail %v pushFailureBytes",
	//	ErrStat.NumDataTransferErr, ErrStat.NumStatusIncorrectErr, ErrStat.NumRAMCacheErr,
	//	ErrStat.NumFirstPushFail, ErrStat.NumFinalPushFail, ErrStat.NumPushFailureBytes)
}

func HandleFailedReq(feTxn *FrontendTxnData, errStr string, errType string, stopClientReq bool) {
	//s := fmt.Sprintf("txn %v: req %v, err %v, req duration %v", feTxn.TxnID, feTxn.ReqContent, errStr, time.Since(feTxn.StartTs))
	s := fmt.Sprintf("%s, %s", errStr, feTxn.DebugString)
	metricErr.WithLabelValues(errType).Inc()

	//atomic.AddInt64(errCounter, 1)

	if stopClientReq {
		_ = feTxn.Pipew.Close()
		feTxn.Ctx.SetConnectionClose()

		for i, _ := range feTxn.Resps {
			if feTxn.ChunkStat.IsAvailable[i] {
				if feTxn.Resps[i] == nil {
					slogger.DPanicf("txn %v: available response is nil", feTxn.TxnID)
				}
				_, _ = io.Copy(ioutil.Discard, feTxn.Resps[i].Body)
				_ = feTxn.Resps[i].Body.Close()
			}
		}

	}

	slogger.DPanicf(s)

	//if ErrStat.NumDataTransferErr > int64(0.1*float64(feTxn.TxnID)) && feTxn.TxnID > 2000 {
	//	slogger.DPanicf("frontend stop due to too many errors\n%v\n%v\n%v",
	//		getRAMCacheStatStr(), getFrontendStatStr(), getFrontendErrStatStr())
	//}
}

func SelectRandKPlusX(K, X int, indexes []int) (chunkIdx, hostIdx []int) {
	chunkIdx = make([]int, K+X)
	hostIdx = make([]int, K+X)

	p := rand.Perm(len(indexes))
	for i := 0; i < K+X; i++ {
		chunkIdx[i] = p[i]
		hostIdx[i] = indexes[p[i]]
	}

	return
}
