package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

/**
this is the checkOneMore server implementation, similar to ICP, but only checks the necessary servers
*/

func CatchPanicWithFeTxn(feTxn *FrontendTxnData, chunkIdx int) {
	if r := recover(); r != nil {
		slogger.DPanicf("Caught panic due to %v, chunkIdx %d, %v", r, chunkIdx, feTxn.DebugString)
	}
}

/* check whether one peer has cached the object, but do not fetch the object nor check origin */
func _checkOnePeerHit(feTxn *FrontendTxnData, url string, chunkIdx int) {
	defer CatchPanicWithFeTxn(feTxn, chunkIdx)

	var err error
	var req *http.Request
	var resp *http.Response
	objType := ""

	if req, err = http.NewRequest(http.MethodHead, url, nil); err != nil {
		slogger.DPanicf("failed to create head requests %v", err)
	}
	req.Header.Set("Cache-Control", "only-if-cached")

	resp, err = atsGetClient.Do(req)
	if err != nil {
		HandleFailedReq(feTxn, fmt.Sprintf("err checking peers %v, %v", url, err),
			"peerCheck", false)
		return
	}

	//appendDebugString(feTxn, fmt.Sprintf("chunkIdx%d-header-%v", chunkIdx, resp.Header))

	cacheHit, _, _, _ := ParseViaHeader(resp.Header.Get("Via"))

	feTxn.IsPeerHit[chunkIdx] = cacheHit
	if cacheHit {
		atomic.AddInt32(&feTxn.nPeerHit, 1)
		objType = resp.Header.Get("Obj-Type")
		feTxn.Mtx.Lock()
		if feTxn.ObjType == "" {
			feTxn.ObjType = objType
			appendDebugString(feTxn, "setObjType- "+objType)
		} else if feTxn != nil && feTxn.ObjType != objType {
			slogger.DPanicf("objType mismatch %v %v, chunk %d, %v", feTxn.ObjType, resp.Header.Get("Obj-Type"), chunkIdx, feTxn.DebugString)
		}
		feTxn.Mtx.Unlock()

		objSize, err := strconv.ParseInt(resp.Header.Get("Obj-Size"), 10, 64)
		if err != nil || objSize != feTxn.ObjSize {
			slogger.DPanicf("object size from local ATS response %v different from reuqest %v, resp header %v",
				objSize, feTxn.ReqContent, resp.Header)
		}
	}

	if atomic.AddInt32(&feTxn.peerCheckFinished, 1) >= int32(feParam.ECN) {
		feTxn.Cond.Signal()
	}

	appendDebugString(feTxn, fmt.Sprintf("peerCheck%d-%v-%v", chunkIdx, cacheHit, objType))

	if _, err = io.Copy(ioutil.Discard, resp.Body); err != nil {
		slogger.DPanicf("txn %v: err discarding miss parity chunk %v, url %v\n", feTxn.TxnID, err, url)
		return
	}
	_ = resp.Body.Close()
}

func checkPeerHit(feTxn *FrontendTxnData) {
	feTxn.peerCheckFinished = 0
	nCheck := feParam.ECN
	if feParam.Mode == "C2DN" && feTxn.PeerIdx[feParam.ECN] != -1 && feTxn.PeerIdx[feParam.ECN] != feTxn.PeerIdx[feParam.ECN-1] {
		nCheck += 1
	}
	feTxn.IsPeerHit = make([]bool, nCheck)
	for i := 0; i < nCheck; i++ {
		if feTxn.PeerIdx[i] == -1 {
			continue
		}
		feTxn.IsPeerHit[i] = false
		url := fmt.Sprintf("http://%s/%s/%s", feParam.Hosts[feTxn.PeerIdx[i]], feTxn.Route, feTxn.ReqContent)
		go _checkOnePeerHit(feTxn, url, i)
	}

	feTxn.Mtx.Lock()
	if feParam.Mode == "twoRepAlways" || feParam.Mode == "noRep" {
		for atomic.LoadInt32(&feTxn.peerCheckFinished) < int32(feParam.ECN) {
			feTxn.Cond.Wait()
		}
	} else if feParam.Mode == "C2DN" || feParam.Mode == "naiveCoding" {
		for atomic.LoadInt32(&feTxn.peerCheckFinished) < int32(feParam.ECK) {
			feTxn.Cond.Wait()
		}
	}
	feTxn.Mtx.Unlock()
}

