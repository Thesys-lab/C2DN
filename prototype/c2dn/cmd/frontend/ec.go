package main

import (
	"bytes"
	"fmt"
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/klauspost/reedsolomon"
	"io"
)

type MyStreamCoder struct {
	Coder     reedsolomon.Encoder
	n, k      int
	blockSize int
	dataBuf   []byte
	shards    [][]byte
}

func NewStreamCoder(n, k, blockSize int) (scoder *MyStreamCoder) {
	scoder = &MyStreamCoder{n: n, k: k, blockSize: blockSize}
	var err error
	scoder.Coder, err = reedsolomon.New(k, n-k, reedsolomon.WithMaxGoroutines(1), reedsolomon.WithCauchyMatrix())
	//scoder.Coder, err = reedsolomon.New(k, n-k, reedsolomon.WithAutoGoroutines(frontend.FIXED_CODING_BLOCK_SIZE), reedsolomon.WithCauchyMatrix())
	if err != nil {
		slogger.Panic("error creating MyStreamCoder ", err)
	}
	scoder.dataBuf = make([]byte, n*blockSize)
	scoder.shards = make([][]byte, n)
	for i := 0; i < n; i++ {
		scoder.shards[i] = scoder.dataBuf[blockSize*i : blockSize*(i+1)]
	}

	return scoder
}

