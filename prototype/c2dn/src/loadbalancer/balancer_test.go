package loadbalancer

import (
	"fmt"
	"github.com/1a1a11a/c2dnPrototype/src/myconst"
	"testing"
)

/* this test is for 200 bucket, need to change for 100 buckets */

//func TestChBalaner(t *testing.T) {
//	if myconst.NBuckets != 200 {
//		log.Panic("test not updated yet")
//	}
//
//	bl := NewConsistentHashBalancer(10, "unavailability.one", 300, 4)
//	assert.Equal(t, bl.GetNode(0, "a"), "6")
//	assert.Equal(t, bl.GetNodes(0, "a", 2), []string{"6", "1"})
//	assert.Equal(t, bl.GetNodes(0, "a", 4), []string{"6", "1", "3", "8"})
//	assert.Equal(t, bl.GetNodes(0, "a", 6), []string{"6", "1", "3", "8", "9", "0"})
//
//	nodesNoUnavail := []string{"6", "1", "3", "8"}
//	nodesUnavail := []string{"1", "3", "8", "9"}
//	assert.Equal(t, bl.GetNodes(int64(12*300), "a", 4), nodesNoUnavail)
//	assert.Equal(t, bl.GetNodes(int64(24*300-1), "a", 4), nodesNoUnavail)
//	assert.Equal(t, bl.GetNodes(int64(24*300), "a", 4), nodesUnavail)
//	assert.Equal(t, bl.GetNodes(int64(36*300-1), "a", 4), nodesUnavail)
//	assert.Equal(t, bl.GetNodes(int64(36*300), "a", 4), nodesNoUnavail)
//
//	bl.Reset()
//	nodesNoUnavail2 := []int{9, 6, 0, 5}
//	nodesUnavail2 := []int{9, 0, 5, 7}
//	assert.Equal(t, bl.GetNodesFromMapping(0, 96), nodesNoUnavail2)
//	assert.Equal(t, bl.GetNodesFromMapping(12*300, 96), nodesNoUnavail2)
//	assert.Equal(t, bl.GetNodesFromMapping(24*300-1, 96), nodesNoUnavail2)
//	assert.Equal(t, bl.GetNodesFromMapping(24*300, 96), nodesUnavail2)
//	assert.Equal(t, bl.GetNodesFromMapping(36*300-1, 96), nodesUnavail2)
//	assert.Equal(t, bl.GetNodesFromMapping(36*300, 96), nodesNoUnavail2)
//}
//
//func TestChBalaner2(t *testing.T) {
//	if myconst.NBuckets != 200 {
//		log.Panic("test not updated yet")
//	}
//
//	bl := NewParityBalancer(10, "unavailability.one", 300, 4, true)
//
//	nodesNoUnavail := []int{9, 6, 0, 5, -1}
//	nodesUnavail := []int{9, 0, 5, 7, -1}
//
//	nodes := bl.GetNodesFromMapping(0, 96)
//	assert.Equal(t, nodes[:3], nodesNoUnavail[:3])
//	assert.Equal(t, nodes[4], -1)
//
//	nodes2 := bl.GetNodesFromMapping(24*300-1, 96)
//	assert.Equal(t, nodes2[:3], nodesNoUnavail[:3])
//	assert.Equal(t, nodes[3], nodes2[3])
//	assert.Equal(t, nodes2[4], -1)
//
//	nodes3 := bl.GetNodesFromMapping(24*300, 96)
//	assert.Equal(t, nodes3[:3], nodesUnavail[:3])
//	assert.Equal(t, nodes3[4], nodes2[3])
//
//	nodes4 := bl.GetNodesFromMapping(36*300-1, 96)
//	assert.Equal(t, nodes4[:3], nodesUnavail[:3])
//	assert.Equal(t, nodes4[3], nodes3[3])
//	assert.Equal(t, nodes4[4], nodes2[3])
//
//	nodes5 := bl.GetNodesFromMapping(36*300, 96)
//	assert.Equal(t, nodes5[:3], nodesNoUnavail[:3])
//	assert.Equal(t, nodes5[4], nodes4[3])
//
//	nodes6 := bl.GetNodesFromMapping(48*300, 96)
//	assert.Equal(t, nodes6[:3], nodesNoUnavail[:3])
//	assert.Equal(t, nodes6[3], nodes5[3])
//	assert.Equal(t, nodes6[4], nodes4[3])
//
//	// should not pass this, but good to pass this
//	assert.Equal(t, nodes[3], 5)
//	assert.Equal(t, nodes5[3], nodes[3])
//}

func TestChBalaner2(t *testing.T) {
	bl := NewParityBalancer(10, "unavailability.one", 300, 4, true)
	//bl.GetNodesFromMapping(0, 0)
	//bl.GetNodesFromMapping(30*300, 0)
	for bucket := 0; bucket < myconst.NBuckets; bucket++ {
		fmt.Println(bucket, bl.GetNodesFromMapping(0, bucket))
	}

	fmt.Println(bl.IsUnavailable(6))
	bl.GetNodesFromMapping(30*300, 0)
	fmt.Println(bl.IsUnavailable(6))
}

func TestChBalaner4(t *testing.T) {
	bl := NewParityBalancer(10, "unavailability.one", 300, 4, true)
	//bl.GetNodesFromMapping(0, 0)
	//bl.GetNodesFromMapping(30*300, 0)
	for bucket := 0; bucket < myconst.NBuckets; bucket++ {
		fmt.Println(bucket, bl.GetNodesFromMapping(0, bucket))
	}

	fmt.Println(bl.IsUnavailable(6))
	bl.GetNodesFromMapping(30*300, 0)
	fmt.Println(bl.IsUnavailable(6))
}
