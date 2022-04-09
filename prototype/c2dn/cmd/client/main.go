package main

import (
	"flag"
	"github.com/1a1a11a/c2dnPrototype/src/loadbalancer"
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	logger  *zap.Logger
	slogger *zap.SugaredLogger
)

var (
	cParam         ClientParam
	lastLogPrintTs int64
	replayFinished int32 = 0

	lb              *loadbalancer.ChBalancer
	lbWithNoFailure *loadbalancer.ChBalancer
)

var HttpClient *http.Client

var (
	traceTime     uint32
	clientStartTs time.Time
)

var (
	clientTraceMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "client_trace_n_read",
		Help: "The number of requests read from trace",
	}, []string{"reqType"})

	//clientSentMetric = promauto.NewCounterVec(prometheus.CounterOpts{
	//	Name: "client_sent_req",
	//	Help: "The number of requests sent since client started",
	//}, []string{"reqType", "bucket", "unit"})

	clientReqMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "client_finished_req",
		Help: "The number of requests finished since client started",
	}, []string{"reqType"})

	clientTrafficMetric = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "client_finished_req_byte",
		Help: "The number of requests finished in byte since client started",
	}, []string{"reqType"})

	clientStateMetric = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "client_state",
		Help: "State of client",
	}, []string{"name"})
)

func init() {
	myutils.CreateDir(myconst.OutputDir)
	myutils.RunTimeInit()
	logger, slogger = myutils.InitLogger("client", myconst.DebugLevel)

	go func() {
		slogger.Infof("metrics collection goroutine starts at port %v", myconst.ClientMetricsPort)
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":"+myconst.ClientMetricsPort, nil)
	}()

	//bucketFBDataMap = make(map[int]*BucketFBData)
	//bucketFRDataMap = make(map[int]*BucketFRData)

	tr := &http.Transport{
		MaxIdleConns:        myconst.MaxIdleConns,
		MaxIdleConnsPerHost: myconst.MaxIdleConnsPerHost,
		IdleConnTimeout:     myconst.HttpTimeOut * time.Second,
		//DisableCompression:  true,
	}
	HttpClient = &http.Client{Transport: tr, Timeout: myconst.HttpTimeOut * time.Second}

	//for i := 0; i < myconst.NBuckets; i++ {
	//	bucketFBDataMap[i] = &BucketFBData{}
	//	bucketFRDataMap[i] = &BucketFRData{}
	//}
}

func main() {

	var replayStartTs, replayEndTs uint64
	slogger.Info(os.Args)

	flag.IntVar(&cParam.NServers, "nServers", 10, "the number of CDN servers")
	flag.StringVar(&cParam.Mode, "mode", "replayRealtime", "client mode")
	flag.BoolVar(&cParam.RandomRoute, "randomRoute", true, "whether a request can be sent to two servers")
	flag.StringVar(&cParam.Trace, "trace", "", "trace path")
	flag.Uint64Var(&replayStartTs, "replayStartTs", 0, "the relative start time to replay the trace")
	flag.Uint64Var(&replayEndTs, "replayEndTs", 0, "the relative end time to replay the trace")
	flag.BoolVar(&cParam.UniqueObj, "uniqueObj", false, "whether only replay unique objects")
	flag.BoolVar(&cParam.IgnoreRemoteReq, "ignoreRemoteReq", true,
		"whether ignore requests for remote origin, this should be true running evaluation, but false when warmup")
	flag.IntVar(&cParam.Concurrency, "concurrency", 1,
		"the number of requesters")
	flag.IntVar(&cParam.RequestRateMbps, "requestRateGbps", 0, "the request rate in Gbps of client (open loop)")
	flag.IntVar(&cParam.ClientID, "clientID", -100, "the id of client")
	//flag.IntVar(&cParam.NClient, "nClient", 1, "the number of clients")

	flag.IntVar(&cParam.ReqSize, "reqSize", myconst.SynWorkloadReqSize,
		"the default req size to use for synthetic workloads")
	flag.IntVar(&cParam.WorkloadParam, "workloadParam", 1,
		"only used for all miss type workload")

	flag.BoolVar(&cParam.RemoteOrigin, "remoteOrigin", false,
		"whether client and origin will use WAN, this option allows the client to send the traffic to a remote origin")
	//flag.BoolVar(&cParam.SupportFailure, "supportFailure", false,
	//	"whether the client will send the traffic to the other node if the chosen node fails")
	flag.Float64Var(&cParam.ReplaySpeedup, "replaySpeedup", 1, "1 means no speed up")
	flag.StringVar(&cParam.UnavailTrace, "unavailTrace", "", "the path to the unavailable node data")
	flag.IntVar(&cParam.UnavailUpdateIntvl, "updateInterval", 300, "the cluster availability update interval")

	flag.Parse()
	cParam.Hosts = flag.Args()

	cParam.ReplayStartTs, cParam.ReplayEndTs = uint32(replayStartTs), uint32(replayEndTs)

	//var hostIdx = []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	//if CParam.UnavailTrace != "" {
	//	lb = loadbalancer.NewConsistentHashBalancer(hostIdx, CParam.UnavailTrace, int64(CParam.UnavailUpdateIntvl))
	//} else {
	//	lb = loadbalancer.NewConsistentHashBalancer(hostIdx, "", -1)
	//}

	if len(cParam.Hosts) != cParam.NServers {
		log.Fatal("specified n serveres ", cParam.NServers, ", but only find ", len(cParam.Hosts), " servers: ", cParam.Hosts)
	}

	var hostIdx []string
	for i := 0; i < cParam.NServers; i++ {
		hostIdx = append(hostIdx, strconv.Itoa(i))
	}

	if cParam.UnavailTrace != "" {
		lb = loadbalancer.NewConsistentHashBalancer(cParam.NServers, cParam.UnavailTrace, int64(cParam.UnavailUpdateIntvl), 2)
	} else {
		lb = loadbalancer.NewConsistentHashBalancer(cParam.NServers, "", -1, 2)
	}
	lbWithNoFailure = loadbalancer.NewConsistentHashBalancer(cParam.NServers, "", -1, 2)

	if cParam.Mode == "replayCloseloop" || cParam.Mode == "replayOpenloop" {
		RunAkamai()
	} else {
		log.Fatal("unknown mode ", cParam.Mode)
	}

}

//export GO111MODULE=on
//go mod init
//go mod vendor # if you have vendor/ folder, will automatically integrate
//go build
