package gossiper

import ( "fmt"
		"os"
		 "github.com/simonwicky/Peerster/utils"
		 "math/rand"
		 "time"
)
type Rumormonger struct {
	G *Gossiper
	address string
	Buffer chan utils.GossipPacket
	synced bool
	waitingForAck bool
	timer *time.Timer
	currentRumor *utils.GossipPacket

}


func NewRumormonger(gossiper *Gossiper, address string, buffer chan utils.GossipPacket, waitingForAck bool, currentRumor *utils.GossipPacket) *Rumormonger {
	rumormonger := &Rumormonger{
		G: gossiper,
		address: address,
		Buffer: buffer,
		synced: false,
		waitingForAck: waitingForAck,
		currentRumor: currentRumor,
	}
	if waitingForAck {
		rumormonger.timer = time.NewTimer(10 * time.Second)
	}
	return rumormonger
}

func (r *Rumormonger) Start() {
	//fmt.Fprintf(os.Stderr,"STARTED talking with %s\n", r.address)
	for {
		if r.synced && !r.waitingForAck {
			return
		}
		select {
			case packet := <- r.Buffer :
				switch {
					case packet.Rumor != nil || packet.TLCMessage != nil:
						r.rumorHandler(&packet)
					case packet.Status != nil :
						r.statusHandler(&packet)
						if r.waitingForAck {
							r.waitingForAck = false
							r.flipCoin(r.currentRumor)
						}
				}
			default:
				if r.waitingForAck {
					if r.timer == nil {
						fmt.Fprintln(os.Stderr,"Timer is nil")
						return
					}
					select {
						case _ = <- r.timer.C :
							r.timer.Stop()
							nextPeerAddr := r.G.sendToRandomPeer(r.currentRumor)
							fmt.Fprintln(os.Stderr,"Timed out, sending to " + nextPeerAddr)
							r.G.createAndRunWorker(nextPeerAddr, true, r.currentRumor, nil)
							return
						default:
							time.Sleep(500 * time.Millisecond)
					}
				}
		}

	}
}

//handle TLCMessage here
func (r *Rumormonger) rumorHandler(packet *utils.GossipPacket) {
	var origin string
	var id uint32
	if packet.Rumor != nil {
		origin = packet.Rumor.Origin
		id = packet.Rumor.ID
		utils.LogRumor(packet.Rumor,r.address)
	} else {
		origin = packet.TLCMessage.Origin
		id = packet.TLCMessage.ID
	}
	newGossiper := r.G.addToKnownPeers(r.address)
	newMessage := false
	utils.LogPeers(r.G.knownPeers)

	//check if new peer, or new origin
	if newGossiper || r.missingPeer(origin,r.G.currentStatus.Want){
		nextID := id + 1
		//if the message we get is out of order, we need all of them, starting from 1
		if id != 1 {
			nextID = 1
		}
		r.G.updateStatus(utils.PeerStatus{Identifer : origin, NextID : nextID}, -1)
		newMessage = true;
	} else {
		//fmt.Fprintln(os.Stderr,"Message from known peer")
		//check if new message from known gossiper
		for index,status := range r.G.currentStatus.Want {
			if origin == status.Identifer && id == status.NextID{
				//update status
				r.G.updateStatus(utils.PeerStatus{Identifer : origin, NextID : r.G.currentStatus.Want[index].NextID + 1}, index)
				newMessage = true
			}
		}
	}
	//acknowledge the message
	r.G.currentStatus_lock.RLock()
	ack := utils.GossipPacket{Status : &r.G.currentStatus}
	r.G.currentStatus_lock.RUnlock()
	r.G.sendToPeer(r.address, &ack)
	r.waitingForAck = false

	if newMessage {
		nextPeerAddr := r.G.sendToRandomPeer(packet)
		utils.LogMongering(nextPeerAddr)
		r.waitingForAck = true
		r.timer = time.NewTimer(10 * time.Second)
		r.currentRumor = packet
		r.G.updateDSDV(origin, r.address)
		if packet.TLCMessage != nil || packet.Rumor.Text != "" {
			r.G.addMessage(packet)
			utils.LogDSDV(origin, r.address)
		}
	}
}

func (r *Rumormonger) statusHandler(packet *utils.GossipPacket) {
	utils.LogStatus(packet.Status.Want,r.address)

	gossip := r.checkVectorClock(packet.Status)
	if gossip == nil {
		utils.LogSync(r.address)
		r.synced = true
	} else {
		r.G.sendToPeer(r.address, gossip)
	}
}

//returns a packet containing a rumor or a message to send, nil if peers are synced
func (r *Rumormonger) checkVectorClock(status *utils.StatusPacket) *utils.GossipPacket {
	r.G.currentStatus_lock.RLock()
	defer r.G.currentStatus_lock.RUnlock()
	for _, localStatus := range r.G.currentStatus.Want {
		if r.missingPeer(localStatus.Identifer,status.Want){
			//other node doesn't know peer from local status
			msg := r.G.getMessage(localStatus.Identifer, 1)
			return msg
		}
		for _,extStatus := range status.Want {
			//both knows the origin
			if localStatus.Identifer == extStatus.Identifer {
				if localStatus.NextID < extStatus.NextID {
					status := utils.GossipPacket{Status : &r.G.currentStatus}
					return &status
				}
				if localStatus.NextID > extStatus.NextID {
					msg := r.G.getMessage(extStatus.Identifer, extStatus.NextID)
					return msg
				}
			}
		}
	}
	for _,extStatus := range status.Want {
		if r.missingPeer(extStatus.Identifer, r.G.currentStatus.Want){
			//we don't know a peer that he knows about
			status := utils.GossipPacket{Status : &r.G.currentStatus}
			return &status
		}
	}

	return nil
}

//check if peer is missing from want or not
func (r *Rumormonger) missingPeer(peerIdentifier string, want []utils.PeerStatus) bool {
	r.G.currentStatus_lock.RLock()
	defer r.G.currentStatus_lock.RUnlock()
	for _, s := range want {
		if s.Identifer == peerIdentifier {
			return false
		}
	}
	return true
}

//flip the coin to choose if continuing or not
func (r *Rumormonger) flipCoin(packet *utils.GossipPacket) {
	coinFlip := rand.Int() % 2 == 0
	if coinFlip {
		nextPeerAddr := r.G.sendToRandomPeer(packet)
		r.G.createAndRunWorker(nextPeerAddr, true, packet, nil)
		utils.LogFlip(nextPeerAddr)
	}
}

