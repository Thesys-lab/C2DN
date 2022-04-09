package main

import (
	"bytes"
	"fmt"
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"
)

//type PushATSData struct {
//	hostID  int
//	url     string
//	content []byte
//}

/* if ChunkSize > myconst.CodingSubChunkSize,
	first figure out how many full blocks (myconst.CodingSubChunkSize) will be in each chunk,
	then calculate the size of last block

	then chunk1 will be dat[0:myconst.CodingSubChunkSize] + dat[feParam.ECK*myconst.CodingSubChunkSize:(feParam.ECK+1)*myconst.CodingSubChunkSize] +


 two possible approaches for streaming
	1. single-threaded chunk encoding, then send to ATS in sequence
	2. copy content in the order needed by streaming, then encode and send in parallel
	3. let ec handle all this
we take one here for less memcpy
*/

const (
	nPushRetry = 5
)

func pushFullObj(dat []byte, feTxn *FrontendTxnData) {
	if int64(len(dat)) != feTxn.ObjSize {
		slogger.DPanicf("txn %v: req %v, passed in obj size %v, required objSize %v",
			feTxn.TxnID, feTxn.ReqContent, int64(len(dat)), feTxn.ObjSize)
	}

	for i := 0; i < feParam.RepFactor; i++ {
		if feTxn.PeerIdx[i] != feTxn.FirstReqHostIdx {
			metricReq.WithLabelValues("pushFullObj").Inc()
			metricTraffic.WithLabelValues("pushFullObj").Add(float64(feTxn.ObjSize))
			metricReq.WithLabelValues("intra").Inc()
			metricTraffic.WithLabelValues("intra").Add(float64(feTxn.ObjSize))
		}
		if myconst.UseFakePush {
			fakePush(feTxn.PeerIdx[i], -1, "full", feTxn, nil, 0)
		} else {
			pushToATS(feTxn.PeerIdx[i], dat, "full", feTxn, nil, nPushRetry)
		}
	}

	appendDebugString(feTxn, "pushFullFinished")
}

func pushChunks(dat []byte, feTxn *FrontendTxnData, chunkIdxs []int) {
	myutils.CatchPanic(slogger)
	var err error
	if int64(len(dat)) != feTxn.ObjSize {
		slogger.DPanicf("txn %v: req %v, passed in obj size %v, required objSize %v",
			feTxn.TxnID, feTxn.ReqContent, int64(len(dat)), feTxn.ObjSize)
	}

	feTxn.ChunkSize = feTxn.ObjSize / int64(feParam.ECK)
	if feTxn.ObjSize%int64(feParam.ECK) != 0 {
		feTxn.ChunkSize += 1
	}

	splitData, err := myEncoder.Split(dat)
	if err != nil {
		slogger.DPanicf("err splitting data %v %v", err, feTxn)
	}
	if int64(len(splitData[0])) != feTxn.ChunkSize {
		slogger.DPanicf("splitted chunk size differ from chunk size %v != %v, %v",
			len(splitData[0]), feTxn.ChunkSize, feTxn)
	}

	if err = myEncoder.Encode(splitData); err != nil {
		slogger.DPanicf("err encoding data %v %v", err, feTxn)
	}

	wg := &sync.WaitGroup{}

	appendDebugString(feTxn, fmt.Sprintf("pushChunk-%v", chunkIdxs))
	for _, chunkIdx := range chunkIdxs {
		hostIdx := feTxn.PeerIdx[chunkIdx]
		if hostIdx != feParam.NodeIdx {
			metricTraffic.WithLabelValues("pushChunk").Add(float64(feTxn.ChunkSize))
			metricTraffic.WithLabelValues("intra").Add(float64(feTxn.ChunkSize))
		}
		metricReq.WithLabelValues("pushChunk").Inc()
		if myconst.UseFakePush {
			go fakePush(hostIdx, chunkIdx, "chunk", feTxn, wg, 0)
		} else {
			go pushToATS(hostIdx, splitData[chunkIdx], "chunk", feTxn, wg, nPushRetry)
		}
	}

	time.Sleep(time.Second)
	wg.Wait()
	appendDebugString(feTxn, fmt.Sprintf("pushChunkFinished-%v", chunkIdxs))
}

func pushMissingChunksToPeers(dat []byte, feTxn *FrontendTxnData) {

	// how do I know a chunk is missing instead of just late?

}

func streamingPush() {

}

