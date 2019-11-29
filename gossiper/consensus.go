package gossiper

import ("github.com/simonwicky/Peerster/utils"

)


type Consensus struct {
	running bool
	msgList []*utils.TLCMessage
	g *Gossiper
	msg chan *utils.TLCMessage
}

func NewConsensus() *Consensus {
	return &Consensus{
		running : false,
		msg : make(chan *utils.TLCMessage,100),
	}
}


func (c *Consensus) Start(g* Gossiper,messages []*utils.TLCMessage) {
	c.running = true
	//round s
	c.g = g
	c.msgList = messages
	message := c.pickFittestMsg()
	c.g.incrementTLCRound()

	//round s + 1
	utils.LogNextRound(c.g.getTLCRound(),c.msgList)
	c.g.sendToRandomPeer(&utils.GossipPacket{TLCMessage : message})
	c.waitMessages()
	message = c.pickFittestMsg()
	c.g.incrementTLCRound()

	//round s + 2
	utils.LogNextRound(c.g.getTLCRound(),c.msgList)
	c.g.sendToRandomPeer(&utils.GossipPacket{TLCMessage : message})
	c.waitMessages()

	//update blockchain
	msg := c.pickFittestMsg()
	c.g.addBlock(&(msg.TxBlock))
	c.running = false
	utils.LogConsensus(c.g.getTLCRound(),msg,c.g.dumpBlockChain())
	c.g.incrementTLCRound()
	utils.LogNextRound(c.g.getTLCRound(),c.msgList)
	return
}

func (c *Consensus) pickFittestMsg() *utils.TLCMessage {
	max := float32(0)
	index_max := -1
	for index, m := range c.msgList {
		if m.Fitness > max {
			index_max = index
		}
	}
	return c.msgList[index_max]
}


func (c  *Consensus) waitMessages(){
	c.msgList = c.msgList[:0]
	confirmed := c.g.tlcStorage.getConfirmedMessages()
	for _,msg := range confirmed {
		if msg.Confirmed == int(c.g.getTLCRound()) {
			c.msgList = append(c.msgList, msg)
		}
	}
	for uint32(len(c.msgList)) <= c.g.peersNumber / 2 {
		msg := <- c.msg
		c.msgList = append(c.msgList,msg)
	}

}