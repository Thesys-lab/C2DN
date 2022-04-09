package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func test1Streaming() {
	respByte := make([]byte, 1024*1024*256)

	resp, _ := http.Get("http://d30.jasony.me:2048/akamai/hi_1000000000")
	//resp, _ := http.Get("http://d30.jasony.me:2048")
	//s, _ := ioutil.ReadAll(resp.Body)
	//fmt.Println(string(s))

	reader := bufio.NewReader(resp.Body)
	for {
		//line, err := reader.ReadBytes('\n')
		n, err := reader.Read(respByte)
		if err == io.EOF {
			fmt.Println("read all")
			break
		} else {
			fmt.Println(n, string(respByte[:12]))
		}
	}
}

func TestPersisent() {
	req, err := http.NewRequest(http.MethodGet, "http://d30.jasony.me:2048", nil)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	client := &http.Client{}

	for i := 0; i < 240; i++ {
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
		_, _ = readBody(resp.Body)
		fmt.Println("done ", i)
		time.Sleep(200 * time.Millisecond)
	}
}

func readBody(readCloser io.ReadCloser) ([]byte, error) {
	defer readCloser.Close()
	body, err := ioutil.ReadAll(readCloser)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func TestPersisentWithStreaming() {
	respByte := make([]byte, 1024*1024*256)
	var totalBytes int64
	var latFirstByte, latFullResp float64

	req, err := http.NewRequest(http.MethodGet, "http://d30.jasony.me:2048/size/102400000", nil)
	//req, err := http.NewRequest(http.MethodGet, "http://d30.jasony.me:2048/", nil)
	req.Header.Add("Host", "d30.jasony.me:2048")
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	client := &http.Client{}

	for i := 0; i < 240; i++ {
		startTs := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println(err.Error())
			return
		}

		totalBytes = 0
		latFirstByte, latFullResp = 0, 0
		reader := bufio.NewReader(resp.Body)
		for {
			n, err := reader.Read(respByte)
			if latFirstByte == 0 {
				latFirstByte = float64(int64(time.Since(startTs).Nanoseconds())) / 1000000.0
			}
			totalBytes += int64(n)
			//fmt.Println(n, string(respByte[:12]))

			if err == io.EOF {
				latFullResp = float64(int64(time.Since(startTs).Nanoseconds())) / 1000000.0
				fmt.Println("total ", totalBytes, " Bytes", "latency ", latFirstByte, " / ", latFullResp)
				break
			}
		}
	}
}

func testHttpClient() {
	client := http.Client{}
	tmp := make([]byte, 1280*1024)

	ts1 := time.Now()
	resp, err := client.Get("http://127.0.0.1:2048/akamai/ab_1200000000")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(float64(time.Since(ts1).Nanoseconds())/1000000, " ms")

	var n = 1
	for n != 0 {
		n, _ = resp.Body.Read(tmp)
		log.Printf("read %d bytes, %v ms\n", n, float64(time.Since(ts1).Nanoseconds())/1000000)
	}

	time.Sleep(time.Duration(time.Second * 20))
	resp.Body.Close()
}
