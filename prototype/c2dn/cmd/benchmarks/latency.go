package main

import (
	"fmt"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
	"time"
)

func MeasureLatency(throughput float64, nTest int64, url string) (latencies []int64, numFail int64) {
	numFail = 0
	tPerQuery := int64(1.0e9 / float64(throughput))

	startTs := time.Now()
	latencyChan := make(chan int64, 20000)
	for i := int64(0); i < nTest; i++ {
		if time.Since(startTs).Nanoseconds() < i * tPerQuery{
			time.Sleep(time.Duration(i*tPerQuery - time.Since(startTs).Nanoseconds() ))
		}
		go RequestServerOnce(url, latencyChan)
	}

	var latency int64;
	for i := int64(0); i < nTest; i++ {
		latency = <-latencyChan
		if latency < 0 {
			numFail += 1
		} else {
			latencies = append(latencies, latency)
		}
	}
	return latencies, numFail
}


func MeasureLatency2(throughput float64, concurrency, nTest int64, url string) (latencies []int64, numFail int64) {
	numFail = 0

	latencyChan := make(chan []int64, 20000)

	for i := int64(0); i < concurrency; i++ {
		go RequestServerNTimesWithThrpt(url, throughput/float64(concurrency), nTest/concurrency, latencyChan)
		time.Sleep(time.Duration(1e9/throughput))
	}

	for i := int64(0); i < concurrency; i++ {
		latency := <-latencyChan
		latencies = append(latencies, latency...)
	}

	for _, latency := range latencies{
		if latency < 0{
			numFail ++
		}
	}
	return latencies, numFail
}


func FindMaxThrpt(){
	const N_TEST_TIME = 20
	throughputs := []float64{6e3, 1e4, 2e4, 3e4, 4e4, 5e4, 6e4, 8e4}
	concurrencies := []int64{36, 48, 60, 72, 96, 120, 160, 240, 360, 480}
	//concurrencies := []int64{4, 8, 12, 16, 20, 24, 30, 40}

	conf := myutils.LoadConfig("conf.json")
	url := "http://" + conf.SingleCache + "/cache"

	for _, throughput := range throughputs {
		for _, concurrency := range concurrencies{
			startTs := time.Now()
			latencies, numFail := MeasureLatency2(throughput, concurrency, N_TEST_TIME*int64(throughput), url)
			//fmt.Println(latencies)
			elapsedTime := time.Since(startTs).Seconds()
			realThrpt := (float64(N_TEST_TIME)*float64(throughput) - float64(numFail)) / elapsedTime

			fmt.Printf("issue thrpt %6.0f, achieved thrpt %8.2f, concurrency %4d, latency avg %8.2f ms, min %2d ms, max %4d ms, failures %6d/%-12.0f %.4f\n",
				throughput, realThrpt, concurrency,
				myutils.AvgIntSlice(latencies)/1e6,
				myutils.MinIntSlice(latencies)/1e6,
				myutils.MaxIntSlice(latencies)/1e6, numFail, N_TEST_TIME*throughput,
				float64(numFail)/float64(N_TEST_TIME*throughput))
		}
	}
}


func ThrptConcurrency(concurrencies []int64) ([][]int64, []int64){
	conf := myutils.LoadConfig("conf.json")
	//url := "http://" + conf.SingleCache + "/cache"
	url := "http://" + conf.SingleOrigin + "/origin"
	fmt.Println(url)
	latencies2D := make([][]int64, len(concurrencies))
	numFail := make([]int64, len(concurrencies))
	latencySliceChan := make(chan []int64, 20000)

	for _, concurrency := range concurrencies{
		//startTs := time.Now()
		for i := int64(0); i < concurrency; i++ {
			go RequestServerTime(url, conf.TestTime, latencySliceChan)
		}

		latencySlice := make([]int64, len(concurrencies))
		for i := int64(0); i < concurrency; i++ {
			latencySliceTemp := <-latencySliceChan
			latencySlice = append(latencySlice, latencySliceTemp...)
		}
		latencies2D = append(latencies2D, latencySlice)

		numFailTemp := int64(0)
		for _, latency := range latencySlice{
			if latency < 0{
				numFailTemp ++
			}
		}
		numFail = append(numFail, numFailTemp)

		//elapsedTime := time.Since(startTs).Seconds()
		realThrpt := (float64(len(latencySlice))-float64(numFailTemp)) / float64(conf.TestTime)

		fmt.Printf("Concurrency %6.0d, achieved thrpt %8.2f, latency avg %8.2f ms, min %2d ms, max %4d ms, 99P %4d ms, 99.9P %4d ms, failures %6d/%-12.0d %.4f\n",
			concurrency, realThrpt,
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
