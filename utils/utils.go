package utils

func ArrayEquals(a []string, b []string) bool {
	if len(a) != len(b){
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
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
	var TLC *TLCMessage
	if packet.TLCMessage != nil {
		TLC = &(*packet.TLCMessage)
	}
	newPacket := &GossipPacket{Simple : simple, Rumor : rumor, Status : status,TLCMessage : TLC,}
	return newPacket
}