package main

import (
	"bufio"
	"fmt"
	"github.com/1a1a11a/c2dnPrototype/src/loadbalancer"
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"go.uber.org/zap"
	"io"
	"os"
)


var (
	logger  *zap.Logger
	slogger *zap.SugaredLogger
)

var lb1 *loadbalancer.ChBalancer
var lb2 *loadbalancer.ParityBalancer


func findLoad(datPath string) {
	var err error
	file, err := os.Open(datPath)
	if err != nil {
		slogger.Fatal("failed to open trace file " + datPath)
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	req, err := readOneReq(reader)

	readLoad := make([]int64, 10)
	writeLoad := make([]int64, 10)
	//var seenObj []map[uint32]bool
	seenObj := make([]map[uint32]bool, 10)
	for i:=0; i<10; i++ {
		seenObj[i] = make(map[uint32]bool)
	}

	for {
		if err == nil {
			bucket := req.ID % myconst.NBuckets
			hosts := lb2.GetNodesFromMapping(0, int(bucket))
			for _, host := range hosts {
				if host == -1 {
					continue
				}
				if _, found := seenObj[host][req.ID]; !found {
					seenObj[host][req.ID] = true
					writeLoad[host] += int64(req.Size)
				} else {
					readLoad[host] += int64(req.Size)
				}
			}
			req, err = readOneReq(reader)

		} else if err == io.EOF {
			break
		} else {
			slogger.DPanic("trace read error ", err)
		}
	}

	for i:=0; i < 10; i++ {
		fmt.Println(readLoad[i], writeLoad[i])
	}
}

/**
  this find the read and write load of different systems under infinite size
 */
func findLoadBalance(mode string) {
	if mode == "no rep" {
		lb2 = loadbalancer.NewParityBalancer(10, "", -1, 1, false)
	} else if mode == "two rep" {
		lb2 = loadbalancer.NewParityBalancer(10, "", -1, 2, false)
	} else if mode == "naive coding" {
		lb2 = loadbalancer.NewParityBalancer(10, "", -1, 4, false)
	} else if mode == "Donut" {
		lb2 = loadbalancer.NewParityBalancer(10, "", -1, 4, true)
	} else {
		slogger.DPanicf("unknown mode")
	}
	findLoad("/disk1/CDN/akamai/video/akamai2.bin.scale10")
}




