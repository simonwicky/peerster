package gossiper

import ("github.com/simonwicky/Peerster/utils"
		"fmt"
		"os"
		"time"

)

type TLCPublisher struct {
	id uint32
	nbAcks uint32
	ackList []string
	g *Gossiper
	acks chan *utils.PrivateMessage
	running bool
}


func Publish(g *Gossiper, name string, size int64, metafileHash []byte) {
//send TLCMessages and wait for acks use stubbornTimeout in gossiper
//broadcast the confirmed message
	publisher := &TLCPublisher{
		id : g.getTLCID(),
		nbAcks : 1,
		g : g,
		ackList : []string{g.Name},
		acks : make(chan *utils.PrivateMessage),
		running : true,
	}
	g.addPublisher(publisher)
	go publisher.sendTLCMessage(name,size,metafileHash,false)
	publisher.handleAcks()
	utils.LogConfirmedID(publisher.id, publisher.ackList)
	publisher.sendTLCMessage(name,size,metafileHash, true)

}

func (p* TLCPublisher) sendTLCMessage(name string, size int64, metafileHash []byte, confirmed bool) {
		txPublish := utils.TxPublish{
		Name: name,
		Size: size,
		MetafileHash: metafileHash,
	}
	block := utils.BlockPublish{
		PrevHash: [32]byte{},
		Transaction : txPublish,
	}
	message := &utils.TLCMessage{
		Origin : p.g.Name,
		ID : p.id,
		Confirmed : confirmed,
		TxBlock : block,
		VectorClock : nil,
		Fitness : 0,
	}
	timeout := time.NewTimer(time.Second * p.g.stubbornTimeout)
	p.g.sendToKnownPeers("",&utils.GossipPacket{TLCMessage : message})
	for {
		if !p.running {
			return
		}
		select {
			case _ = <- timeout.C:
				timeout.Reset(p.g.stubbornTimeout * time.Second)
				p.g.sendToKnownPeers("",&utils.GossipPacket{TLCMessage : message})
			default:
				time.Sleep(500 * time.Millisecond)
		}
	}

}

func (p *TLCPublisher) handleAcks(){
	for {
		if p.nbAcks > p.g.peersNumber / 2 {
			p.running = false
			return
		}
		select {
			case ack := <- p.acks:
				new_Peer := true
				for _, peer := range p.ackList{
					if ack.Origin == peer {
						fmt.Fprintln(os.Stderr,"This peer already acked this ID")
						new_Peer = false
					}
				}
				if new_Peer {
					p.ackList = append(p.ackList,ack.Origin)
					p.nbAcks += 1
				}
			default:
				time.Sleep(10 * time.Millisecond)
		}
	}
}


func (g *Gossiper) TLCAck(packet *utils.GossipPacket){
	msg := packet.TLCMessage
	pm := utils.PrivateMessage{
		Origin: g.Name,
		ID: msg.ID,
		Text : "",
		Destination: msg.Origin,
		HopLimit: g.hopLimit,
	}
	utils.LogAck(msg.Origin, msg.ID)
	g.sendPointToPoint(&utils.GossipPacket{Private: &pm}, pm.Destination)
}