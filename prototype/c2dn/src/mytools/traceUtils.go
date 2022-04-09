package mytools

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
)

func FindInTrace(datPath string) {
	var reqID uint64
	flag.Uint64Var(&reqID, "reqID", 0, "reqID")
	flag.Parse()
	if reqID == 0 {
		fmt.Println("please provide reqID")
		os.Exit(1)
	}

	req := main2.Request{}

	file, err := os.Open(datPath)
	if err != nil {
		log.Fatal("failed to open trace file " + datPath)
	}
	defer file.Close()

	err = binary.Read(file, binary.LittleEndian, &req)
	reqCount := 0

	for {
		if err == nil {
			if req.ID == uint32(reqID) {
				fmt.Println("req ", reqCount, ": ", req)
			}
		} else if err == io.EOF {
			break
		} else {
			log.Fatal("trace read error ", err)
		}
		reqCount++
		err = binary.Read(file, binary.LittleEndian, &req)
	}
}

func SplitTrace(datPath string, nHosts uint32) {

	log.Printf("Loading AkamaiWorkLoad %v\n", datPath)
	req := main2.Request{}

	ifile, err := os.Open(datPath)
	if err != nil {
		log.Fatal("failed to open trace file " + datPath)
	}
	defer ifile.Close()

	var ofiles []*os.File
	for i := 0; i < int(nHosts); i++ {
		ofile, err := os.Create(datPath + "." + strconv.Itoa(i))
		ofiles = append(ofiles, ofile)
		if err != nil {
			log.Fatal(err)
		}
		defer ofile.Close()
	}

	buf := new(bytes.Buffer)

	err = binary.Read(ifile, binary.LittleEndian, &req)

	for {
		if err == nil {
			hostIdx := (myutils.Hash(fmt.Sprintf("%d_%d", req.ID, req.Size)) + uint32(rand.Intn(2))) % nHosts
			err := binary.Write(buf, binary.LittleEndian, req)
			if err != nil {
				log.Fatal("binary Write failed:", err)
			}

			if _, err = ofiles[hostIdx].Write(buf.Bytes()); err != nil {
				log.Fatal("write to file failed")
			}
			buf.Reset()

		} else if err == io.EOF {
			break
		} else {
			log.Fatal("trace read error ", err)
		}
		err = binary.Read(ifile, binary.LittleEndian, &req)
	}
	log.Println("done")
}

// generate n traces for all local clients and one for remote client
func SplitTraceWithRemoteClient(datPath string, nHosts uint32, ratio uint32) {

	log.Printf("Loading AkamaiWorkLoad %v, nHosts %v, sample ratio %v\n", datPath, nHosts, ratio)
	req := main2.Request{}

	ifile, err := os.Open(datPath)
	if err != nil {
		log.Fatal("failed to open trace file " + datPath)
	}
	defer ifile.Close()

	var ofiles []*os.File
	var ofileSample *os.File
	ofileSample, err = os.Create(datPath + ".remote")

	for i := 0; i < int(nHosts); i++ {
		ofile, err := os.Create(datPath + "." + strconv.Itoa(i))
		ofiles = append(ofiles, ofile)
		if err != nil {
			log.Fatal(err)
		}
		defer ofile.Close()
	}

	buf := new(bytes.Buffer)
	err = binary.Read(ifile, binary.LittleEndian, &req)

	for {
		if err == nil {
			err := binary.Write(buf, binary.LittleEndian, req)
			if err != nil {
				log.Fatal("binary Write failed:", err)
			}
			// because in our hashing, we used hash(objID)%10 for nodeID, so we cannot use hash(objID)%100<ratio here
			if myutils.Hash(fmt.Sprintf("%d_%d", req.ID, req.Size))%10002 < ratio*100 {
				//if Hash(fmt.Sprintf("%d_%d", req.ID, req.Size))%100 < ratio {
				if _, err = ofileSample.Write(buf.Bytes()); err != nil {
					log.Fatal("write to file failed")
				}
			} else {
				hostIdx := (myutils.Hash(fmt.Sprintf("%d_%d", req.ID, req.Size)) + uint32(rand.Intn(2))) % nHosts
				if _, err = ofiles[hostIdx].Write(buf.Bytes()); err != nil {
					log.Fatal("write to file failed")
				}
			}
		} else if err == io.EOF {
			break
		} else {
			log.Fatal("trace read error ", err)
		}
		buf.Reset()
		err = binary.Read(ifile, binary.LittleEndian, &req)
	}
	log.Println("done")
}

