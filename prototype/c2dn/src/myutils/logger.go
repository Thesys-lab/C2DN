package myutils

import (
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/log"
	"go.uber.org/zap"
)

var (
	Logger      *zap.Logger
	SugarLogger *zap.SugaredLogger
)

func init() {
	CreateDir(myconst.LogDir)
	Logger, SugarLogger = InitLogger("misc", myconst.DebugLevel)
}

func InitLogger(name string, debugMode int) (logger *zap.Logger, sugarlogger *zap.SugaredLogger) {
	var cfg zap.Config
	if debugMode >= myconst.PerReqLogging {
		cfg = zap.NewProductionConfig()
		cfg = zap.NewDevelopmentConfig()
		//cfg.Level.SetLevel(zap.DebugLevel)
	} else {
		cfg = zap.NewProductionConfig()
		cfg.Level.SetLevel(zap.InfoLevel)
	}
	//cfg.OutputPaths = []string{"stdout", "/tmp/log" + name}
	//cfg.ErrorOutputPaths = []string{"stderr", "/tmp/logerr" + name}

	cfg.OutputPaths = []string{"stdout", "/tmp/c2dn/log/" + name}
	cfg.ErrorOutputPaths = []string{"stderr", "/tmp/c2dn/log/" + name + ".err"}

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}
	sugarlogger = logger.Sugar()

	defer logger.Sync() // flushes buffer, if any
	return logger, sugarlogger
}

func GetCounterValue(metric *prometheus.CounterVec, label []string) float64 {
	var m = &dto.Metric{}
	if err := metric.WithLabelValues(label...).Write(m); err != nil {
		log.Error(err)
		return 0
	}
	return m.Counter.GetValue()
}

func GetGaugeValue(metric *prometheus.GaugeVec, label []string) float64 {
	var m = &dto.Metric{}
	if err := metric.WithLabelValues(label...).Write(m); err != nil {
		log.Error(err)
		return 0
	}
	return m.Gauge.GetValue()
}
