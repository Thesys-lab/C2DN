package main

import (
	"fmt"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
	"strconv"
)

func LatencySize(sizes []int64) ([][]int64, []int64) {

	conf := myutils.LoadConfig("conf.json")

	latencies2D := make([][]int64, len(sizes))
	numFail := make([]int64, len(sizes))
	latencySliceChan := make(chan []int64, 20000)

	for _, objSize := range sizes {
		url := "http://" + conf.TestServer + "/size/" + strconv.Itoa(int(objSize))
		for i := int64(0); i < conf.ClientConcurrency; i++ {
			go RequestServerTime(url, conf.TestTime, latencySliceChan)
		}

		latencySlice := make([]int64, conf.ClientConcurrency)
		for i := int64(0); i < conf.ClientConcurrency; i++ {
			latencySliceTemp := <-latencySliceChan
			latencySlice = append(latencySlice, latencySliceTemp...)
		}
		latencies2D = append(latencies2D, latencySlice)

		numFailTemp := int64(0)
		for _, latency := range latencySlice {
			if latency < 0 {
				numFailTemp++
			}
		}
		numFail = append(numFail, numFailTemp)

		realThrpt := (float64(len(latencySlice)) - float64(numFailTemp)) / float64(conf.TestTime)

		fmt.Printf("ObjSize %16.0d bytes, achieved thrpt %8.2f, latency avg %8.2f ms, min %2d ms, max %4d ms, P99 %4d ms, P99.9 %4d ms, failures %6d/%-12.0d %.4f\n",
			objSize, realThrpt,
			myutils.AvgIntSlice(latencySlice)/1e6,
			myutils.MinIntSlice(latencySlice)/1e6,
			myutils.MaxIntSlice(latencySlice)/1e6,
			myutils.PercentileIntSlice(latencySlice, 99.0)/1e6,
			myutils.PercentileIntSlice(latencySlice, 99.9)/1e6,
			numFailTemp, len(latencySlice),
			float64(numFailTemp)/float64(len(latencySlice)))
	}
	return latencies2D, numFail
}
