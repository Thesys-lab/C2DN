package main

import (
	"io/ioutil"
	"net/http"
	"time"
)

func RequestServer(url string) (latency int64){
	latency = -1
	startTs := time.Now()

	if resp, err := http.Get(url); err != nil {
		//log.Println(err);
		return -1
	} else {
		defer resp.Body.Close()
		if _, err := ioutil.ReadAll(resp.Body); err != nil {
			//log.Println(err);
			return -1
		} else{
			latency = int64(time.Since(startTs))
			//fmt.Println(strconv.FormatInt(latency, 10) + " us: " + string(body))
		}
	}
	return latency
}

func RequestServerOnce(url string, latenciesChan chan<- int64){
	startTs := time.Now()

	if resp, err := http.Get(url); err != nil {
		//log.Println(err);
		latenciesChan<- -1
	} else {
		defer resp.Body.Close()
		if _, err := ioutil.ReadAll(resp.Body); err != nil {
			//log.Println(err);
			latenciesChan<- -1
		} else{
			latency := int64(time.Since(startTs))
			latenciesChan<-latency
			//close(latenciesChan)
		}
	}
}

func RequestServerNTimes(url string, latenciesChan chan<- []int64, n int){
	latencies := make([]int64, 0, n)

	for i:=0; i< n; i++{
		latencies = append(latencies, RequestServer(url))
	}
	latenciesChan<-latencies
	//close(latenciesChan)
}



func RequestServerNTimesWithThrpt(url string, thrpt float64, n int64, latenciesChan chan<- []int64){
	latencies := make([]int64, 0, n)
	tPerQuery := int64(1.0e9 / thrpt)
	startTs := time.Now()

	for i:=int64(0); i< n; i++{
		if time.Since(startTs).Nanoseconds() < i * tPerQuery{
			time.Sleep(time.Duration(i*tPerQuery - time.Since(startTs).Nanoseconds() ))
		}
		latencies = append(latencies, RequestServer(url))
	}
	latenciesChan<-latencies
	//close(latenciesChan)
}

func RequestServerTime(url string, nSec int64, latenciesChan chan<- []int64){
	latencies := make([]int64, 0, nSec*20000)
	startTs := time.Now()

	for ;time.Since(startTs).Nanoseconds()<nSec*1e9; {
		latencies = append(latencies, RequestServer(url))
	}
	latenciesChan<-latencies
}