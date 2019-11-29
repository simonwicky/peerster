package gossiper

import ("github.com/simonwicky/Peerster/utils"
		"fmt"
		"os"
		"time"
		"math/rand"

)

type TLCPublisher struct {
	id uint32
	roundID uint32
	nbAcks uint32
	acks chan *utils.TLCAck
	ackList []string
	g *Gossiper
	msg chan *utils.TLCMessage
	msgList []*utils.TLCMessage
	nbMsg uint32
	running bool
	infos *utils.FileInfo
}


func Publish(g *Gossiper, infos *utils.FileInfo) {
	if g.hw3ex3 && g.checkPublisher(g.getTLCRound()) != nil {
		g.bufferInfos(infos)
		return
	}
	publisher := &TLCPublisher{
		id : g.getTLCID(),
		roundID : g.getTLCRound(),
		nbAcks : 1, //for itself
		g : g,
		ackList : []string{g.Name},
		acks : make(chan *utils.TLCAck,g.peersNumber),
		msg : make(chan *utils.TLCMessage,g.peersNumber),
		nbMsg : 0,
		running : false,
		infos : infos,
	}
	publisher.Start()
}

func (p *TLCPublisher) Start() {
	p.running = true
	p.g.addPublisher(p)
	go p.sendTLCMessage(p.infos.Name,p.infos.Size,p.infos.MetafileHash,-1, p.id)
	if p.g.hw3ex3 {
		go p.handleRoundTransition()
	}
	p.handleAcks()
	utils.LogConfirmedID(p.id, p.ackList)
	p.sendTLCMessage(p.infos.Name,p.infos.Size,p.infos.MetafileHash, int(p.roundID), p.g.getTLCID())
	if !p.g.hw3ex3 {
		p.g.deletePublisher(p.id)
	}
}

func (p* TLCPublisher) sendTLCMessage(name string, size int64, metafileHash []byte, confirmed int, id uint32) {
	if p.g.hw3ex3 && !p.running {
		//no broadcast if moving to next round
		return
	}
	txPublish := utils.TxPublish{
		Name: name,
		Size: size,
		MetafileHash: metafileHash,
	}
	block := utils.BlockPublish{
		PrevHash: p.g.getLastHash(),
		Transaction : txPublish,
	}
	message := &utils.TLCMessage{
		Origin : p.g.Name,
		ID : id,
		Confirmed : confirmed,
		TxBlock : block,
		VectorClock : &p.g.currentStatus,
		Fitness : rand.Float32(),
	}
	timeout := time.NewTimer(time.Second * p.g.stubbornTimeout)
	p.g.sendToRandomPeer(&utils.GossipPacket{TLCMessage : message})
	p.g.tlcStorage.addMessage(message)
	if confirmed != -1 {
		publisher := p.g.checkPublisher(uint32(confirmed))
		if (publisher != nil) {
			publisher.msg <- message
		}
		p.running = false
	}
	for {
		if !p.running {
			return
		}
		select {
			case _ = <- timeout.C:
				timeout.Reset(p.g.stubbornTimeout * time.Second)
				p.g.sendToRandomPeer(&utils.GossipPacket{TLCMessage : message})
			default:
				time.Sleep(500 * time.Millisecond)
		}
	}

}

func (p *TLCPublisher) handleAcks(){
	for {
		if p.nbAcks > p.g.peersNumber / 2 {
			fmt.Fprintln(os.Stderr,"Enough ACK received")
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

func (p *TLCPublisher) handleRoundTransition(){
	confirmed := p.g.tlcStorage.getConfirmedMessages()
	for _,msg := range confirmed {
		if msg.Confirmed == int(p.roundID) {
			p.msgList = append(p.msgList, msg)
			p.nbMsg += 1
		}
	}
	fmt.Fprintln(os.Stderr,"Waiting for confirmed message")
	for p.nbMsg <= p.g.peersNumber / 2 {
		msg := <- p.msg
		p.msgList = append(p.msgList,msg)
		p.nbMsg += 1
	}
	if p.running {
		p.running = false
		p.g.bufferInfos(p.infos)
	}
	if p.g.hw3ex4 {
		p.g.consensus.Start(p.g,p.msgList)
	} else {
		p.g.incrementTLCRound()
		utils.LogNextRound(p.g.getTLCRound(),p.msgList)
	}
		p.g.deletePublisher(p.id)
	infos := p.g.getNextPublishInfos()
	if infos == nil {
		return
	}
	Publish(p.g,infos)
}



func (g *Gossiper) TLCAck(packet *utils.GossipPacket){
	msg := packet.TLCMessage
	ack := utils.TLCAck{
		Origin: g.Name,
		ID: msg.ID,
		Text : "",
		Destination: msg.Origin,
		HopLimit: g.hopLimit,
	}
	utils.LogAck(msg.Origin, msg.ID)
	g.sendPointToPoint(&utils.GossipPacket{Ack: &ack}, ack.Destination)
}