func (scoder *MyStreamCoder) Decode(feTxn *FrontendTxnData) (respBuffer *bytes.Buffer) {
	var err error
	respBuffer = &bytes.Buffer{}
	respBuffer.Grow(int(feTxn.ObjSize))

	var n, lastN = 0, 0
	var lastBlock = false
	remainingSize := feTxn.ObjSize

	firstAvailableReader := 0
	for i := 0; i < len(feTxn.Resps); i++ {
		if feTxn.ChunkStat.IsAvailable[i] {
			firstAvailableReader = i
			break
		}
	}

	rndCnt := 0
	for !lastBlock {
		rndCnt++
		// read one iteration of blocks
		for i := 0; i < scoder.n; i++ {
			if feTxn.ChunkStat.IsAvailable[i] == true {
				n, err = io.ReadAtLeast(feTxn.Resps[i].Body, scoder.shards[i], scoder.blockSize)
				scoder.shards[i] = scoder.shards[i][:n]
				if err != nil {
					// this can happen only when
					if err == io.ErrUnexpectedEOF {
						lastBlock = true
						if myconst.DebugLevel >= myconst.PerReqDetailedLogging2 {
							slogger.Debugf("txn %v: set last block because current chunk gives ErrUnexpectedEOF, i %v, req %v", feTxn.TxnID, i, feTxn.ReqContent)
						}
						if int64(rndCnt) < feTxn.ObjSize/int64(scoder.blockSize*scoder.k) {
							HandleFailedReq(feTxn, fmt.Sprintf("ec round count err %v", err), "chunkReadEC", true)
							slogger.DPanicf("txn %v: get ErrUnexpectedEOF before expected, req %v, i %v, n %v, lastN %v, rndCnt %v",
								feTxn.TxnID, feTxn.ReqContent, i, n, lastN, rndCnt)
						}
					} else {
						HandleFailedReq(feTxn, fmt.Sprintf("ec round count err %v", err), "chunkReadEC", true)
						slogger.DPanicf("txn %v, error in reading from resp %v (%v), %v", feTxn.TxnID, i, err, feTxn)
						_ = feTxn.Pipew.Close()
						for i := 0; i < scoder.n; i++ {
							scoder.shards[i] = scoder.dataBuf[scoder.blockSize*i : scoder.blockSize*(i+1)]
						}
						return nil
					}
				} else {
					if lastBlock {
						slogger.Warnf("txn %v: round %v, this should be the last block, but I don't get ErrUnexpectedEOF, "+
							"i %v, n %v, lastN %v, firstAvailableReader %v, %v", feTxn.TxnID, rndCnt, i, n, lastN, firstAvailableReader, feTxn)
						metricErr.WithLabelValues("writeEcChunk").Inc()
						for i := 0; i < scoder.n; i++ {
							scoder.shards[i] = scoder.dataBuf[scoder.blockSize*i : scoder.blockSize*(i+1)]
						}
						return nil
					}
				}

				// check whether each chunk has the same size       // this might happen
				if i != firstAvailableReader && lastN != 0 && n != lastN {
					slogger.Warnf("txn %v: round %v, i %d, firstAvailableReader %v, lastN %d, "+
						"current read %d, lastBlock %v err %v, %v",
						feTxn.TxnID, rndCnt, i, firstAvailableReader, lastN, n, lastBlock, err, feTxn)
					for i := 0; i < scoder.n; i++ {
						scoder.shards[i] = scoder.dataBuf[scoder.blockSize*i : scoder.blockSize*(i+1)]
					}
					return nil
				}
				lastN = n
			} else {
				scoder.shards[i] = scoder.shards[i][:0]
			}
		}

		if err = scoder.Coder.ReconstructData(scoder.shards); err != nil {
			//if err = scoder.Coder.Reconstruct(scoder.shards); err != nil{
			HandleFailedReq(feTxn, fmt.Sprintf("ec decode err %v", err), "ecDecode", true)
			for i := 0; i < scoder.n; i++ {
				scoder.shards[i] = scoder.dataBuf[scoder.blockSize*i : scoder.blockSize*(i+1)]
			}
			return nil
		}

		for i := 0; i < scoder.k; i++ {
			if remainingSize > int64(n) {
				remainingSize -= int64(n)
			} else {
				// this should only happen on the last write
				n = int(remainingSize)
				remainingSize = 0
			}
			if remainingSize == 0 {
				// this is needed because when objSize % (blockSize*k) == 0
				// we won't get io.EOF in the read, thus we won't set lastBlock in the read
				if myconst.DebugLevel >= myconst.PerReqDetailedLogging {
					slogger.Debugf("txn %v: set last block because remaining size 0, i %v", feTxn.TxnID, i)
				}
				lastBlock = true
			}
			if _, err = feTxn.Pipew.Write(scoder.shards[i][:n]); err != nil {
				slogger.Errorf("txn %v: round %v, ec err in writing to client %v, err %v", feTxn.TxnID, rndCnt, fmt.Sprintf("%v", feTxn.Ctx.RemoteAddr(), err))
				HandleFailedReq(feTxn, fmt.Sprintf("write to client err %v", err), "clientWrite", true)
				metricErr.WithLabelValues("writeEcChunk").Inc()
				//slogger.DPanicf("txn %v: error write to pipe writer, %v, i %v, n %v, objSize %v, remainingSize %v, %v",
				//	feTxn.TxnID, err, i, n, feTxn.ObjSize, remainingSize, feTxn)
				for i := 0; i < scoder.n; i++ {
					scoder.shards[i] = scoder.dataBuf[scoder.blockSize*i : scoder.blockSize*(i+1)]
				}
				return nil
			}
			feTxn.SendSize += int64(n)
			if _, err := respBuffer.Write(scoder.shards[i][:n]); err != nil {
				slogger.DPanicf("txn %v: err writing to buf, %v", feTxn.TxnID, feTxn)
			}
			if remainingSize == 0 {
				break
			}
		}
	}

	if err = feTxn.Pipew.Close(); err != nil {
		slogger.DPanicf("txn %v: round %v, ec err in closing client %v connection %v", feTxn.TxnID, rndCnt, fmt.Sprintf("%v", feTxn.Ctx.RemoteAddr(), err))
	}

	for i := 0; i < scoder.n; i++ {
		scoder.shards[i] = scoder.dataBuf[scoder.blockSize*i : scoder.blockSize*(i+1)]
	}

	return respBuffer
}
