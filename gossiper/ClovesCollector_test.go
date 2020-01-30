package gossiper

import (
	"testing"

	"github.com/simonwicky/Peerster/utils"
)

func TestEmptySeqDoesNotMeetThreshold(t *testing.T) {
	cc := NewClovesCollector(nil)
	//cc.Add(&utils.Clove{Index: 1, Thresho})
	if ok, _, _ := cc.MeetsThreshold("jfnsakbf", 1); ok {
		t.Error("threshold should not be met")
	}
}

func TestThresholdIsMetForIndependentPaths(t *testing.T) {
	cc := NewClovesCollector(nil)
	cc.Add(&utils.Clove{Index: 1, Threshold: 3, SequenceNumber: []byte("abc")}, "Alice")
	cc.Add(&utils.Clove{Index: 2, Threshold: 3, SequenceNumber: []byte("abc")}, "Bob")
	cc.Add(&utils.Clove{Index: 3, Threshold: 3, SequenceNumber: []byte("abc")}, "Jack")
	if ok, _, _ := cc.MeetsThreshold("abc", 3); !ok {
		t.Error("threshold should be met")
	}
}

func TestThresholdIsNotMetForKClovesButLessPaths(t *testing.T) {
	cc := NewClovesCollector(nil)
	cc.Add(&utils.Clove{Index: 1, Threshold: 3, SequenceNumber: []byte("abc")}, "Alice")
	cc.Add(&utils.Clove{Index: 2, Threshold: 3, SequenceNumber: []byte("abc")}, "Alice")
	cc.Add(&utils.Clove{Index: 3, Threshold: 3, SequenceNumber: []byte("abc")}, "Jack")
	if ok, _, _ := cc.MeetsThreshold("abc", 3); ok {
		t.Error("threshold should not be met")
	}
}

func TestThresholdIsNotMetForKClovesButSameIndex(t *testing.T) {
	cc := NewClovesCollector(nil)
	cc.Add(&utils.Clove{Index: 2, Threshold: 3, SequenceNumber: []byte("abc")}, "Alice")
	cc.Add(&utils.Clove{Index: 2, Threshold: 3, SequenceNumber: []byte("abc")}, "Emily")
	cc.Add(&utils.Clove{Index: 2, Threshold: 3, SequenceNumber: []byte("abc")}, "Jack")
	if ok, _, _ := cc.MeetsThreshold("abc", 3); ok {
		t.Error("threshold should not be met")
	}
}
