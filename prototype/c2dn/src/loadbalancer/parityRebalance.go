package loadbalancer

import (
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"github.com/1a1a11a/graph"
	//"github.com/yourbasic/graph"
	"math"
	"strconv"
)

type ParityBalancer struct {
	ChBalancer

	rebalance         bool
	currParityMapping map[int]int
	prevParityMapping map[int]int
}

func (bl *ParityBalancer) rebalanceParity() {
	nHost, nBucket := len(bl.hosts), myconst.NBuckets

	nodeWeight, bucketUsedHost := calNodeWeight(bl.bucketMapping, nHost, nBucket, bl.nHostPerBucket-1)
	slogger.Infof("weight before balancing %v", nodeWeight)

	nAvailableHost := nHost
	if len(bl.unavailableList) > 0 {
		nAvailableHost -= len(bl.unavailableList[bl.currUnavailPos])
	}
	expectedWeight := int(math.Ceil(float64(bl.nHostPerBucket*nBucket) / float64(nAvailableHost)))

	gm := graph.New(nHost + nBucket + 2)
	for host := 0; host < nHost; host++ {
		/* sink to server */
		gm.AddCost(0, host+2, int64(expectedWeight-nodeWeight[host]))
	}
	for bucket := 0; bucket < nBucket; bucket++ {
		/* bucket to sink */
		gm.AddCost(bucket+nHost+2, 1, 1)
		/* bucket to server */
		for host := 0; host < nHost; host++ {
			if _, found := bucketUsedHost[bucket][host]; found {
				continue
			}
			gm.AddCost(host+2, bucket+nHost+2, 1)
		}
	}

	if len(bl.unavailableList) > 0 {
		for _, host := range bl.unavailableList[bl.currUnavailPos] {
			gm.Delete(0, host+2)
		}
	}

	flow, iter := graph.MaxFlow(gm, 0, 1)
	if flow != int64(nBucket) {
		slogger.DPanicf("max flow does not reach nBucket %d != %d", flow, nBucket)
	}

	parityMapping := make(map[int]int)
	graph.BFS(iter, 0, func(left, right int, c int64) {
		if left != 0 && right != 1 {
			//fmt.Println(left, "to", right, "c", c)
			bucketID := right - nHost - 2
			hostID := left - 2
			parityMapping[bucketID] = hostID
			bl.bucketMapping[bucketID][bl.nHostPerBucket-1] = hostID
		} else if left == 0 {
			hostID := right - 2
			nodeWeight[hostID] += int(c)
		}
	})
	bl.prevParityMapping = bl.currParityMapping
	bl.currParityMapping = parityMapping

	slogger.Infof("weight after balancing %v", nodeWeight)
}

func NewParityBalancer(nHost int, unavailFilePath string, updateIntvl int64, nHostPerBucket int, rebalance bool) *ParityBalancer {
	bl := &ParityBalancer{}

	var hosts []string
	for i := 0; i < nHost; i++ {
		hosts = append(hosts, strconv.Itoa(i))
	}

	bl.hosts = hosts
	bl.updateIntvl = updateIntvl
	bl.unavailFilePath = unavailFilePath
	bl.startTs = -1
	bl.currUnavailPos = 0
	bl.rebalance = rebalance
	bl.nHostPerBucket = nHostPerBucket
	bl.currParityMapping = nil

	bl.ring = NewRing(bl.hosts)

	if unavailFilePath == "" {
		if updateIntvl == -1 {
			bl.unavailableList = nil
		} else {
			slogger.DPanic("empty unavailable trace with update interval %d", updateIntvl)
		}
	} else {
		bl.unavailableList = parseUnavailTrace(unavailFilePath)
	}

	bl.createMapping()
	return bl
}

func (bl *ParityBalancer) GetNode(currTs int64, id string) string {
	slogger.DPanic("does not support")
	return ""
}

func (bl *ParityBalancer) GetNodes(currTs int64, id string, n int) []string {
	slogger.DPanic("does not support")
	return []string{""}
}

func (bl *ParityBalancer) GetNodesFromMapping(currTs int64, id int) []int {
	bl.checkUpdate(currTs)
	return bl.bucketMapping[id]
}

func (bl *ParityBalancer) checkUpdate(currTs int64) {
	if bl.updateIntvl < 0 {
		return
	}

	if bl.startTs == -1 {
		bl.startTs = currTs
		bl.currUnavailPos = 0
	}

	if (currTs-bl.startTs)/bl.updateIntvl > bl.currUnavailPos {
		bl.update(currTs)
	}
}

func (bl *ParityBalancer) update(currTs int64) {
	newUnavailPos := (currTs - bl.startTs) / bl.updateIntvl

	if !cmpSlice(bl.unavailableList[newUnavailPos], bl.unavailableList[bl.currUnavailPos]) {
		slogger.Infof("time %v, pos %v, update load balancer with unavailabilities %v",
			currTs, newUnavailPos, bl.unavailableList[newUnavailPos])
		newRing := NewRing(bl.hosts)
		for _, nodeIdx := range bl.unavailableList[newUnavailPos] {
			newRing = newRing.RemoveNode(bl.hosts[nodeIdx])
		}

		bl.updateMtx.Lock()
		bl.ring = newRing
		bl.updateMtx.Unlock()

		bl.currUnavailPos = newUnavailPos
		bl.createMapping()
	} else {
		bl.currUnavailPos = newUnavailPos
		slogger.Debugf("time %v, pos %v, load balancer no update",
			currTs, newUnavailPos)
	}
}

func (bl *ParityBalancer) createMapping() {
	/* create consistent hash mapping */
	bl.bucketMapping = make(map[int][]int)
	for bucket := 0; bucket < myconst.NBuckets; bucket++ {
		mappedNodesInt := make([]int, bl.nHostPerBucket+1)
		nodes, ok := bl.ring.GetNodes(strconv.Itoa(bucket), bl.nHostPerBucket)
		if !ok {
			slogger.DPanicf("failed to get nodes %v", nodes)
		}
		for i, node := range nodes {
			nodeInt, _ := strconv.Atoi(node)
			mappedNodesInt[i] = nodeInt
		}
		mappedNodesInt[bl.nHostPerBucket] = -1
		bl.bucketMapping[bucket] = mappedNodesInt
	}

	if bl.rebalance {
		/* now rebalance parity */
		bl.rebalanceParity()

		/* now add previous mapping to end bucket mapping */
		if bl.prevParityMapping != nil {
			for bucket, host := range bl.prevParityMapping {
				bl.bucketMapping[bucket][bl.nHostPerBucket] = host
			}
		}
	}
}
