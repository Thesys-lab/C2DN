package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

func testSlice() {

	s := make([]int, 18)
	fmt.Println(len(s), cap(s))
	for i := 0; i < 24; i++ {
		s = append(s, i)
		fmt.Println(len(s), cap(s))
	}
}


func createHTTPClient(MaxIdleConnections, RequestTimeout int) *http.Client {
	client := &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: MaxIdleConnections,
		},
		Timeout: time.Duration(RequestTimeout) * time.Second,
	}

	return client
}


func Test2() {
	var httpClient = createHTTPClient(200, 2400)

	endPoint := "https://baidu.com"

	req, err := http.NewRequest("GET", endPoint, bytes.NewBuffer([]byte("Post this data")))
	if err != nil {
		log.Fatalf("Error Occured. %+v", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	response, err := httpClient.Do(req)
	if err != nil && response == nil {
		log.Fatalf("Error sending request to API endpoint. %+v", err)
	}
	defer response.Body.Close()

	// Let's check if the work actually is done
	// We have seen inconsistencies even when we get 200 OK response
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Couldn't parse response body. %+v", err)
	}

	log.Println("Response Body:", string(body))
}
