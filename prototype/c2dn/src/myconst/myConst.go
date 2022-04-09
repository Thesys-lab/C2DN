package myconst

import "time"

const (
	PerIntvlLogging        = 1
	PerReqLogging          = 3
	PerReqDetailedLogging  = 4
	PerReqDetailedLogging2 = 5
)

const (
	DebugLevel      = PerReqLogging
	DumpStat        = true
	UseLocalATSAsFE = false
	UseOriginAsATS  = false
	LocalDebug      = false
	UseFakePush     = true
)

const (
	ATSPort           = "8080"
	FEPort            = "8081"
	OriginPort        = "2048"
	OriginMetricsPort = "2020"
	ClientMetricsPort = "2021"
	FEMetricsPort     = "2022"
)

const (
	ChanBufLen         = 2000
	NumClients         = 10
	SynWorkloadReqSize = 128*1024 - 1
	AllHitWorksetSize  = 2000
	StressTestRunTime  = 20
)

const (
	OutputDir = "/tmp/c2dn/output/"
	LogDir    = "/tmp/c2dn/log/"
)

const (
	OriginStreamingChunkSize = 1024 * 1024
	RAMCacheAdmitThreshold   = int64(16 * 1024 * 1024)
	CodingObjSizeThreshold   = int64(128)
	//CodingObjSizeThreshold   = int64(128 * 1024)
	//CodingObjSizeThreshold   = int64(1024 * 1024)
	CodingSubChunkSize = int64(128 * 1024)

	ParityChunkFetchDelayMs = 5 * time.Millisecond
	NBuckets                = 200
)

const (
	HttpTimeOut         = 300
	MaxIdleConns        = 3000
	MaxIdleConnsPerHost = 300
)

const (
//MAXECN = 5
)

const (
	ClientReportInterval = 20
)

const ()

const (
	KB = 1000
	MB = 1000 * 1000
	GB = 1000 * 1000 * 1000
	TB = 1000 * 1000 * 1000 * 1000
)

const (
	KiB = 1024
	MiB = 1024 * 1024
	GiB = 1024 * 1024 * 1024
	TiB = 1024 * 1024 * 1024 * 1024
)

const (
	Sec      = 1000 * 1000 * 1000
	MilliSec = 1000 * 1000
	MicroSec = 1000
	NanoSec  = 1
)

const (
	RamHit = iota // this is enum
	Hit
	Miss
	//NoDecodeHit
	//NoDecodeMiss
	DecodeHit
	DecodeMiss
	FullObjHit
	FullObjMiss
)
