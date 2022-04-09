package main

import (
	"flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"
	"os"
	"time"

	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
)

var (
	logger  *zap.Logger
	slogger *zap.SugaredLogger
)

func init() {
	myutils.CreateDir(myconst.OutputDir)
	myutils.RunTimeInit()
	logger, slogger = myutils.InitLogger("origin", myconst.DebugLevel)

	go func() {
		slogger.Infof("metrics collection goroutine starts at port %v", myconst.OriginMetricsPort)
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":"+myconst.OriginMetricsPort, nil)
	}()
}

func main() {
	slogger.Info(os.Args)

	var sleepTimeMs int

	// origin parameters
	flag.IntVar(&sleepTimeMs, "sleepTimeMs", 0, "how long in millisecond the origin should sleep")
	flag.BoolVar(&OParam.SetNoCacheHeader, "noCache", false, "whether origin should add no-cache header")
	flag.Parse()

	OParam.SleepTimeMs = time.Duration(sleepTimeMs) * time.Millisecond

	Run()
}
