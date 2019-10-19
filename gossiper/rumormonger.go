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
	fmt.Fprintf(os.Stderr,"STARTING talking with %s\n", r.address)
	for {
		if r.synced && !r.waitingForAck {
			return
		}
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
				if r.waitingForAck {
					if r.timer == nil {
						fmt.Fprintln(os.Stderr,"WHY THE FUCK")
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

func (r *Rumormonger) rumorHandler(packet *utils.GossipPacket) {
	utils.LogRumor(packet.Rumor,r.address)
	newGossiper := r.G.addToKnownPeers(r.address)
	newMessage := false
	utils.LogPeers(r.G.knownPeers)

	//check if new peer, or new origin
	if newGossiper || r.missingPeer(packet.Rumor.Origin,r.G.currentStatus.Want){
		nextID := packet.Rumor.ID + 1
		//if the message we get is out of order, we need all of them, starting from 1
		if packet.Rumor.ID != 1 {
			nextID = 1
		}
		r.G.updateStatus(utils.PeerStatus{Identifer : packet.Rumor.Origin, NextID : nextID}, -1)
		newMessage = true;
	} else {

		//check if new message from known gossiper
		for index,status := range r.G.currentStatus.Want {
			if packet.Rumor.Origin == status.Identifer && packet.Rumor.ID == status.NextID{
				//update status
				r.G.updateStatus(utils.PeerStatus{Identifer : packet.Rumor.Origin, NextID : r.G.currentStatus.Want[index].NextID + 1}, index)
				newMessage = true
			}
		}
	}
	//acknowledge the message
	r.G.currentStatus_lock.RLock()
	ack := utils.GossipPacket{Status : &r.G.currentStatus}
	r.G.currentStatus_lock.RUnlock()
	r.G.sendToPeer(r.address, &ack)

	if newMessage {
		nextPeerAddr := r.G.sendToRandomPeer(packet)
		utils.LogMongering(nextPeerAddr)
		r.waitingForAck = true
		r.timer = time.NewTimer(10 * time.Second)
		r.currentRumor = packet
		r.G.updateDSDV(packet.Rumor.Origin, r.address)
		if packet.Rumor.Text != "" {
			r.G.addMessage(packet.Rumor)
			utils.LogDSDV(packet.Rumor.Origin, r.address)
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
			return &utils.GossipPacket{Rumor : &msg}
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
		r.G.createAndRunWorker(nextPeerAddr, true, packet, nil)
		utils.LogFlip(nextPeerAddr)
	}
}

func (r *Rumormonger) setCurrentRumor(packet *utils.GossipPacket) {
	r.currentRumor = packet
}
