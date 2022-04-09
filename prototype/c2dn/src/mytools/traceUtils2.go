package mytools

import (
	"bytes"
	"encoding/binary"
	"fmt"
	main2 "github.com/1a1a11a/c2dnPrototype/cmd/client"
	"github.com/1a1a11a/c2dnPrototype/src/loadbalancer"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

func SplitUsingConsistentHash(datPath string, nServers uint32, remoteClientRatio uint32, useNodeID bool, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	log.Printf("Loading AkamaiWorkLoad %v, nServers %v, sample remoteClientRatio %v\n",
		datPath, nServers, remoteClientRatio)
	ring := loadbalancer.NewConsistentHashBalancer([]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}, "", -1)
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
	if remoteClientRatio > 0 {
		ofileRemoteClient, err = os.Create(datPath + ".remote")
	}

	for i := 0; i < int(nServers); i++ {
		ofile, err := os.Create(datPath + "." + strconv.Itoa(i))
		ofiles = append(ofiles, ofile)
		if err != nil {
			log.Fatal(err)
		}
		defer ofile.Close()
	}

	buf := new(bytes.Buffer)
	if useNodeID {
		err = binary.Read(ifile, binary.LittleEndian, &reqWithNode)
	} else {
		err = binary.Read(ifile, binary.LittleEndian, &req)
	}

	nIgnored := 0
	lastSeenTs := make(map[uint32]uint32)
	lastSeenTs[req.ID] = req.Timestamp

	for {
		if useNodeID {
			req.ID = reqWithNode.ID
			req.Timestamp = reqWithNode.Timestamp
			req.Size = reqWithNode.Size
		}

		// fix for akamai1 dataset
		if req.Size < 11 {
			req.Size = 11
		}

		if err == nil {
			err := binary.Write(buf, binary.LittleEndian, req)
			if err != nil {
				log.Fatal("binary write failed:", err)
			}
			var hashKey = fmt.Sprintf("%d_%d", req.ID, req.Size)
			if myutils.Hash(hashKey)%10002 < remoteClientRatio*100 {
				if _, err = ofileRemoteClient.Write(buf.Bytes()); err != nil {
					log.Fatal("write to file failed")
				}
			} else {
				if useNodeID {
					hostIdx = int(reqWithNode.NodeID)
				} else {
					hostIdxs := ring.GetNodes(0, hashKey, 2)
					hostIdx, _ = strconv.Atoi(hostIdxs[rand.Intn(2)])
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
		if useNodeID {
			err = binary.Read(ifile, binary.LittleEndian, &reqWithNode)
		} else {
			err = binary.Read(ifile, binary.LittleEndian, &req)
		}

		// this is for preprocessing akamai1 trace, this big req should not happen
		for req.Size > 100*1000*1000 && req.Timestamp-lastSeenTs[req.ID] < 3 {
			if useNodeID {
				err = binary.Read(ifile, binary.LittleEndian, &reqWithNode)
				req.ID = reqWithNode.ID
				req.Timestamp = reqWithNode.Timestamp
				req.Size = reqWithNode.Size
			} else {
				err = binary.Read(ifile, binary.LittleEndian, &req)
			}
			nIgnored++
		}
		lastSeenTs[req.ID] = req.Timestamp
	}
	log.Println("done ", nIgnored, " ignored")
}

func FilterNonUniqReq(datPath string, useBucket bool, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	ofilename := strings.Replace(datPath, "warmup.", "warmup.disk.", 1)
	ifile, err := os.Open(datPath)
	if err != nil {
		log.Fatal("failed to open trace file " + datPath)
	} else {
		log.Printf("Loading AkamaiWorkLoad %v, ofile %v\n", datPath, ofilename)
	}
	defer ifile.Close()

	req := main2.Request{}
	reqWithBucket := main2.RequestWithBucket{}

	var ofile *os.File
	ofile, err = os.Create(ofilename)
	if err != nil {
		log.Fatal(err)
	}
	defer ofile.Close()

	seenMap := make(map[string]bool)
	objCnt := 0
	wss := 0

	buf := &bytes.Buffer{}
	if useBucket {
		err = binary.Read(ifile, binary.LittleEndian, &reqWithBucket)
		req.ID = reqWithBucket.ID
		req.Size = reqWithBucket.Size
		req.Timestamp = reqWithBucket.Timestamp
	} else {
		err = binary.Read(ifile, binary.LittleEndian, &req)
	}

	for {
		if err == nil {
			s := fmt.Sprintf("%v_%v", req.ID, req.Size)
			if seenMap[s] {
				if useBucket {
					err = binary.Read(ifile, binary.LittleEndian, &reqWithBucket)
					req.ID = reqWithBucket.ID
					req.Size = reqWithBucket.Size
					req.Timestamp = reqWithBucket.Timestamp
				} else {
					err = binary.Read(ifile, binary.LittleEndian, &req)
				}
				continue
			} else {
				objCnt++
				wss += int(req.Size)
				seenMap[s] = true
				if useBucket {
					err = binary.Write(buf, binary.LittleEndian, reqWithBucket)
				} else {
					err = binary.Write(buf, binary.LittleEndian, req)
				}
				if err != nil {
					log.Fatal("binary Write failed:", err)
				}

				if _, err = ofile.Write(buf.Bytes()); err != nil {
					log.Fatal("write to file failed")
				}
			}
		} else if err == io.EOF {
			break
		} else {
			log.Fatal("trace read error ", err)
		}
		buf.Reset()
		if useBucket {
			err = binary.Read(ifile, binary.LittleEndian, &reqWithBucket)
			req.ID = reqWithBucket.ID
			req.Size = reqWithBucket.Size
			req.Timestamp = reqWithBucket.Timestamp
		} else {
			err = binary.Read(ifile, binary.LittleEndian, &req)
		}
	}
	log.Println(datPath, objCnt, " obj", wss/1000/1000/1000, " GB")
}

func SplitWithBucketUsingConsistentHash(datPath string, clusterSize uint32, remoteClientRatio uint32, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	log.Printf("Loading AkamaiWorkLoad %v, clusterSize %v, sample remoteClientRatio %v\n",
		datPath, clusterSize, remoteClientRatio)
	ring := loadbalancer.NewConsistentHashBalancer([]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}, "", -1)
	req := main2.RequestWithBucket{}

	var hostIdx int
	//var traffic int64 = 0
	//var wss int64 = 0

	ifile, err := os.Open(datPath)
	if err != nil {
		log.Fatal("failed to open trace file " + datPath)
	}
	defer ifile.Close()

	var ofiles []*os.File
	var ofileRemoteClient *os.File
	if remoteClientRatio > 0 {
		ofileRemoteClient, err = os.Create(datPath + ".remote")
	}

	for i := 0; i < int(clusterSize); i++ {
		ofile, err := os.Create(datPath + "." + strconv.Itoa(i))
		ofiles = append(ofiles, ofile)
		if err != nil {
			log.Fatal(err)
		}
		defer ofile.Close()
	}

	buf := new(bytes.Buffer)
	err = binary.Read(ifile, binary.LittleEndian, &req)

	nIgnored := 0
	lastSeenTs := make(map[uint32]uint32)
	lastSeenTs[req.ID] = req.Timestamp

	for {
		// fix for akamai1 dataset
		if req.Size < 11 {
			req.Size = 11
		}

		if err == nil {
			err := binary.Write(buf, binary.LittleEndian, req)
			if err != nil {
				log.Fatal("binary write failed:", err)
			}
			if req.Bucket > 100 {
				log.Fatalf("bucket %v > 100", req.Bucket)
			}
			hashKey := fmt.Sprintf("%d", req.Bucket)
			objHashKey := fmt.Sprintf("%d_%d", req.ID, req.Size)
			if myutils.Hash(objHashKey)%10002 < remoteClientRatio*100 {
				if _, err = ofileRemoteClient.Write(buf.Bytes()); err != nil {
					log.Fatal("write to file failed")
				}
			} else {
				hostIdxs := ring.GetNodes(0, hashKey, 2)
				hostIdx, _ = strconv.Atoi(hostIdxs[rand.Intn(2)])
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
		err = binary.Read(ifile, binary.LittleEndian, &req)

		// this is for preprocessing akamai1 trace, this big req should not happen
		for req.Size > 100*1000*1000 && req.Timestamp-lastSeenTs[req.ID] < 3 {
			err = binary.Read(ifile, binary.LittleEndian, &req)
			nIgnored++
		}
		lastSeenTs[req.ID] = req.Timestamp
	}
	log.Println("done ", nIgnored, " ignored")
}

func SampleTrace(datPath string, ratio uint32, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	splitPath := strings.Split(datPath, "/")
	splitPath = append(splitPath, splitPath[len(splitPath)-1])
	splitPath[len(splitPath)-2] = "sample"
	ofilename := strings.Join(splitPath, "/")
	ifile, err := os.Open(datPath)
	if err != nil {
		log.Fatal("failed to open trace file " + datPath)
	} else {
		//log.Printf("Loading AkamaiWorkLoad %v, ofile %v\n", datPath, ofilename)
	}
	defer ifile.Close()

	req := main2.RequestWithBucket{}
	var ofile *os.File
	ofile, err = os.Create(ofilename)
	if err != nil {
		log.Fatal(err)
	}
	defer ofile.Close()

	buf := &bytes.Buffer{}
	err = binary.Read(ifile, binary.LittleEndian, &req)
	seenmap := make(map[uint32]bool)
	reqCnt, objCnt := 0, 0
	startTs := req.Timestamp

	for {
		if err == nil {
			req.Timestamp = (req.Timestamp - startTs) / ratio
			err := binary.Write(buf, binary.LittleEndian, req)
			if myutils.Hash(fmt.Sprintf("%d", req.ID))%ratio < 1 {
				if _, err = ofile.Write(buf.Bytes()); err != nil {
					log.Fatal("write to file failed")
				}
				reqCnt++
				if !seenmap[req.ID] {
					seenmap[req.ID] = true
					objCnt++
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
	log.Println(ofilename, "before sampling ", reqCnt, " req, ", objCnt, " obj, new time span ", req.Timestamp)
}

func getNewObj(smallDat, largeDat, ofilename string, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	ifile1, err := os.Open(smallDat)
	if err != nil {
		log.Fatal("failed to open trace file " + smallDat)
	}
	defer ifile1.Close()
	ifile2, err := os.Open(largeDat)
	if err != nil {
		log.Fatal("failed to open trace file " + largeDat)
	}
	defer ifile2.Close()

	req := main2.Request{}
	var ofile *os.File
	ofile, err = os.Create(ofilename)
	if err != nil {
		log.Fatal(err)
	}
	defer ofile.Close()

	buf := &bytes.Buffer{}
	seenmap := make(map[uint32]bool)
	var wss, objCnt int64 = 0, 0

	for {
		err = binary.Read(ifile1, binary.LittleEndian, &req)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				log.Fatal(err)
			}
		}
		seenmap[req.ID] = true
	}

	err = binary.Read(ifile2, binary.LittleEndian, &req)

	for {
		if err == nil {
			if !seenmap[req.ID] {
				_ = binary.Write(buf, binary.LittleEndian, req)
				objCnt++
				wss += int64(req.Size)
			}
		} else if err == io.EOF {
			break
		} else {
			log.Fatal("trace read error ", err)
		}
		buf.Reset()
		err = binary.Read(ifile2, binary.LittleEndian, &req)
	}
	log.Println(ofilename, objCnt, " obj, ", wss, " bytes")

}

func mytest1() {
	//var hostIdxs []string
	ring := loadbalancer.NewConsistentHashBalancer([]string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}, "", -1)

	//hostIdxs = ring.GetNodes(0, "21402_1048576", 4)
	for i := 0; i < 100; i++ {
		hosts := ring.GetNodes(0, strconv.Itoa(i+0), 4)
		if hosts[0] == "8" {
			fmt.Println(i, hosts)
		}
	}

	//fmt.Println(myutils.Hash("394479_11") % 10002)
	fmt.Println(ring.GetNodes(0, "394479_11", 4))
	fmt.Println(ring.GetNodes(0, "0", 4))
}

func main() {
	wg := &sync.WaitGroup{}
	//go SplitUsingConsistentHash("/home/jason/data/akamai/C2DN/prototype/akamai2/30_10_bucket/warmup.bin", 10, 0, false, wg)
	//go SplitUsingConsistentHash("/home/jason/data/akamai/C2DN/prototype/akamai2/30_10_bucket/eval.bin", 10, 1, false, wg)

	//go SplitWithBucketUsingConsistentHash("/home/jason/data/akamai/C2DN/prototype/akamai1/300_300_bucket/warmup.bin", 10, 0, wg)
	//go SplitWithBucketUsingConsistentHash("/home/jason/data/akamai/C2DN/prototype/akamai1/300_300_bucket/eval.bin", 10, 1, wg)

	go SplitWithBucketUsingConsistentHash("/home/jason/data/CDN/akamai/video/macrobenchmark/sample30/akamai.bucket100.bin.warmup", 10, 0, wg)
	go SplitWithBucketUsingConsistentHash("/home/jason/data/CDN/akamai/video/macrobenchmark/sample30/akamai.bucket100.bin.eval", 10, 1, wg)

	for i := 0; i < 10; i++ {
		//go FilterNonUniqReq("/home/jason/data/akamai/C2DN/prototype/akamai1/300_100_bucket/warmup/warmup.bin."+strconv.Itoa(i), true, wg)
		//go SampleTrace("/home/jason/data/akamai/C2DN/prototype/akamai2/30_10_bucket/warmup/warmup.bin."+strconv.Itoa(i), 10, wg)
		//go SampleTrace("/home/jason/data/akamai/C2DN/prototype/akamai2/30_10_bucket/warmupDisk/warmup.bin."+strconv.Itoa(i), 10, wg)
		//go SampleTrace("/home/jason/data/akamai/C2DN/prototype/akamai2/30_10_bucket/warmupRAM/warmup.bin."+strconv.Itoa(i), 10, wg)
		//go SampleTrace("/home/jason/data/akamai/C2DN/prototype/akamai2/30_10_bucket/eval/eval.bin."+strconv.Itoa(i), 10, wg)
		//getNewObj(
		//	"/home/jason/data/akamai/C2DN/prototype/akamai2/20_10/warmupDisk/warmup.bin."+strconv.Itoa(i),
		//	"/home/jason/data/akamai/C2DN/prototype/akamai2/30_10/warmupDisk/warmup.bin."+strconv.Itoa(i),
		//	"/home/jason/data/akamai/C2DN/prototype/akamai2/diff/warmup.bin."+strconv.Itoa(i),
		//	wg,
		//	)
	}

	//go SampleTrace("/home/jason/data/CDN/akamai/video/macrobenchmark/akamai.bucket100.bin.macro", 30, wg)

	time.Sleep(8 * time.Second)
	wg.Wait()
}
