package main

import (
	"bytes"
	"fmt"
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"time"
)

var lastUnavailWarnTs time.Time = time.Now()

//deprecated
func C2DNSimple(feTxn *FrontendTxnData) {
	// go fetch from each ATS, each ATS will fetch from host:port/akamai/CodedOrigin/n_k/chunkID/objID_objSize
	// note the url is changed
	slogger.Panic("C2DNSimple deprecated")
	handleChunk(feTxn, false)
}

func FrontendMain(feTxn *FrontendTxnData) {
	// most of the time: send to local ATS with exactly the same url
	// when local cache server is down: frontend is simply a load balancer and request will be sent to a remote ATS
	// perform load balancing here

	feTxn.NFrontendMain += 1
	if feTxn.NFrontendMain > 6 {
		slogger.DPanicf("How can a request go through frontend main so many times? %v", feTxn.DebugString)
		return
	}
	/* localECIdx indicates the index of the ECN servers, for CDN, it just means the index of the two servers */
	if feParam.Mode != "noRep" {
		if feTxn.PeerIdx[0] == feParam.NodeIdx {
			feTxn.localECIdx = 0
		} else if feTxn.PeerIdx[1] == feParam.NodeIdx {
			feTxn.localECIdx = 1
		} else {
			feTxn.localECIdx = -1
			/* in this case, any of them can be local, let's use a random one */
			feTxn.localECIdx = rand.Intn(2)
			if feTxn.ReqContent[:1] != "a" && time.Since(lastUnavailWarnTs).Seconds() > 20 {
				slogger.Warnf("unavailability observed")
				//slogger.Warnf("txn %d frontend unavailability different from client unavailability (time is not synced), "+
				//	"req %v, bucket %v possible hosts %v, nodeIdx %v?",
				//	feTxn.TxnID, feTxn.ReqContent, feTxn.Bucket, feTxn.PeerIdx, feParam.NodeIdx)
				lastUnavailWarnTs = time.Now()
				metricErr.WithLabelValues("unavailabilityDiff").Inc()
			}
		}
	}

	/* CDN: check whether one of the two lead servers has the object
	 * C2DN: check which chunk is available */
	checkPeerHit(feTxn)
	appendDebugString(feTxn, fmt.Sprintf("localECIdx_%d", feTxn.localECIdx))

	/* if it is served from cluster, whether it is local hit or fetched via ICP */
	feTxn.localHit = feTxn.IsPeerHit[feTxn.localECIdx]
	if feParam.Mode != "noRep" {
		feTxn.alternateLeadHit = feTxn.IsPeerHit[1-feTxn.localECIdx]
	}

	feTxn.FirstReqHostIdx = feTxn.PeerIdx[feTxn.localECIdx]
	if (!feTxn.localHit) && feTxn.alternateLeadHit {
		metricReq.WithLabelValues("localMissAlterHit").Inc()
		if feTxn.ObjType == "full" {
			metricTraffic.WithLabelValues("localMissAlterHitFull").Add(float64(feTxn.ObjSize))
		}
		feTxn.FirstReqHostIdx = feTxn.PeerIdx[1-feTxn.localECIdx]
	}

	if feTxn.localHit || feTxn.alternateLeadHit {
		if feTxn.ObjType == "full" {
			handleFullObj(feTxn, false)
		} else if feTxn.ObjType == "chunk" {
			handleChunk(feTxn, true)
		} else {
			slogger.DPanicf("unknown objType \"%v\", %v", feTxn.ObjType, feTxn)
		}
	} else {
		/* we do not have object/chunk cached for this object, so send to ATS and
		 * ask it to fetch the full object
		 */
		//feTxn.ObjType = "full"
		handleFullObj(feTxn, true)
	}
}

