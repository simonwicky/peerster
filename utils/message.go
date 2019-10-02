package utils

type SimpleMessage struct {
	OriginalName string
	RelayPeerAddr string
	Contents string
}

type RumorMessage struct {
	Origin string
	ID	uint32
	Text string
}
type RumorMessageKey struct {
	Origin string
	ID uint32
}

type PeerStatus struct {
	Identifer string
	NextID uint32
}

type StatusPacket struct {
	Want []PeerStatus
}

type GossipPacket struct {
	Simple *SimpleMessage
	Rumor *RumorMessage
	Status *StatusPacket
}

func CopyGossipPacket(packet *GossipPacket) *GossipPacket {
	var simple *SimpleMessage
	if packet.Simple != nil {
		simple = &(*packet.Simple)
	} else {
		simple = nil
	}
	var rumor *RumorMessage
	if packet.Rumor != nil {
		rumor = &(*packet.Rumor)
	} else {
		rumor = nil
	}
	var status *StatusPacket
	if packet.Status != nil {
		want := make([]PeerStatus,len(packet.Status.Want))
		copy(want,packet.Status.Want)
		status = &StatusPacket{Want: want}
	} else {
		status = nil
	}
	newPacket := &GossipPacket{Simple : simple, Rumor : rumor, Status : status}
	return newPacket
}