package main

import (
	"bytes"
	"fmt"
	"github.com/valyala/fasthttp"
	"io"
	"net/http"
	"sync"
	"time"
)

type FrontendParam struct {
	Mode     string
	NServers int
	Hosts    []string
	NodeIdx  int

	ECN                    int
	ECK                    int
	CodingObjSizeThreshold int64

	RamCacheSize      int64
	RamCacheThreshold int64
	RepFactor         int

	UnavailTrace       string
	UnavailUpdateIntvl int
	//LB                 *loadbalancer.ChBalancer
}

type FrontendLatencyChan struct {
	Started bool

	Temp  chan float64
	TempS chan interface{}

	RamCache    chan float64
	NoDecodeHit chan float64
	DecodeHit   chan float64
	FullObjHit  chan float64

	NoDecodeMiss chan float64
	DecodeMiss   chan float64
	FullObjMiss  chan float64

	FetchLocalHeaderHit  chan float64
	FetchLocalHeaderMiss chan float64
	FetchPeerHeader      chan float64
}

// this represents the data used when handling one request from client
type FrontendTxnData struct {
	TxnID     uint64
	TxnIDByte []byte
	Ctx       *fasthttp.RequestCtx
	Pipew     *io.PipeWriter

	StartTs time.Time

	Mtx  sync.Mutex
	Cond *sync.Cond

	Route       string
	ReqContent  string
	Bucket      string
	RAMCacheKey []byte

	NFrontendMain int

	SaveToFeRAM bool
	ShouldCode  bool

	ObjType   string
	ObjSize   int64
	ChunkSize int64
	SendSize  int64
	Remote    bool

	// fetching chunks related
	// the index of localhost in ec, should be 0 or 1
	localECIdx int

	localHit         bool
	alternateLeadHit bool

	// peer check results
	peerCheckFinished int32
	nPeerHit          int32
	IsPeerHit         []bool

	PeerIdx []int
	Resps   []*http.Response
	//RespReaders []io.ReadCloser

	// this is the host where the first request is sent to, it may not be local if there is unavailability
	FirstReqHostIdx int
	FirstReqResp    *http.Response
	ChunkStat       FETxnChunkStat

	DebugString string
}

type FETxnChunkStat struct {
	NumAvailable     int32
	NumDataAvailable int32
	NumSkipped       int32
	NumFailed        int32

	HasMissingChunk bool

	//NumDChunkHit  int32
	//NumPChunkHit  int32
	//NumDChunkMiss int32
	//NumPChunkMiss int32

	//LocalHit bool

	//NumRamHit   int
	MissingPChunkIdx []int
	IsAvailable      []bool
}

//type FrontendStat struct {
//	NumPartialHitGood int64 // partial hit and at least ECK chunks
//	NumPartialHitBad  int64 // partial hit and less than ECK chunks
//	NumAllHit         int64
//	NumAllMiss        int64
//
//	NumFullObjHit  int64
//	NumFullObjMiss int64
//
//	TrafficToClient   int64
//	TrafficFromOrigin int64
//	PushTraffic       int64
//	PeerGetTraffic    int64
//
//	NumMissXChunk []int64
//}
//
//type FrontendErrStat struct {
//	//NumCopyFromPiperErr int64
//	//NumCopyToPipewErr int64
//
//	//NumToClientErr        int64
//
//	NumLocalReadErr int64
//	NumRAMCacheErr  int64
//	//NumFailedClientReq    int64
//
//	//NumFromATSErr         int64
//	NumDataTransferErr    int64
//	NumStatusIncorrectErr int64
//	NumCodingErr          int64
//
//	NumFirstPushFail    int64
//	NumFinalPushFail    int64
//	NumPushFailureBytes int64
//}

type ObjInServing struct {
	ReqCnt  int
	Content *bytes.Buffer
}

func (feTxn *FrontendTxnData) String() (s string) {
	return fmt.Sprintf("txn %v: req %v/%v, txnTime %v, chunkServerIdx %v, objType %v, "+
		"availData/avail/skip/fail %v/%v/%v/%v, client %v, debug str %v",
		feTxn.TxnID, feTxn.Route, feTxn.ReqContent, time.Since(feTxn.StartTs), feTxn.PeerIdx, feTxn.ObjType,
		feTxn.ChunkStat.NumDataAvailable, feTxn.ChunkStat.NumAvailable, feTxn.ChunkStat.NumSkipped, feTxn.ChunkStat.NumFailed,
		fmt.Sprintf("%v", feTxn.Ctx.RemoteAddr()), feTxn.DebugString)
}