func PrintHostIdx(datPath string) {

	req := main2.Request{}
	ifile, err := os.Open(datPath)
	if err != nil {
		log.Fatal("failed to open trace file " + datPath)
	}
	defer ifile.Close()

	buf := new(bytes.Buffer)
	err = binary.Read(ifile, binary.LittleEndian, &req)

	for {
		if err == nil {
			err := binary.Write(buf, binary.LittleEndian, req)
			if err != nil {
				log.Fatal("binary Write failed:", err)
			}

			var indexs []int
			firstHostID := int(myutils.HashByte([]byte(fmt.Sprintf("%v_%v", req.ID, req.Size)))) % 10
			for i := 0; i < 4; i++ {
				indexs = append(indexs, (firstHostID+i)%10)
			}

			fmt.Println(indexs)
		} else if err == io.EOF {
			break
		} else {
			log.Fatal("trace read error ", err)
		}
		buf.Reset()
		err = binary.Read(ifile, binary.LittleEndian, &req)
	}
	log.Println("done")

}

func TraceStat(datPath string, objSizeThreshold uint32, window int64) {
	ifile, err := os.Open(datPath)
	if err != nil {
		log.Fatal("failed to open trace file " + datPath)
	} else {
		log.Printf("Loading AkamaiWorkLoad %v\n", datPath)
	}
	defer ifile.Close()

	req := main2.Request{}
	err = binary.Read(ifile, binary.LittleEndian, &req)
	var traffic, traffic2 int64 = 0, 0
	var cnt2 = 0
	objMap := make(map[uint32]bool)
	var objBytes int64 = 0
	var reqCnt, objCnt int64 = 0, 0
	var thrptPerSec, thrptPerSecMax, thrptPerSecMaxTs int64 = 0, 0, 0
	var thrptPerWindow, thrptPerWindowMax, thrptPerWindowMaxTs int64 = 0, 0, 0
	var startTs, lastTs, lastThrptWindowCalTs int64 = int64(req.Timestamp), 0, int64(req.Timestamp)

	for {
		if err == nil {
			reqCnt += 1
			if !objMap[req.ID] {
				objBytes += int64(req.Size)
				objCnt += 1
				objMap[req.ID] = true
			}

			if req.Size >= objSizeThreshold {
				traffic2 += int64(req.Size)
				cnt2 += 1
			}
			traffic += int64(req.Size)
			if int64(req.Timestamp) != lastTs {
				if thrptPerSec > thrptPerSecMax {
					thrptPerSecMax = thrptPerSec
					thrptPerSecMaxTs = lastTs
				}
				lastTs = int64(req.Timestamp)
				//fmt.Println(thrptPerSec)
				thrptPerSec = 0
			}

			if int64(req.Timestamp)-lastThrptWindowCalTs >= window {
				if thrptPerWindow > thrptPerWindowMax {
					thrptPerWindowMax = thrptPerWindow
					thrptPerWindowMaxTs = lastThrptWindowCalTs
				}
				lastThrptWindowCalTs = int64(req.Timestamp)
				thrptPerWindow = 0
			}

			thrptPerSec += int64(req.Size)
			thrptPerWindow += int64(req.Size)
		} else if err == io.EOF {
			break
		} else {
			log.Fatal("trace read error ", err)
		}
		err = binary.Read(ifile, binary.LittleEndian, &req)
	}
	log.Printf("tracetime %v s, %v req, %v obj, %v Bytes, %v GB, "+
		"largeObj traffic %v Bytes, %v GB, "+
		"throughputPerSecMax %.2f (at %vs) Gbps, "+
		"throughputPerWindowMax (window %vs) %.2f (at %v s) Gbps, "+
		"uniqueObj bytes %v, %v GB\n",
		int64(req.Timestamp)-startTs, reqCnt, objCnt,
		traffic, traffic/(1000*1000*1000),
		traffic2, traffic2/(1000*1000*1000),
		float64(thrptPerSecMax)*8/(1000*1000*1000), thrptPerSecMaxTs-startTs,
		window, float64(thrptPerWindowMax)*8/(1000*1000*1000)/float64(window), thrptPerWindowMaxTs-startTs,
		objBytes, objBytes/(1000*1000*1000),
	)
	//log.Println(traffic2, traffic2/1000/1000/1000, cnt2)
}

