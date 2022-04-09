package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"
)


//func main() {
//	//testPushHttp(0, nil, nil)
//	testConcurrent()
//}

func testGetHttp() {
	client := http.Client{}
	tmp := make([]byte, 1280*1024*10240)
	//tmp := make([][]byte, 8)

	ts1 := time.Now()
	resp, err := client.Get("http://127.0.0.1:2048/akamai/ab_1200000000")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(float64(time.Since(ts1).Nanoseconds())/1000000, " ms")
	reader := bufio.NewReader(resp.Body)

	var n = 1
	for n != 0 {
		n, _ = reader.Read(tmp)
		log.Printf("read %d bytes, %v ms\n", n, float64(time.Since(ts1).Nanoseconds())/1000000)
	}

	time.Sleep(time.Duration(time.Second * 20))
	resp.Body.Close()
}

func testPushTcp() {
	host := "asrock.jasony.me:8080"
	//host := "127.0.0.1:8080"
	conn, err := net.Dial("tcp", host)
	if err != nil {
		log.Fatal("dial error:", err)
		return
	}
	var n int
	tmp := make([]byte, 128*1024)
	//s2 := "PUSH http://127.0.0.1:8080/push HTTP/1.1\r\nContent-Length: 51\r\n\r\n"+
	//  "HTTP/1.1 200 OK\r\nobjType:full\r\nobjSize:24\r\n\r\n!!!!!!"
	s2 := fmt.Sprintf("PUSH http://%v/push HTTP/1.1\r\nContent-Length: 51\r\n\r\n"+
		"HTTP/1.1 200 OK\r\nobjType:full\r\nobjSize:24\r\n\r\n!!!!!!", host)

	//fmt.Println(s2)
	n, err = fmt.Fprintf(conn, s2)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("writes ", n, " bytes")

	if n, err := conn.Read(tmp); err != nil {
		if err != io.EOF {
			log.Fatal(err)
		} else {
			log.Fatal(err)
		}
	} else {
		log.Println("read ", n, " bytes ", string(tmp[:n]))
	}

	log.Fatal("done")
}

func testPushHttp(i int, wg *sync.WaitGroup, client *http.Client) {
	n := 20000
	if wg != nil {
		defer wg.Done()
	}
	if client == nil {
		client = &http.Client{}
	}

	startTs := time.Now()
	//host := "asrock.jasony.me:8080"
	host := "174.129.155.19:8080"
	//host := "127.0.0.1:8080"

	s := RandString(8)
	content := []byte("HTTP/1.1 200 OK\r\nobjType:full\r\nobjSize:24\r\n\r\n")
	content2 := bytes.Repeat([]byte("!!!!!!!!!!!!!!!!!!!!"), n)
	reader := io.MultiReader(bytes.NewReader(content), bytes.NewReader(content2))
	//req, err := http.NewRequest("PUSH", "http://"+host+"/push2", bytes.NewReader(content))
	req, err := http.NewRequest("PUSH", "http://"+host+"/"+s, reader)
	req.ContentLength = int64(45 + 20*n)
	if err != nil {
		log.Fatal("err creating requests ", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("req %v: err pushing %v", i, err)
	}
	fmt.Printf("req %v %v: size %v KB, status %v, %v\n", i, string(s), 20*n/1000, resp.Status, time.Since(startTs))
	_, _ = io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()
	//fmt.Println(resp.Header)
	//tmp := make([]byte, 8*1024)
	//n, err := resp.Body.Read(tmp)
	//fmt.Println(tmp[:n])
}

func testConcurrent() {
	rand.Seed(rand.Int63())
	wg := &sync.WaitGroup{}

	//tr := &http.Transport{
	//	MaxIdleConns:        600,
	//	MaxIdleConnsPerHost: 600,
	//	IdleConnTimeout:     30 * time.Second,
	//}
	//client := &http.Client{Transport:tr}
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go testPushHttp(i, wg, nil)
		//time.Sleep(200*time.Millisecond)
	}
	wg.Wait()
}

//"HTTP/1.0 200 OK${CRLF}Content-type: ${f_type}${CRLF}Content-length: ${len_content}${CRLF}${CRLF}${f}${CRLF}";
