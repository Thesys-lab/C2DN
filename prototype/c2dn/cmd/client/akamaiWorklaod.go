package main

import (
	"bufio"
	"encoding/binary"
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
	"io"
	"os"
	"sync"
)

var (
	traceStartTs uint32
)

func IsRemote(objID uint32) bool {
	//if clientID == -1 && myutils.Hash(fmt.Sprintf("%d_%d", req.ID, req.Size))%10002 >= 1*100 {

	if objID%1001 <= 2 {
		return true
	} else {
		return false
	}
}

func CalBucket(objID uint32) int {
	return int(objID) % myconst.NBuckets
}

func readOneReq(reader *bufio.Reader) (FullRequest, error) {
	var req Request
	var fullReq FullRequest

	err := binary.Read(reader, binary.LittleEndian, &req)
	if err != nil {
		return fullReq, err
	}
	fullReq.Timestamp = req.Timestamp - traceStartTs
	fullReq.ID = req.ID
	fullReq.Size = req.Size
	fullReq.Bucket = req.ID % myconst.NBuckets
	fullReq.Remote = IsRemote(fullReq.ID)

	//hashKey := fmt.Sprintf("%d", fullReq.Bucket)
	//peers := lbWithNoFailure.GetNodes(0, hashKey, 2)
	//fullReq.OriginalHostIdx, _ = strconv.Atoi(peers[0])
	//fullReq.AlternateHostIdx, _ = strconv.Atoi(peers[1])

	peers := lbWithNoFailure.GetNodesFromMapping(0, int(fullReq.Bucket))
	fullReq.OriginalHostIdx = peers[0]
	fullReq.AlternateHostIdx = peers[1]

	clientTraceMetric.WithLabelValues("all_read").Inc()
	return fullReq, err
}

func LoadAkamaiBinData(datPath string, reqChan chan FullRequest, wg *sync.WaitGroup) {
	if wg != nil {
		wg.Add(1)
		defer (*wg).Done()
	}

	var err error
	var req FullRequest
	var nUsedReq uint64 = 0
	slogger.Infof("Loading AkamaiWorkLoad %v with requestRate %v Mbps, replay staart ts %v, end ts %v, speedup %v",
		datPath, cParam.RequestRateMbps, cParam.ReplayStartTs, cParam.ReplayEndTs, cParam.ReplaySpeedup)
	slogger.Debug("NOTE: jason has changed the size lower limit of the request")
	lastSeenTs := make(map[uint32]uint32)

	file, err := os.Open(datPath)
	if err != nil {
		slogger.Fatal("failed to open trace file " + datPath)
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	req, err = readOneReq(reader)

	// discard the first 2000 requests because the init trace is not correct
	//for i := 0; i < 2000; i++ {
	//	req, err = readOneReq(reader)
	//}

	traceStartTs = req.Timestamp
	req.Timestamp = 0
	// for changed request rate
	var fakeTs uint32 = 0
	var reqByteCurTs uint64 = 0
	seenObj := make(map[uint32]bool)
	nReq := 0

	for {
		if err == nil {
			nReq += 1
			if req.Timestamp < cParam.ReplayStartTs {
				clientTraceMetric.WithLabelValues("skip_start").Inc()
				req, err = readOneReq(reader)
				continue
			} else if req.Timestamp > cParam.ReplayEndTs {
				break
			}

			if cParam.IgnoreRemoteReq && IsRemote(req.ID) {
				clientTraceMetric.WithLabelValues("skip_remote").Inc()
				req, err = readOneReq(reader)
				continue
			}

			/* we need to deterministically choose between original host and alternate host */
			randChosenHost := req.OriginalHostIdx
			if cParam.RandomRoute && nReq%2 == 0 {
				randChosenHost = req.AlternateHostIdx
			}

			if cParam.ClientID != -1 {
				if randChosenHost != cParam.ClientID {
					clientTraceMetric.WithLabelValues("different_client").Inc()
					req, err = readOneReq(reader)
					continue
				}
			} else if !IsRemote(req.ID) {
				req, err = readOneReq(reader)
				continue
			}

			if cParam.UniqueObj {
				clientTraceMetric.WithLabelValues("non_uniq").Inc()
				if _, found := seenObj[req.ID]; found {
					req, err = readOneReq(reader)
					continue
				} else {
					seenObj[req.ID] = true
				}
			}

			if cParam.RequestRateMbps <= 0 {
				// fix paced trace replay
				req.Timestamp = uint32(float64(req.Timestamp-cParam.ReplayStartTs) / cParam.ReplaySpeedup)
			} else {
				req.Timestamp = fakeTs
				reqByteCurTs += uint64(req.Size)
				if int(reqByteCurTs*8/1024/1024) >= cParam.RequestRateMbps {
					reqByteCurTs = 0
					fakeTs++
				}
			}

			if req.Size < 11 {
				req.Size = 11
			}
			if req.Size > 100000000 {
				if ts, found := lastSeenTs[req.ID]; found {
					if req.Timestamp-ts < 10 {
						clientTraceMetric.WithLabelValues("ignore_large").Inc()
						req, err = readOneReq(reader)
						continue
					}
				} else {
					lastSeenTs[req.ID] = req.Timestamp
				}
				//if req.Size > 500000000 {
				//	req.Size = 500000000
				//}
			}
			if myconst.DebugLevel >= myconst.PerReqDetailedLogging {
				slogger.Debugf("put req %v in chan", req)
			}

			reqChan <- req
			nUsedReq += 1
			clientTraceMetric.WithLabelValues("used_read").Inc()
			req, err = readOneReq(reader)

		} else if err == io.EOF {
			break
		} else {
			slogger.DPanic("trace read error ", err)
		}
	}
	slogger.Infof("finish reading trace %v useful requests",
		myutils.GetCounterValue(clientTraceMetric, []string{"used_read"}))

	slogger.Infof("finish reading trace %v skip_start, %v skip_remote, %v different_client, %v non_uniq requests",
		myutils.GetCounterValue(clientTraceMetric, []string{"skip_start"}),
		myutils.GetCounterValue(clientTraceMetric, []string{"skip_remote"}),
		myutils.GetCounterValue(clientTraceMetric, []string{"different_client"}),
		myutils.GetCounterValue(clientTraceMetric, []string{"non_uniq"}),
	)
	close(reqChan)
}