func handleFullObj(feTxn *FrontendTxnData, peerCheckIsMiss bool) {
	/* TODO: we are not doing incremental save or coding, we fetch the big object, save in memory and do everything */
	// could be local hit or miss, if miss this was fetched from origin
	// since this is full object, if it is hit, then it means that this obj does not need encoding
	// if it is miss, then it might need encoding, so check whether it needs encoding
	var url string
	var err error
	var resp *http.Response

	url = fmt.Sprintf("http://%s/%s/%s", feParam.Hosts[feTxn.FirstReqHostIdx], feTxn.Route, feTxn.ReqContent)

	/* retrieve object */
	resp, err = atsGetClient.Get(url)
	if err != nil {
		HandleFailedReq(feTxn, fmt.Sprintf("local fetch err %v", err), "localFetch", true)
		return
	}
	if resp.StatusCode != http.StatusOK {
		appendDebugString(feTxn, fmt.Sprintf("localFetchFailed-%v", resp.Status))
		metricErr.WithLabelValues("localFullObjFetch").Inc()
		if resp.Body != nil {
			_, _ = io.Copy(ioutil.Discard, resp.Body)
			_ = resp.Body.Close()
		}
		FrontendMain(feTxn)

		//HandleFailedReq(feTxn, fmt.Sprintf("local fetch status %v", resp.Status), "localFetch", true)
		return
	}

	feTxn.FirstReqResp = resp
	cacheHit, _, _, _ := ParseViaHeader(resp.Header.Get("Via"))

	/* it is possible that this fetch was a miss, but now it becomes a chunk hit */
	appendDebugString(feTxn, fmt.Sprintf("firstRespBack-%v-%v", cacheHit, resp.Header.Get("Obj-Type")))

	/* if this chunk check was late - feTxn.IsPeerHit[0] and feTxn.IsPeerHit[1] were false earlier,
	 * but now feTxn.IsPeerHit[0] is true, we cannot detect it
	 */
	if cacheHit && peerCheckIsMiss {
		/* not only this can happen sometimes, the fetched content can also become a chunk */
		slogger.Warnf("very rare, obj miss during peer check, but hit during get %v", feTxn.DebugString)
		metricErr.WithLabelValues("peerCheckMissBecomeHit").Inc()
		if feTxn.FirstReqHostIdx == feParam.NodeIdx {
			feTxn.localHit = true
			feTxn.IsPeerHit[feTxn.localECIdx] = cacheHit
		} else {
			feTxn.alternateLeadHit = true
			feTxn.IsPeerHit[1-feTxn.localECIdx] = cacheHit
		}

		if resp.Header.Get("Obj-Type") == "chunk" {
			/* need to restart the request */
			metricErr.WithLabelValues("peerCheckMissBecomeChunkHit").Inc()
			feTxn.ObjType = "chunk"
			_, _ = io.Copy(ioutil.Discard, resp.Body)
			_ = resp.Body.Close()
			appendDebugString(feTxn, "restartTxnPeerCheckMissBecomeChunkHit")
			FrontendMain(feTxn)
			return
		}
	}

	CheckObjCachePolicy(feTxn)
	var reader io.Reader = feTxn.FirstReqResp.Body
	buf := &bytes.Buffer{}
	buf.Grow(int(feTxn.ObjSize))

	if feTxn.localHit || feTxn.alternateLeadHit {
		if cacheHit, _, _, _ := ParseViaHeader(resp.Header.Get("Via")); !cacheHit {
			slogger.Warnf("peer check is hit, but fetch is miss, this should be very rare")
			metricErr.WithLabelValues("peerCheckHitBecomeMiss").Inc()
			_, _ = io.Copy(ioutil.Discard, resp.Body)
			_ = resp.Body.Close()
			appendDebugString(feTxn, "restartTxnHitBecomeMiss")
			FrontendMain(feTxn)
			return
		}

		metricReqClient.WithLabelValues("fullObjHit").Inc()
		metricByteClient.WithLabelValues("fullObjHit").Add(float64(feTxn.ObjSize))
		if !feTxn.localHit || feTxn.FirstReqHostIdx != feParam.NodeIdx {
			metricTraffic.WithLabelValues("ICPFull").Add(float64(feTxn.ObjSize))
			metricTraffic.WithLabelValues("intra").Add(float64(feTxn.ObjSize))
			metricReq.WithLabelValues("ICPFull").Inc()
			metricReq.WithLabelValues("intra").Inc()
		}

		feTxn.Ctx.Response.Header.Set("Via", "[cH]")
		if feTxn.SaveToFeRAM {
			reader = io.TeeReader(feTxn.FirstReqResp.Body, buf)
		}
	} else {
		metricReqClient.WithLabelValues("fullObjMiss").Inc()
		metricByteClient.WithLabelValues("fullObjMiss").Add(float64(feTxn.ObjSize))
		metricTraffic.WithLabelValues("origin").Add(float64(feTxn.ObjSize))
		metricTraffic.WithLabelValues("originFull").Add(float64(feTxn.ObjSize))

		feTxn.Ctx.Response.Header.Set("Via", "[cM]")
		// we will push this object or encoded chunks
		reader = io.TeeReader(feTxn.FirstReqResp.Body, buf)
	}

	go func() {
		n, err := io.Copy(feTxn.Pipew, reader)
		if err != nil || n != feTxn.ObjSize {
			s := fmt.Sprintf("error copy from local resp to client, err %v, copied %v bytes", err, n)
			HandleFailedReq(feTxn, s, "pipeFullToClient", true)
			return
		} else {
			feTxn.SendSize = n
			if err = feTxn.FirstReqResp.Body.Close(); err != nil {
				HandleFailedReq(feTxn, fmt.Sprintf("err closing close resp %v", err), "pipeFullToClient", true)
			}
			if err = feTxn.Pipew.Close(); err != nil {
				HandleFailedReq(feTxn, fmt.Sprintf("err closing write pipe %v", err), "pipeFullToClient", true)
			}
		}

		if feTxn.localHit || feTxn.alternateLeadHit {
			// we decide to store objects into fe cache when it is fullObj/chunk hit
			if feTxn.SaveToFeRAM {
				saveToFeRAM(feTxn, buf.Bytes())
			} // full obj hit is finished up to here
		} else {
			// an full object miss will result in
			// 1. obj <= codingSizeThreshold being stored as full obj, pushed to replica hosts
			// 2. obj > codingSizeThreshold being chunked and send to K peers, notice that we don't encode here
			if feParam.Mode == "twoRepAlways" || feParam.Mode == "noRep" {
				pushFullObj(buf.Bytes(), feTxn)
			} else if feParam.Mode == "C2DN" || feParam.Mode == "naiveCoding" {
				if feTxn.ShouldCode {
					// first hit is chunking and we don't need to push data chunks
					var chunkIdxs []int
					for chunkIdx := 0; chunkIdx < feParam.ECN; chunkIdx++ {
						chunkIdxs = append(chunkIdxs, chunkIdx)
					}
					pushChunks(buf.Bytes(), feTxn, chunkIdxs)
				} else {
					pushFullObj(buf.Bytes(), feTxn)
					// because we have asked origin to cooperate (just for reducing unnecessary push in emulating C2DN),
					// so origin knows which objects will be coded, and for objects that will be coded,
					// origin will add no-cache header, so ATS does not store it, on the other end,
					// for object that will be coded, origin does not add no-cache header,
					// so we don't need to push full objects into local ATS
					//pushToATS(feParam.NodeIdx, buf.Bytes(), "full", feTxn, nil, 1)
					//slogger.Debugf("txn %v: push full obj %v", feTxn.TxnID, feTxn.ReqContent)
				}
			} else {
				slogger.Panicf("unknown mode %v", feParam.Mode)
			}
		}
		appendDebugString(feTxn, "handleLocalFullFinish")
		if myconst.DebugLevel >= myconst.PerReqLogging {
			slogger.Debug(feTxn.DebugString)
		}
	}()
}