func fetchChunk(feTxn *FrontendTxnData, chunkIdx int, hostIdx int) {
	if chunkIdx >= feParam.ECN {
		// TODO: this might be a problem
		chunkIdx = feParam.ECN - 1
	}
	feTxn.ChunkStat.IsAvailable[chunkIdx] = false

	if chunkIdx >= feParam.ECK {
		startWaitTs := time.Now()
		for atomic.LoadInt32(&feTxn.ChunkStat.NumAvailable) == 0 {
			time.Sleep(time.Millisecond)
		}
		time.Sleep(time.Duration(time.Since(startWaitTs).Nanoseconds() / 5 * int64(chunkIdx-feParam.ECK+1)))
		if myconst.DebugLevel >= myconst.PerReqDetailedLogging {
			slogger.Debugf("sleep %v before fetching parity",
				time.Since(startWaitTs).Milliseconds())
		}

	}

	if atomic.LoadInt32(&feTxn.ChunkStat.NumAvailable) >= int32(feParam.ECK) {
		metricReq.WithLabelValues("skipFetch").Inc()
		metricTraffic.WithLabelValues("skipFetch").Add(float64(feTxn.ChunkSize))
		appendDebugString(feTxn, fmt.Sprintf("skip_c%v_h%v", chunkIdx, hostIdx))
		if myconst.DebugLevel >= myconst.PerReqLogging {
			slogger.Debugf("%d chunks available, skip chunk %v host %v, %v",
				atomic.LoadInt32(&feTxn.ChunkStat.NumAvailable), chunkIdx, hostIdx, feTxn.DebugString)
		}
		atomic.AddInt32(&feTxn.ChunkStat.NumSkipped, 1)
		return
	}

	if chunkIdx != feTxn.localECIdx {
		/* this does not include skipped chunk */
		metricReq.WithLabelValues("ICPChunk").Inc()
		metricTraffic.WithLabelValues("ICPChunk").Add(float64(feTxn.ChunkSize))
		metricReq.WithLabelValues("intra").Inc()
		metricTraffic.WithLabelValues("intra").Add(float64(feTxn.ChunkSize))
	}

	var err error
	var url string
	var req *http.Request
	var resp *http.Response
	url = fmt.Sprintf("http://%s/%s/%s", feParam.Hosts[hostIdx], feTxn.Route, feTxn.ReqContent)
	if req, err = http.NewRequest("GET", url, nil); err != nil {
		slogger.DPanicf("failed to create requests %v", err)
	}
	req.Header.Set("Obj-Type", "chunk")

	// setup chunk request
	if chunkIdx < feParam.ECK {
		// for data chunk, ATS can fetch from origin with chunk
		req.Header.Set("Ec-Chunk", fmt.Sprintf("%d_%d_%d", feParam.ECN, feParam.ECK, chunkIdx))
	} else {
		// for parity chunk, it cannot be a miss, but in case, we add this as assert
		req.Header.Set("Cache-Control", "only-if-cached")
		metricReq.WithLabelValues("parityFetch_" + strconv.Itoa(chunkIdx-feParam.ECK+1)).Inc()
	}

	if resp, err = atsGetClient.Do(req); err != nil {
		HandleFailedReq(feTxn, fmt.Sprintf("err fetching chunk from %v, %v", url, err),
			"remoteChunkFetch", false)
		return
	}

	if resp.StatusCode != http.StatusOK {
		//HandleFailedReq(feTxn, fmt.Sprintf("chunk fetch incorrect status %v, url %v", resp.Status, url),
		//	"chunkFetch", true)
		//atomic.AddInt32(&feTxn.ChunkStat.NumFailed, 1)
		metricErr.WithLabelValues("parityChunkHitBecomeMiss").Inc()
		_, err = io.Copy(ioutil.Discard, resp.Body)
		_ = resp.Body.Close()
		return
	}

	appendDebugString(feTxn, fmt.Sprintf("chunkFetch_c%d-h%d", chunkIdx, hostIdx))

	if chunkHit, _, _, _ := ParseViaHeader(resp.Header.Get("Via")); chunkHit != feTxn.IsPeerHit[chunkIdx] {
		slogger.Warnf("peer check and chunk fetch results are not the same %v peerHit[%d] %v", chunkHit, chunkIdx, feTxn.IsPeerHit)
		metricErr.WithLabelValues("chunkGetDiffPeerCheck").Inc()
	}

	// this cannot be here, otherwise cleanup will think this chunk needs cleanup,
	// and also when piping to client will start to read from this chunk, when it has already read several rounds from
	// other chunks
	//feTxn.ChunkStat.IsAvailable[chunkIdx] = true
	feTxn.Mtx.Lock()
	if _, err = TxnFinishCache.Get(feTxn.TxnIDByte); err != nil {
		// this chunk is still needed
		feTxn.Resps[chunkIdx] = resp
		//feTxn.RespReaders[chunkIdx] = resp.Body
		feTxn.ChunkStat.IsAvailable[chunkIdx] = true

		atomic.AddInt32(&feTxn.ChunkStat.NumAvailable, 1)
		if chunkIdx < feParam.ECK {
			atomic.AddInt32(&feTxn.ChunkStat.NumDataAvailable, 1)
		}
		nFailed := atomic.LoadInt32(&feTxn.ChunkStat.NumFailed)
		if atomic.LoadInt32(&feTxn.ChunkStat.NumAvailable) >= int32(feParam.ECK) || nFailed > int32(feParam.ECN-feParam.ECK) {
			feTxn.Cond.Signal()
		}
	} else {
		// the chunk is not needed, clean myself
		//feTxn.ChunkStat.NumSkipped++
		metricReq.WithLabelValues("chunkRespSkipped").Inc()
		if _, err = io.Copy(ioutil.Discard, resp.Body); err != nil {
			HandleFailedReq(feTxn, fmt.Sprintf("err discarding data for %v", err), "chunkFetch", false)
		}
		_ = resp.Body.Close()
	}
	feTxn.Mtx.Unlock()
}

