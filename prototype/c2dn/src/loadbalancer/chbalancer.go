package loadbalancer

import (
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"strconv"
)

type ChBalancer struct {
	baseBalancer

	ring *HashRing
}

func NewConsistentHashBalancer(nHost int, unavailFilePath string, updateIntvl int64, nHostPerBucket int) *ChBalancer {
	bl := &ChBalancer{}

	var hosts []string
	for i := 0; i < nHost; i++ {
		hosts = append(hosts, strconv.Itoa(i))
	}

	bl.hosts = hosts
	bl.updateIntvl = updateIntvl
	bl.unavailFilePath = unavailFilePath
	bl.startTs = -1
	bl.currUnavailPos = 0
	bl.nHostPerBucket = nHostPerBucket

	slogger.Infof("unavailability trace \"%v\", update interval %v", unavailFilePath, updateIntvl)
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

func (bl *ChBalancer) Reset() {
	bl.currUnavailPos = 0
	bl.startTs = -1
	slogger.Infof("consistent hash loadbalancer reset")
}

func (bl *ChBalancer) GetNode(currTs int64, id string) string {
	bl.checkUpdate(currTs)

	bl.updateMtx.RLock()
	node, ok := bl.ring.GetNode(id)
	bl.updateMtx.RUnlock()
	if !ok {
		slogger.DPanic("failed to get node")
	}
	return node
}

func (bl *ChBalancer) GetNodes(currTs int64, id string, n int) []string {
	bl.checkUpdate(currTs)

	bl.updateMtx.RLock()
	nodes, ok := bl.ring.GetNodes(id, n)
	bl.updateMtx.RUnlock()
	if !ok {
		slogger.DPanic("failed to get nodes, %v %v", nodes, ok)
	}
	return nodes
}

func (bl *ChBalancer) createMapping() {
	bl.bucketMapping = make(map[int][]int)
	for bucket := 0; bucket < myconst.NBuckets; bucket++ {
		mappedNodesInt := make([]int, bl.nHostPerBucket)
		nodes, ok := bl.ring.GetNodes(strconv.Itoa(bucket), bl.nHostPerBucket)
		if !ok {
			slogger.DPanicf("failed to get nodes %v", nodes)
		}
		for i, node := range nodes {
			nodeInt, _ := strconv.Atoi(node)
			mappedNodesInt[i] = nodeInt
		}
		bl.bucketMapping[bucket] = mappedNodesInt
	}
	//nodeWeight, _ := calNodeWeight(bl.bucketMapping, len(bl.hosts), myconst.NBuckets, 2)
	//slogger.Debug("node weight %v", nodeWeight)
}

func (bl *ChBalancer) GetNodesFromMapping(currTs int64, id int) []int {
	bl.checkUpdate(currTs)
	return bl.bucketMapping[id]
}

func (bl *ChBalancer) checkUpdate(currTs int64) {
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

func (bl *ChBalancer) update(currTs int64) {
	bl.currUnavailPos = (currTs - bl.startTs) / bl.updateIntvl

	if !cmpSlice(bl.unavailableList[bl.currUnavailPos], bl.unavailableList[bl.currUnavailPos-1]) {
		slogger.Infof("time %v, pos %v, update load balancer with unavailabilities %v",
			currTs, bl.currUnavailPos, bl.unavailableList[bl.currUnavailPos])
		newRing := NewRing(bl.hosts)
		for _, nodeIdx := range bl.unavailableList[bl.currUnavailPos] {
			newRing = newRing.RemoveNode(bl.hosts[nodeIdx])
		}

		bl.updateMtx.Lock()
		bl.ring = newRing
		bl.updateMtx.Unlock()

		bl.createMapping()
	} else {
		slogger.Debugf("time %v, pos %v, load balancer no update",
			currTs, bl.currUnavailPos)
	}
}
