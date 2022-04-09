package myutils

import (
	"fmt"
	"go.uber.org/zap"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"time"
)

const (
	DEFAULT_SYNC_TIME_MOD = 20
	DEFAULT_MIN_WAIT_TIME = 8
)

func PrintDir(path string) {
	files, err := ioutil.ReadDir(".")
	if err != nil {
		log.Fatal(err)
	}

	for _, file := range files {
		fmt.Println(file.Name())
	}
}

func RunTimeInit() {
	numcpu := runtime.NumCPU()
	runtime.GOMAXPROCS(numcpu)
}

func CatchPanic(logger *zap.SugaredLogger) {
	if r := recover(); r != nil {
		//debug.PrintStack()
		logger.DPanicf("Caught panic due to %v", r)
	}
}

func CatchPanicWithMsg(logger *zap.SugaredLogger, msg string) {
	if r := recover(); r != nil {
		//debug.PrintStack()
		logger.DPanicf("Caught panic due to %v, %v", r, msg)
	}
}

func TimeSync(syncMod int, minWaitTime int) {
	if syncMod <= 0 {
		syncMod = DEFAULT_SYNC_TIME_MOD
	}
	if minWaitTime <= 0 {
		minWaitTime = DEFAULT_MIN_WAIT_TIME
	}

	now := time.Now().Second()
	start := now
	SugarLogger.Infof("Sync time")
	for i := 0; now%syncMod != 0 && now-start < minWaitTime; i++ {
		if i%20 == 0 {
			fmt.Print(".")
		}
		time.Sleep(time.Millisecond * 20)
		now = time.Now().Second()
	}
	fmt.Println("")
}

func CreateDir(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0775); err != nil {
			panic(err)
		}
	}
}