func handleChunk(feTxn *FrontendTxnData, hasLocalResp bool) {
	if !hasLocalResp {
		slogger.DPanic("not supported")
	}
	if feTxn.ObjSize < myconst.CodingObjSizeThreshold {
		slogger.DPanicf("txn %v: obj size too small, only %v bytes, feTxn %v", feTxn.TxnID, feTxn.ObjSize, feTxn)
	}

	feTxn.ChunkSize = feTxn.ObjSize / int64(feParam.ECK)
	if feTxn.ObjSize%int64(feParam.ECK) != 0 {
		feTxn.ChunkSize += 1
	}

	feTxn.TxnIDByte = make([]byte, 8)
	binary.LittleEndian.PutUint64(feTxn.TxnIDByte, feTxn.TxnID)

	feTxn.Resps = make([]*http.Response, feParam.ECN+1)
	feTxn.ChunkStat.IsAvailable = make([]bool, feParam.ECN+1)
	//feTxn.RespReaders = make([]io.ReadCloser, feParam.ECN + 1)

	nDataChunkFetchFromOrigin := 0
	for chunkIdx, chunkHit := range feTxn.IsPeerHit {
		if chunkIdx < feParam.ECK {
			if !chunkHit && int(feTxn.nPeerHit)+nDataChunkFetchFromOrigin < feParam.ECK {
				metricTraffic.WithLabelValues("origin").Add(float64(feTxn.ChunkSize))
				metricTraffic.WithLabelValues("originChunk").Add(float64(feTxn.ChunkSize))
				metricReq.WithLabelValues("origin").Inc()
				nDataChunkFetchFromOrigin += 1
			}
			go fetchChunk(feTxn, chunkIdx, feTxn.PeerIdx[chunkIdx])
		} else {
			/* parity chunk is only fetched if it is there */
			if chunkIdx < feParam.ECN {
				if chunkHit {
					go fetchChunk(feTxn, chunkIdx, feTxn.PeerIdx[chunkIdx])
				} else {
					//atomic.AddInt32(&feTxn.ChunkStat.NumPChunkMiss, 1)
					feTxn.Mtx.Lock()
					feTxn.ChunkStat.MissingPChunkIdx = append(feTxn.ChunkStat.MissingPChunkIdx, chunkIdx)
					feTxn.Mtx.Unlock()
				}
			} else if chunkIdx == feParam.ECN {
				if feTxn.IsPeerHit[chunkIdx-1] {
					continue
				}
			} else {
				slogger.DPanicf("chunkIdx %v too large", chunkIdx)
			}
		}
	}

	feTxn.Mtx.Lock()
	for atomic.LoadInt32(&feTxn.ChunkStat.NumAvailable) < int32(feParam.ECK) &&
		atomic.LoadInt32(&feTxn.ChunkStat.NumFailed) <= int32(feParam.ECN-feParam.ECK) {
		feTxn.Cond.Wait()
	}
	feTxn.Mtx.Unlock()
	if atomic.LoadInt32(&feTxn.ChunkStat.NumFailed) > int32(feParam.ECN-feParam.ECK) {
		HandleFailedReq(feTxn, fmt.Sprintf("too many %d failed chunk requests", feTxn.ChunkStat.NumFailed),
			"tooManyFailedChunkReq", true)
		return
	}

	appendDebugString(feTxn, "enoughChunkReady")

	// we don't want new chunks to be available after this point
	feTxn.Mtx.Lock()
	if err := TxnFinishCache.Set(feTxn.TxnIDByte, []byte{'1'}, 300); err != nil {
		slogger.DPanic("fail to set TxnFinishCache ", err)
	}
	feTxn.Mtx.Unlock()

	if myconst.DebugLevel >= myconst.PerReqDetailedLogging {
		slogger.Debug(feTxn.DebugString)
	}

	if feTxn.ChunkStat.NumDataAvailable > int32(feParam.ECK) {
		slogger.DPanicf("why can there be more than K IsAvailable data chunks %d > %d",
			feTxn.ChunkStat.NumDataAvailable, feParam.ECK)
	}

	if feTxn.nPeerHit >= int32(feParam.ECK) {
		feTxn.Ctx.Response.Header.Set("Via", "[cH]")

		metricReqClient.WithLabelValues("chunkHit").Inc()
		metricByteClient.WithLabelValues("chunkHit").Add(float64(feTxn.ObjSize))
	} else {
		feTxn.Ctx.Response.Header.Set("Via", "[cM]")

		metricReqClient.WithLabelValues("partialHit_" + strconv.Itoa(int(atomic.LoadInt32(&feTxn.nPeerHit)))).Inc()
		metricByteClient.WithLabelValues("partialHit_" + strconv.Itoa(int(atomic.LoadInt32(&feTxn.nPeerHit)))).Add(
			float64(feTxn.ObjSize))
		slogger.Warnf("%v partial hit %v", atomic.LoadInt32(&feTxn.nPeerHit), feTxn)
	}

	// this should be the common case, optimize for common case
	if feTxn.ChunkStat.NumDataAvailable == int32(feParam.ECK) {
		appendDebugString(feTxn, "noDecode")
		metricReq.WithLabelValues("noDecode").Inc()
		go PipeDataChunkToClient(feTxn)
	} else {
		// we need to decode
		appendDebugString(feTxn, "decode")
		metricReq.WithLabelValues("decode").Inc()
		go pipeDataParityChunkToClient(feTxn)
	}
}

