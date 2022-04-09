package loadbalancer

import (
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/1a1a11a/c2dnPrototype/src/myutils"
	"go.uber.org/zap"
	"io/ioutil"
	"strconv"
	"strings"
	"sync"
)

var (
	logger  *zap.Logger
	slogger *zap.SugaredLogger
)

func init() {
	logger, slogger = myutils.InitLogger("misc", myconst.DebugLevel)
}

type baseBalancer struct {
	unavailableList [][]int

	hosts          []string
	bucketMapping  map[int][]int
	nHostPerBucket int

	unavailFilePath string
	updateMtx       sync.RWMutex
	updateIntvl     int64
	startTs         int64
	currUnavailPos  int64
}

type Balancer interface {
	GetNodesFromMapping(currTs int64, id string) (nodes []int)
}

func parseUnavailTrace(unavailFilePath string) [][]int {
	var unavailableList [][]int
	dat, err := ioutil.ReadFile(unavailFilePath)
	if err != nil {
		slogger.Panic(err)
	}
	nUnavailability := 0
	unavailEvents := strings.Split(string(dat), "\n")
	for _, g := range unavailEvents {
		var unavailTs []int
		if len(g) != 0 {
			unavailableList := strings.Split(g, " ")
			nUnavailability += len(unavailableList)

			for _, node := range unavailableList {
				nodeInt, _ := strconv.Atoi(node)
				unavailTs = append(unavailTs, nodeInt)
			}
		}
		unavailableList = append(unavailableList, unavailTs)
	}
	meanUnavail := float64(nUnavailability) / float64(len(unavailableList))
	slogger.Infof("unavailability trace %s has %d time intervals, "+
		"%.2f mean unavailability", unavailFilePath, len(unavailableList), meanUnavail)

	return unavailableList
}

func cmpSlice(s1 []int, s2 []int) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := 0; i < len(s1); i++ {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

func (bl *baseBalancer) IsUnavailable(server int) bool {
	if len(bl.unavailableList) == 0 || len(bl.unavailableList[bl.currUnavailPos]) == 0 {
		return false
	}
	for _, unavailableServer := range bl.unavailableList[bl.currUnavailPos] {
		if server == unavailableServer {
			return true
		}
	}
	return false
}

func calNodeWeight(bucketMapping map[int][]int, nHost int, nBucket int, n int) ([]int, map[int]map[int]bool) {
	nodeWeight := make([]int, nHost)
	bucketUsedHost := make(map[int]map[int]bool)
	for bucket := 0; bucket < nBucket; bucket++ {
		nodes := bucketMapping[bucket]
		for i := 0; i < n; i++ {
			nodeWeight[nodes[i]] += 1
			if _, found := bucketUsedHost[bucket]; !found {
				bucketUsedHost[bucket] = make(map[int]bool)
			}
			bucketUsedHost[bucket][nodes[i]] = true
		}
	}
	return nodeWeight, bucketUsedHost
}