func fakePush(hostIdx int, chunkIdx int, objType string, feTxn *FrontendTxnData, wg *sync.WaitGroup, nRetry int) {
	if wg != nil {
		wg.Add(1)
		defer wg.Done()
	}
	var err error
	var req *http.Request
	var resp *http.Response

	if objType != "full" && objType != "chunk" {
		slogger.DPanicf("txn %v: unknown objType %v", feTxn.TxnID, objType)
	}

	var pushUrl string = "http://" + feParam.Hosts[hostIdx] + "/" + feTxn.Route + "/" + feTxn.ReqContent
	req, err = http.NewRequest(http.MethodGet, pushUrl, nil)
	if err != nil {
		metricErr.WithLabelValues("push").Inc()
		slogger.DPanicf("txn %d: failed to create new requests %v", feTxn.TxnID, err)
		return
	}
	req.Header.Add("Fake-Push", "true")
	req.Header.Add("Obj-Type", objType)
	if chunkIdx >= 0 {
		req.Header.Add("Ec-Chunk", fmt.Sprintf("%d_%d_%d", feParam.ECN, feParam.ECK, chunkIdx))
	}

	resp, err = atsPushClient.Do(req)
	if err != nil || (resp.StatusCode != 200 && resp.StatusCode != 201) {
		if err != nil {
			metricErr.WithLabelValues("push").Inc()
		} else {
			metricErr.WithLabelValues("pushErrStatusCode" + strconv.Itoa(resp.StatusCode)).Inc()
			slogger.Warnf("fake push status %v", resp.Status)
		}
	}

	defer resp.Body.Close()

	if _, err = io.Copy(ioutil.Discard, resp.Body); err != nil {
		slogger.DPanicf("discard body err %v, %v", err, feTxn.DebugString)
	}
}

func pushToATS(hostIdx int, dat []byte, objType string, feTxn *FrontendTxnData, wg *sync.WaitGroup, nRetry int) {
	if wg != nil {
		wg.Add(1)
		defer wg.Done()
	}
	var err error
	var req *http.Request
	var resp *http.Response

	if objType != "full" && objType != "chunk" {
		slogger.DPanicf("txn %v: unknown objType %v", feTxn.TxnID, objType)
	}

	headerReader := bytes.NewReader([]byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nobjType:%s\r\nobjSize:%d\r\n\r\n",
		objType, feTxn.ObjSize)))
	bodyReader := bytes.NewReader(dat)
	reader := io.MultiReader(headerReader, bodyReader)

	var pushUrl string
	if hostIdx == feParam.NodeIdx {
		pushUrl = "http://127.0.0.1:" + myconst.ATSPort + "/" + feTxn.Route + "/" + feTxn.ReqContent
	} else {
		pushUrl = "http://" + feParam.Hosts[hostIdx] + "/" + feTxn.Route + "/" + feTxn.ReqContent
	}
	if req, err = http.NewRequest("PUSH", pushUrl, reader); err != nil {
		slogger.DPanicf("txn %v: create push request error %v", feTxn.TxnID, err)
		return
	}
	objSizeNDigits := int(math.Ceil(math.Log10(float64(feTxn.ObjSize + 1))))
	if objType == "chunk" {
		req.ContentLength = 44 + feTxn.ChunkSize + int64(objSizeNDigits)
	} else if objType == "full" {
		req.ContentLength = 43 + feTxn.ObjSize + int64(objSizeNDigits)
	}

	//slogger.Debugf("txn %v: push %v, content-length %v", feTxn.TxnID,
	//	"http://"+feParam.Hosts[hostIdx]+"/akamai/"+feTxn.ReqContent, req.ContentLength)

	resp, err = atsPushClient.Do(req)
	if err != nil || (resp.StatusCode != 200 && resp.StatusCode != 201) {
		if err != nil {
			metricErr.WithLabelValues("push").Inc()
		} else {
			metricErr.WithLabelValues("pushErrStatusCode" + strconv.Itoa(resp.StatusCode)).Inc()
			//slogger.Warnf("push status code %v %v", resp.StatusCode, resp.Status)
		}

		if nRetry > 0 {
			if nRetry == nPushRetry {
				metricErr.WithLabelValues("firstPush").Inc()
			}
			time.Sleep(time.Duration(2000/nRetry/nRetry) * time.Millisecond)
			pushToATS(hostIdx, dat, objType, feTxn, nil, nRetry-1)
		} else {
			slogger.Warnf("txn %v: final push %v to ATS %v err %v, objType %v, chunkSize %v, objSize %v, host %v",
				feTxn.TxnID, feTxn.ReqContent, hostIdx, err, objType, feTxn.ChunkSize, feTxn.ObjSize, feParam.Hosts[hostIdx])
			metricErr.WithLabelValues("finalPush").Inc()
		}
		if resp != nil {
			_ = resp.Body.Close()
		}
		return
	}

	defer resp.Body.Close()

	if _, err = io.Copy(ioutil.Discard, resp.Body); err != nil {
		slogger.DPanicf("txn %v: discard body err %v", feTxn.TxnID, err)
	}
}