func pipeDataParityChunkToClient(feTxn *FrontendTxnData) {
	scoder := myStreamCoders.Get().(*MyStreamCoder)
	respBuffer := scoder.Decode(feTxn)
	myStreamCoders.Put(scoder)

	chunkGetCleanupTxn(feTxn, respBuffer)
}

func PipeDataChunkToClient(feTxn *FrontendTxnData) {

	var n int64 = 0
	var err error

	readers := make([]io.Reader, len(feTxn.Resps))
	respBuffer := &bytes.Buffer{}
	respBuffer.Grow(int(feTxn.ObjSize))
	for i := 0; i < feParam.ECK; i++ {
		readers[i] = io.TeeReader(feTxn.Resps[i].Body, respBuffer)
	}

	if feTxn.ChunkSize <= myconst.CodingSubChunkSize {
		// we only need single read
		slogger.Debugf("txn %v: req %v single round piping", feTxn.TxnID, feTxn.ReqContent)
		for i := 0; i < feParam.ECK-1; i++ {
			slogger.Debugf("txn %v: req %v single round piping i %v", feTxn.TxnID, feTxn.ReqContent, i)
			if n, err = io.Copy(feTxn.Pipew, readers[i]); err != nil {
				HandleFailedReq(feTxn, fmt.Sprintf("err pipe from ats/to client, %v", err), "pipeToClient", true)
				return
			} else {
				feTxn.SendSize += n
			}
		}
		// for last chunk, the real size might be different from stored size
		sizeOfLastChunk := feTxn.ObjSize - int64(feParam.ECK-1)*feTxn.ChunkSize
		slogger.Debugf("txn %v: req %v single round piping last chunk", feTxn.TxnID, feTxn.ReqContent)
		if n, err = io.CopyN(feTxn.Pipew, readers[feParam.ECK-1], sizeOfLastChunk); err != nil || n != sizeOfLastChunk {
			HandleFailedReq(feTxn, fmt.Sprintf("pipe from ats, last chunk err %v/%v, %v", n, sizeOfLastChunk, err),
				"pipeToClient", true)
			return
		} else {
			// because each chunk is of the same size, but ObjSize may not be of multiple of ChunkSize
			// Jason::PotentialBug
			_, _ = io.Copy(ioutil.Discard, feTxn.Resps[feParam.ECK-1].Body)
			feTxn.SendSize += sizeOfLastChunk
		}
		slogger.Debugf("txn %v: req %v single round piping finish", feTxn.TxnID, feTxn.ReqContent)
	} else {
		// so each read we will read either myconst.CodingSubChunkSize or until EOF
		lastBlock := false
		round := 0
		for !lastBlock {
			for i := 0; i < feParam.ECK; i++ {
				if myconst.DebugLevel >= myconst.PerReqDetailedLogging {
					slogger.Debugf("txn %v: req %v send %v round chunk %v, lastBlock %v",
						feTxn.TxnID, feTxn.ReqContent, round, i, lastBlock)
				}
				if (!lastBlock) || i != feParam.ECK-1 {
					if feTxn.SendSize == feTxn.ObjSize {
						if !lastBlock {
							if feTxn.ObjSize%(int64(feParam.ECK)*feTxn.ChunkSize) == 0 {
								lastBlock = true
								break
							} else {
								s := fmt.Sprintf("this is not last block, but SendSize==ObjSize, i %v n %v SendSize %v ObjSize %v, %v",
									i, n, feTxn.SendSize, feTxn.ObjSize, feTxn)
								HandleFailedReq(feTxn, s, "pipeToClient", true)
								return
							}
						}
						// this happens when the size of last block is (1,0,0), so we discard everything in the left over RespReaders
						_, _ = io.Copy(ioutil.Discard, feTxn.Resps[i].Body)
					} else {
						//slogger.Debugf("begin reading buf size %v", respBuffer.Len())
						n, err = io.CopyN(feTxn.Pipew, readers[i], myconst.CodingSubChunkSize)
						//slogger.Debugf("after reading buf size %v, read size %v", respBuffer.Len(), n)
						if err != nil || n != myconst.CodingSubChunkSize {
							if err == io.EOF {
								lastBlock = true
							} else {
								s := fmt.Sprintf("i %v, error copy from resp to pipe, copied %v bytes %v, err %v", i, n, feTxn, err)
								HandleFailedReq(feTxn, s, "pipeToClient", true)
								return
							}
						} else if lastBlock {
							slogger.DPanicf("why last round is true?")
						}
						feTxn.SendSize += n
					}
				} else {
					// this happens when the size of last block is (1,0,0), so we discard everything in the left over RespReaders
					if feTxn.SendSize == int64(feTxn.ObjSize) {
						_, _ = io.Copy(ioutil.Discard, feTxn.Resps[i].Body)
					}

					// last block of last chunk
					sizeOfLastChunk := feTxn.ObjSize%(myconst.CodingSubChunkSize*int64(feParam.ECK)) - (feTxn.ChunkSize%int64(myconst.CodingSubChunkSize))*int64(feParam.ECK-1)
					if sizeOfLastChunk < 0 {
						if int(sizeOfLastChunk) < -feParam.ECK {
							slogger.DPanicf("size of last chunk error %v, %v", sizeOfLastChunk, feTxn)
						}
						sizeOfLastChunk = 0
					}
					if sizeOfLastChunk+feTxn.SendSize != feTxn.ObjSize {
						s := fmt.Sprintf("size of last chunk error, %v %v %v %v, %v", sizeOfLastChunk, n, feTxn.SendSize, feTxn.ObjSize, feTxn)
						HandleFailedReq(feTxn, s, "pipeToClient", true)
						return
					}
					if sizeOfLastChunk > 0 {
						if n, err = io.CopyN(feTxn.Pipew, readers[feParam.ECK-1], sizeOfLastChunk); err != nil {
							s := fmt.Sprintf("error copying last chunk size %v, %v %v", sizeOfLastChunk, err, feTxn)
							HandleFailedReq(feTxn, s, "pipeToClient", true)
							return
						} else {
							feTxn.SendSize += sizeOfLastChunk
						}
					}
					_, _ = io.Copy(ioutil.Discard, feTxn.Resps[feParam.ECK-1].Body)
				}
			}
			round += 1
		}
	}

	if feTxn.SendSize != feTxn.ObjSize {
		slogger.DPanicf("txn %v: req %v, SendSize and ObjSize are different %v/%v, %v",
			feTxn.TxnID, feTxn.ReqContent, feTxn.SendSize, feTxn.ObjSize, feTxn)
	}

	if err = feTxn.Pipew.Close(); err != nil {
		s := fmt.Sprintf("txn %v: req %v, err closing write pipe for client %v, %v, %v",
			feTxn.TxnID, feTxn.ReqContent, fmt.Sprintf("%v", feTxn.Ctx.RemoteAddr()), err, feTxn)
		HandleFailedReq(feTxn, s, "pipeToClient", true)
		return
	}

	//if LatChans.Started {
	//	if feTxn.ChunkStat.HasMissingChunk {
	//		if feTxn.Remote {
	//			LatChansRemote.NoDecodeMiss <- float64(time.Since(feTxn.StartTs).Nanoseconds()) / myconst.MilliSec
	//		} else {
	//			LatChans.NoDecodeMiss <- float64(time.Since(feTxn.StartTs).Nanoseconds()) / myconst.MilliSec
	//		}
	//	} else {
	//		if feTxn.Remote {
	//			LatChansRemote.NoDecodeHit <- float64(time.Since(feTxn.StartTs).Nanoseconds()) / myconst.MilliSec
	//		} else {
	//			LatChans.NoDecodeHit <- float64(time.Since(feTxn.StartTs).Nanoseconds()) / myconst.MilliSec
	//		}
	//	}
	//}

	chunkGetCleanupTxn(feTxn, respBuffer)
}

