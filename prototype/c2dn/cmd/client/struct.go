package main

import (
	"fmt"
	"github.com/montanaflynn/stats"
	"log"
	"sort"
	"sync"
	"time"
)

type ClientParam struct {
	//NClient         int
	ClientID        int
	Concurrency     int
	RequestRateMbps int
	//ClientType       string
	Mode            string
	RandomRoute     bool
	Trace           string
	ReplayStartTs   uint32
	ReplayEndTs     uint32
	UniqueObj       bool
	IgnoreRemoteReq bool
	WorkloadParam   int
	ReqSize         int
	//ReplayInRealTime bool
	ReplaySpeedup float64
	//DataDir          string
	RemoteOrigin bool

	//SupportFailure bool

	NServers int
	Hosts    []string

	UnavailTrace       string
	UnavailUpdateIntvl int
	//LB                 *loadbalancer.ChBalancer
}

type LatencyResult struct {
	Latency float64
	Req     string
}

type ClientLatencyChans struct {
	FirstByteRAM  chan LatencyResult
	FullRespRAM   chan LatencyResult
	FirstByteHit  chan LatencyResult
	FullRespHit   chan LatencyResult
	FirstByteMiss chan LatencyResult
	FullRespMiss  chan LatencyResult

	//FirstByteNoDecodeHit  chan LatencyResult
	//FullRespNoDecodeHit   chan LatencyResult
	//FirstByteNoDecodeMiss chan LatencyResult
	//FullRespNoDecodeMiss  chan LatencyResult
	//FirstByteDecodeHit    chan LatencyResult
	//FullRespDecodeHit     chan LatencyResult
	//FirstByteDecodeMiss   chan LatencyResult
	//FullRespDecodeMiss    chan LatencyResult

	//FirstByteFullObjHit  chan LatencyResult
	//FullRespFullObjHit   chan LatencyResult
	//FirstByteFullObjMiss chan LatencyResult
	//FullRespFullObjMiss  chan LatencyResult
}

//type ClientLat struct {
//
//}

type BucketFBData struct {
	IntervalLatencySum float64 // first byte
	IntervalMissCnt    int64
	IntervalReqCnt     int64
	IntervalMissBytes  int64
	IntervalReqBytes   int64
	Mtx                sync.Mutex
}

type BucketFRData struct {
	IntervalLatencySum float64 // full resp
	IntervalMissCnt    int64
	IntervalReqCnt     int64
	IntervalMissBytes  int64
	IntervalReqBytes   int64
	Mtx                sync.Mutex
}

type ClientStat struct {
	Workload    string
	ClientID    int
	Concurrency int
	HasCal      bool

	StartTs  time.Time
	FinishTs time.Time
	Runtime  time.Duration

	LatencyFirstByteSlice []float64
	LatencyFullRespSlice  []float64

	LatencyFirstByteMean   float64
	LatencyFirstByteMedian float64
	LatencyFirstByteMin    float64
	LatencyFirstByteMax    float64
	LatencyFirstByteP90    float64
	LatencyFirstByteP95    float64
	LatencyFirstByteP99    float64
	LatencyFirstByteP999   float64

	LatencyFullRespMean   float64
	LatencyFullRespMedian float64
	LatencyFullRespMin    float64
	LatencyFullRespMax    float64
	LatencyFullRespP90    float64
	LatencyFullRespP95    float64
	LatencyFullRespP99    float64
	LatencyFullRespP999   float64

	IssueThroughput    float64
	AchievedThroughput float64

	IssuedTrafficInByte   int64
	ReceivedTrafficInByte int64
}

type ClientErrStat struct {
	IncompleErr   int64
	ConnBrokenErr int64
}

func (r *ClientStat) CalStat(latencyFirstByteSlice, latencyFullRespSlice []float64) {
	if len(latencyFirstByteSlice) < 1000 {
		log.Fatal("not enough data points")
	}

	if latencyFirstByteSlice != nil {
		sort.Float64s(latencyFirstByteSlice)
		r.LatencyFirstByteMin, _ = stats.Min(latencyFirstByteSlice)
		r.LatencyFirstByteMax, _ = stats.Max(latencyFirstByteSlice)
		r.LatencyFirstByteMean, _ = stats.Mean(latencyFirstByteSlice)
		r.LatencyFirstByteMedian, _ = stats.Median(latencyFirstByteSlice)
		r.LatencyFirstByteP90 = latencyFirstByteSlice[int(float64(len(latencyFirstByteSlice))*0.9)]
		r.LatencyFirstByteP95 = latencyFirstByteSlice[int(float64(len(latencyFirstByteSlice))*0.95)]
		r.LatencyFirstByteP99 = latencyFirstByteSlice[int(float64(len(latencyFirstByteSlice))*0.99)]
		r.LatencyFirstByteP999 = latencyFirstByteSlice[int(float64(len(latencyFirstByteSlice))*0.999)]
	}

	if latencyFullRespSlice != nil {
		sort.Float64s(latencyFullRespSlice)
		r.LatencyFullRespMin, _ = stats.Min(latencyFullRespSlice)
		r.LatencyFullRespMax, _ = stats.Max(latencyFullRespSlice)
		r.LatencyFullRespMean, _ = stats.Mean(latencyFullRespSlice)
		r.LatencyFullRespMedian, _ = stats.Median(latencyFullRespSlice)
		r.LatencyFullRespP90 = latencyFullRespSlice[int(float64(len(latencyFullRespSlice))*0.9)]
		r.LatencyFullRespP95 = latencyFullRespSlice[int(float64(len(latencyFullRespSlice))*0.95)]
		r.LatencyFullRespP99 = latencyFullRespSlice[int(float64(len(latencyFullRespSlice))*0.99)]
		r.LatencyFullRespP999 = latencyFullRespSlice[int(float64(len(latencyFullRespSlice))*0.999)]
	}
	r.HasCal = true
}

func (r *ClientStat) String() string {
	if r.Runtime == 0 {
		r.Runtime = r.FinishTs.Sub(r.StartTs)
	}
	output1 := fmt.Sprintf("workload %s: client %d, %d concurrency finish using %.2f seconds\n, issueThrpt %d rps %f Gbps, achieved thrpt %f Gbps\n",
		r.Workload, r.ClientID, r.Concurrency, float64(r.Runtime.Nanoseconds())/1000000000.0, r.IssueThroughput, r.AchievedThroughput)

	output2 := fmt.Sprintf("\t\tfirstByteLatency mean %f, median %f, P90 %f, P99 %f, P999 %f\n",
		r.LatencyFirstByteMean, r.LatencyFirstByteMedian, r.LatencyFirstByteP90, r.LatencyFirstByteP99, r.LatencyFirstByteP999)

	output3 := fmt.Sprintf("\t\tFullRespLatency mean %f, median %f, P90 %f, P99 %f, P999 %f\n",
		r.LatencyFullRespMean, r.LatencyFullRespMedian, r.LatencyFullRespP90, r.LatencyFullRespP99, r.LatencyFullRespP999)

	return output1 + output2 + output3
}

type Request struct {
	Timestamp uint32
	ID        uint32
	Size      uint32
}

type RequestWithNodeID struct {
	Timestamp uint32
	ID        uint32
	Size      uint32
	NodeID    uint32
}

type RequestWithBucket struct {
	Timestamp uint32
	ID        uint32
	Size      uint32
	Bucket    uint32
}

type FullRequest struct {
	Timestamp        uint32
	ID               uint32
	Size             uint32
	Bucket           uint32
	OriginalHostIdx  int
	AlternateHostIdx int
	//HostIdxs        []uint32
	Remote bool
	//TrafficClass uint8
}