func FindHost(datPath string) {

	ifile, err := os.Open(datPath)
	if err != nil {
		log.Fatal("failed to open trace file " + datPath)
	} else {
		log.Printf("Loading AkamaiWorkLoad %v\n", datPath)
	}
	defer ifile.Close()

	req := main2.Request{}
	err = binary.Read(ifile, binary.LittleEndian, &req)

	for {
		if err == nil {
			hostIdx := myutils.Hash(fmt.Sprintf("%d_%d", req.ID, req.Size)) % 10
			//if Hash(fmt.Sprintf("%d_%d", req.ID, req.Size))%10001 < 2*100 {
			if myutils.Hash(fmt.Sprintf("%d_%d", req.ID, req.Size))%100 < 2 {
				fmt.Println(req, hostIdx, hostIdx+1)
			} else {
				fmt.Println("error ")
			}
		} else if err == io.EOF {
			break
		} else {
			log.Fatal("trace read error ", err)
		}
		err = binary.Read(ifile, binary.LittleEndian, &req)
	}
}

func PrintTrace(datPath string) {
	ifile, err := os.Open(datPath)
	if err != nil {
		log.Fatal("failed to open trace file " + datPath)
	} else {
		log.Printf("Loading AkamaiWorkLoad %v\n", datPath)
	}
	defer ifile.Close()

	req := main2.Request{}
	err = binary.Read(ifile, binary.LittleEndian, &req)

	for {
		if err == nil {
			fmt.Printf("%v %v\n", req.ID, req.Size)
		} else if err == io.EOF {
			break
		} else {
			log.Fatal("trace read error ", err)
		}
		err = binary.Read(ifile, binary.LittleEndian, &req)
	}
}

// generate traces using the new binary traces with nodeID
// if no remote client needed, set remoteClientRatio = 0,
// otherwise, set to x denoting x% total traffic will be sampled to remote client
func GenTraceFromNodeIDTrace(datPath string, clusterSize uint32, remoteClientRatio uint32, useNodeID bool) {
	log.Printf("Loading AkamaiWorkLoad %v, clusterSize %v, sample remoteClientRatio %v\n",
		datPath, clusterSize, remoteClientRatio)
	req := main2.Request{}
	reqWithNode := main2.RequestWithNodeID{}
	var hostIdx int

	ifile, err := os.Open(datPath)
	if err != nil {
		log.Fatal("failed to open trace file " + datPath)
	}
	defer ifile.Close()

	var ofiles []*os.File
	var ofileRemoteClient *os.File
	ofileRemoteClient, err = os.Create(datPath + ".remote")

	for i := 0; i < int(clusterSize); i++ {
		ofile, err := os.Create(datPath + "." + strconv.Itoa(i))
		ofiles = append(ofiles, ofile)
		if err != nil {
			log.Fatal(err)
		}
		defer ofile.Close()
	}

	buf := new(bytes.Buffer)
	err = binary.Read(ifile, binary.LittleEndian, &reqWithNode)

	for {
		req.ID = reqWithNode.ID
		req.Timestamp = reqWithNode.Timestamp
		req.Size = reqWithNode.Size

		if err == nil {
			err := binary.Write(buf, binary.LittleEndian, req)
			if err != nil {
				log.Fatal("binary write failed:", err)
			}
			// because in our hashing, we used hash(objID)%10 for nodeID, so we cannot use hash(objID)%100<ratio here
			if myutils.Hash(fmt.Sprintf("%d_%d", req.ID, req.Size))%10002 < remoteClientRatio*100 {
				if _, err = ofileRemoteClient.Write(buf.Bytes()); err != nil {
					log.Fatal("write to file failed")
				}
			} else {
				if useNodeID {
					hostIdx = int(reqWithNode.NodeID)
				} else {
					hostIdx = int((myutils.Hash(fmt.Sprintf("%d_%d", req.ID, req.Size)) + uint32(rand.Intn(2))) % clusterSize)
				}
				//noinspection GoNilness
				if _, err = ofiles[hostIdx].Write(buf.Bytes()); err != nil {
					log.Fatal("write to file failed")
				}
			}
		} else if err == io.EOF {
			break
		} else {
			log.Fatal("trace read error ", err)
		}
		buf.Reset()
		err = binary.Read(ifile, binary.LittleEndian, &reqWithNode)
	}
	log.Println("done")
}

//func main(){
//	TraceStat("/home/jason/data/akamai/video/akamai2.bin", 128*1024, 60)
//	TraceStat("/home/jason/data/akamai/nodeID/akamai.bin", 128*1024, 60)
//}