func chunkGetCleanupTxn(feTxn *FrontendTxnData, buf *bytes.Buffer) {
	appendDebugString(feTxn, "chunkGetCleanup")

	if feTxn.SaveToFeRAM {
		saveToFeRAM(feTxn, buf.Bytes())
	}

	for i := range feTxn.Resps {
		if feTxn.ChunkStat.IsAvailable[i] {
			if feTxn.Resps[i] == nil {
				slogger.DPanicf("txn %v: chunkStat is available, but resp is nil, chunkID %v, %v", feTxn.TxnID, i, feTxn)
			}
			_, _ = io.Copy(ioutil.Discard, feTxn.Resps[i].Body)
			_ = feTxn.Resps[i].Body.Close()
		}
	}
	//slogger.Debugf("txn %v: finishes cleaning unneeded chunks", feTxn.TxnID)

	// now restore all chunks
	// Jason: do we need to restore data chunks?
	// Jason: we do not need to restore data chunks because they should be restored when being fetched from origin
	// Jason: I need to optimize and only push the missing parity chunks
	// but how do I know which chunks are missing

	// this should work if we don't have more than one parity
	//if !feTxn.ChunkStat.IsAvailable[feParam.ECK]{
	//	pushChunks(buf.Bytes(), feTxn, true, true)
	//}

	// this only works if we don't skip getting parity chunk
	if len(feTxn.ChunkStat.MissingPChunkIdx) > 0 {
		pushChunks(buf.Bytes(), feTxn, feTxn.ChunkStat.MissingPChunkIdx)
	}

	// OK, wait for all chunk requests getting back to calculate chunk hit
	//waitTime := 0
	//for waitTime < myconst.HttpTimeOut && feTxn.ChunkStat.NumAvailable+feTxn.ChunkStat.NumPChunkMiss+
	//	feTxn.ChunkStat.NumSkipped+feTxn.ChunkStat.NumFailed < int64(feParam.ECK+feParam.ECX) {
	//	waitTime += 2
	//	time.Sleep(2 * time.Second)
	//}
	//if waitTime > 30 || feTxn.ChunkStat.NumAvailable+feTxn.ChunkStat.NumPChunkMiss+
	//	feTxn.ChunkStat.NumSkipped+feTxn.ChunkStat.NumFailed < int64(feParam.ECK+feParam.ECX) {
	//	slogger.Warnf("txn %v: cleanup, wait time %v, %v", feTxn.TxnID, waitTime, feTxn)
	//}

	//atomic.AddInt64(&(Stat.NumMissXChunk[feTxn.ChunkStat.NumDChunkMiss+feTxn.ChunkStat.NumPChunkMiss]), 1)

	// there are two cases
	// 1. chunk hit and miss or 2. decode and non-decode

	// this is statistics for case 2
	//if feTxn.ChunkStat.NumDChunkHit+feTxn.ChunkStat.NumPChunkHit == int64(feParam.ECK+feParam.ECX) {
	//	atomic.AddInt64(&(Stat.NumAllHit), 1)
	//} else if feTxn.ChunkStat.NumDChunkHit+feTxn.ChunkStat.NumPChunkHit == 0 {
	//	atomic.AddInt64(&(Stat.NumAllMiss), 1)
	//} else if feTxn.ChunkStat.NumDChunkHit+feTxn.ChunkStat.NumPChunkHit >= int64(feParam.ECK) {
	//	atomic.AddInt64(&(Stat.NumPartialHitGood), 1)
	//} else if feTxn.ChunkStat.NumDChunkHit+feTxn.ChunkStat.NumPChunkHit < int64(feParam.ECK) {
	//	atomic.AddInt64(&(Stat.NumPartialHitBad), 1)
	//} else {
	//	slogger.DPanicf("this should not be reached %v+%v+%v+%v, %v, %v+%v+%v",
	//		feTxn.ChunkStat.NumDChunkHit, feTxn.ChunkStat.NumPChunkHit,
	//		feTxn.ChunkStat.NumDChunkMiss, feTxn.ChunkStat.NumPChunkMiss,
	//		feTxn.ChunkStat.NumAvailable, feTxn.ChunkStat.NumDataAvailable,
	//		feTxn.ChunkStat.NumSkipped, feTxn.ChunkStat.NumFailed)
	//}

	//atomic.AddInt64(&(Stat.TrafficFromOrigin), feTxn.ChunkSize*feTxn.ChunkStat.NumDChunkMiss)

	appendDebugString(feTxn, "chunkGetCleanupFinish")
	if myconst.DebugLevel >= myconst.PerReqLogging {
		slogger.Debug(feTxn.DebugString)
	}
}
