package main

import (
	"flag"
	"fmt"
	"github.com/1a1a11a/c2dnPrototype/src/loadbalancer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"
	"os"
	"strings"

	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
)

var (
	logger  *zap.Logger
	slogger *zap.SugaredLogger
)

var (
	metricByteClient = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "frontend_client_byte",
		Help: "The number of bytes served to the client",
	}, []string{"reqType"})

	metricReqClient = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "frontend_client_nReq",
		Help: "The number of requests served to the client",
	}, []string{"reqType"})

	metricReq = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "frontend_nReq",
		Help: "The number of requests",
	}, []string{"reqType"})

	metricTraffic = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "frontend_cdn_traffic",
		Help: "The cluster traffic in byte",
	}, []string{"trafficType"})

	metricErr = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "frontend_cdn_stat_err",
		Help: "Stat about err",
	}, []string{"errType"})

	metricEC = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "frontend_cdn_n_chunk",
		Help: "The number of chunks found/used for erasure-coded objects",
	}, []string{"nChunk"})
)

func init() {
	myutils.CreateDir(myconst.OutputDir)
	myutils.RunTimeInit()
	logger, slogger = myutils.InitLogger("frontend", myconst.DebugLevel)

	go func() {
		slogger.Infof("metrics export goroutine starts at port %v", myconst.FEMetricsPort)
		http.Handle("/metrics", promhttp.Handler())
		if err := http.ListenAndServe(":"+myconst.FEMetricsPort, nil); err != nil {
			slogger.DPanic("Cannot start frontend metric export service", err)
		}
	}()
}

func parseParams() {
	flag.StringVar(&feParam.Mode, "mode", "twoRepAlways", "the mode of frontend")
	flag.IntVar(&feParam.NodeIdx, "nodeIdx", -100, "the index of this server in the cluster")
	flag.IntVar(&feParam.NServers, "nServers", 10, "the number of CDN servers")
	flag.IntVar(&feParam.ECN, "EC_n", 4, "the number of total chunks in erasure coding")
	flag.IntVar(&feParam.ECK, "EC_k", 3, "the number of data chunks in erasure coding")
	flag.IntVar(&feParam.RepFactor, "repFactor", 2, "replication factor for small objects in C2DN or all objects in CDN")
	flag.Int64Var(&feParam.RamCacheSize, "ramCacheSize", 16*myconst.GiB, "the RAM cache size of frontend")
	flag.StringVar(&feParam.UnavailTrace, "unavailTrace", "", "the path to the unavailable node data")
	flag.IntVar(&feParam.UnavailUpdateIntvl, "updateInterval", 300, "the cluster availability update interval")

	flag.Parse()
	feParam.Hosts = flag.Args()
}

func checkParams() {
	if len(feParam.Hosts) != feParam.NServers {
		s := fmt.Sprintf("specified %v, servers, but only find %v server %v",
			feParam.NServers, len(feParam.Hosts), feParam.Hosts)
		panic(s)
	}

	if feParam.NodeIdx == -100 {
		slogger.Panic("frontend node idx is -1")
	}

	if feParam.Mode == "twoRepAlways" {
		if feParam.RepFactor != 2 || feParam.ECN != 2 || feParam.ECK != 1 {
			slogger.Panic("twoRepAlways parameters not correct")
		}
	} else if feParam.Mode == "C2DN" {
		if feParam.RepFactor != 2 {
			slogger.Panic("C2DN replication factor not 2")
		}
	} else if feParam.Mode == "noRep" {
		if feParam.RepFactor != 1 || feParam.ECN != 1 || feParam.ECK != 1 {
			slogger.Panic("noRep parameters not correct")
		}
	} else if feParam.Mode == "naiveCoding" {
		if feParam.RepFactor != 2 {
			slogger.Panic("naive coding parameter not correct")
		}
	} else {
		slogger.Panicf("unknown mode %v", feParam)
	}

	ipPortSplit := strings.Split(feParam.Hosts[feParam.NodeIdx], ":")
	if !myconst.LocalDebug && ipPortSplit[0] != feIP && ipPortSplit[0] != "127.0.0.1" {
		slogger.Panicf("frontend ip %v does not match hosts[%d] %v", feIP, feParam.NodeIdx, feParam.Hosts[feParam.NodeIdx])
	}
	if !myconst.LocalDebug && ipPortSplit[1] != myconst.ATSPort {
		slogger.Panicf("frontend hosts port %v does not match ats port %v", ipPortSplit[1], myconst.ATSPort)
	}

	slogger.Infof("server %d, mode %v, feIP %v, ec n %d k %d ram %.2f GiB, unavailTrace %v",
		feParam.NodeIdx, feParam.Mode, feIP, feParam.ECN, feParam.ECK,
		float64(feParam.RamCacheSize)/myconst.GiB, feParam.UnavailTrace)
}

func main() {
	slogger.Info(os.Args)

	parseParams()
	checkParams()

	rebalance := false
	if feParam.Mode == "C2DN" {
		rebalance = true
	}

	if feParam.UnavailTrace != "" {
		lb = loadbalancer.NewParityBalancer(feParam.NServers, feParam.UnavailTrace, int64(feParam.UnavailUpdateIntvl), feParam.ECN, rebalance)
	} else {
		lb = loadbalancer.NewParityBalancer(feParam.NServers, "", -1, feParam.ECN, rebalance)
	}

	Run()
}
