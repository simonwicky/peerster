package gossiper

import ( "fmt"
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


func NewRumormonger(gossiper *Gossiper, address string, buffer chan utils.GossipPacket) *Rumormonger {
	return &Rumormonger{
		G: gossiper, 
		address: address, 
		Buffer: buffer,
		synced: false,
		waitingForAck: false,
	}
}

func (r *Rumormonger) Start() {
	fmt.Printf("STARTING talking with %s\n", r.address)
	for {
		select {
		case packet := <- r.Buffer :
			switch {
				case packet.Rumor != nil :
					r.rumorHandler(&packet)
				case packet.Status != nil :
					r.statusHandler(&packet)
					if r.waitingForAck {
						r.waitingForAck = false
						r.flipCoin(&packet)
					}
			}
		default:
			if r.synced {
				return
			} else {
				if r.waitingForAck{
					select {
						case _ = <- r.timer.C :
							r.timer.Stop()
							r.G.sendToRandomPeer(r.currentRumor)
							return
						default:  
							time.Sleep(500 * time.Millisecond)
					}
				} else {
					return
				}	
			}
		}

	}
}

func (r *Rumormonger) rumorHandler(packet *utils.GossipPacket) {
	utils.LogRumor(packet.Rumor,r.address)
	newGossiper := r.G.addToKnownPeers(r.address)
	newMessage := false
	utils.LogPeers(r.G.knownPeers)

	if newGossiper || r.missingPeer(packet.Rumor.Origin,r.G.currentStatus.Want){
		r.G.updateStatus(utils.PeerStatus{Identifer : packet.Rumor.Origin, NextID : 1}, -1)
		r.G.addMessage(packet.Rumor)
	} else {

		//check if new message from known gossiper
		for index,status := range r.G.currentStatus.Want {
			if packet.Rumor.Origin == status.Identifer && packet.Rumor.ID == status.NextID{
				//update status
				r.G.updateStatus(utils.PeerStatus{Identifer : packet.Rumor.Origin, NextID : r.G.currentStatus.Want[index].NextID + 1}, index)
				r.G.addMessage(packet.Rumor)
				newMessage = true
			}
		}
	}
	//acknowledge the message
	r.G.currentStatus_lock.RLock()
	ack := utils.GossipPacket{Status : &r.G.currentStatus}
	r.G.currentStatus_lock.RUnlock()
	r.G.sendToPeer(r.address, &ack)

	if newMessage || newGossiper {
		nextPeerAddr := r.G.sendToRandomPeer(packet)
		utils.LogMongering(nextPeerAddr)
		r.waitingForAck = true
		r.timer = time.NewTimer(10 * time.Second)
		r.currentRumor = packet
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
			key := utils.RumorMessageKey{Origin: localStatus.Identifer, ID: 1}
			msg := r.G.messages[key]
			return &utils.GossipPacket{Rumor : &msg}
		}
		for _,extStatus := range status.Want {
			//both knows the origin
			if localStatus.Identifer == extStatus.Identifer {
				if localStatus.NextID < extStatus.NextID {
					status := utils.GossipPacket{Status : &r.G.currentStatus}
					fmt.Println("HERE")
					return &status
				}
				if localStatus.NextID > extStatus.NextID {
					key := utils.RumorMessageKey{Origin: extStatus.Identifer, ID: extStatus.NextID}
					msg := r.G.messages[key]
					return &utils.GossipPacket{Rumor : &msg}
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
		utils.LogFlip(nextPeerAddr)
	}
}
