package gossiper

import ("fmt"
		"os"
		"github.com/dedis/protobuf"
		"github.com/simonwicky/Peerster/utils"
)


//================================================================
//PEER SIDE
//================================================================

//loop handling the peer side
func (g *Gossiper) PeersHandle(simple bool){
		fmt.Fprintln(os.Stderr,"Listening on " + g.addressPeer.String())
		var packetBytes []byte = make([]byte, 10000)
		for {
			var packet utils.GossipPacket
			n,address,err := g.connPeer.ReadFromUDP(packetBytes)
			if err != nil {
				fmt.Fprintln(os.Stderr,"Error!")
				return
			}
			if n > 0 {
				err = protobuf.Decode(packetBytes[:n], &packet)
				if err != nil {
					fmt.Fprintln(os.Stderr,err.Error())
				}
				switch {
					case simple:
						g.peerSimpleMessageHandler(&packet)
					case packet.Private != nil :
						g.peerPrivateMessageHandler(&packet)
					case packet.DataRequest != nil :
						g.peerDataRequestHandler(&packet)
					case packet.DataReply != nil :
						g.peerDataReplyHandler(&packet)
					case packet.SearchRequest != nil :
						g.peerSearchRequestHandler(&packet)
					case packet.SearchReply != nil :
						g.peerSearchReplyHandler(&packet)
					case packet.GCSearchRequest != nil:
						g.peerGCSearchRequestHandler(&packet)
					case packet.GCSearchReply != nil:
						g.peerGCSearchReplyHandler(&packet)
					case packet.Rumor != nil || packet.Status != nil :
						g.peerRumorStatusHandler(&packet,address.String())
					case packet.TLCMessage != nil :
						g.peerTLCMessageHandler(&packet, address.String())
					case packet.Ack != nil :
						g.peerTLCAckHandler(&packet)
					//case packet.Clove != nil :
						//g.peerCloveHandler(&packet)
					default:
						fmt.Fprintln(os.Stderr,"Message unknown, dropping packet")
				}
			}
		}
}

func (g *Gossiper) peerSimpleMessageHandler(packet *utils.GossipPacket) {

	utils.LogSimpleMessage(packet.Simple)
	relayPeer := packet.Simple.RelayPeerAddr
	packet.Simple.RelayPeerAddr = g.addressPeer.String()
	g.addToKnownPeers(relayPeer)
	utils.LogPeers(g.knownPeers)
	g.sendToKnownPeers(relayPeer, packet)
}

func (g *Gossiper) peerRumorStatusHandler(packet *utils.GossipPacket, address string){
	fmt.Fprintln(os.Stderr,"Rumor or Status received")
	if worker, ok := g.lookupWorkers(address); ok {
		worker.Buffer <- *utils.CopyGossipPacket(packet)
	} else {
		new := *utils.CopyGossipPacket(packet)
		g.createAndRunWorker(address, false, nil, &new)
	}
}

func (g *Gossiper) peerPrivateMessageHandler(packet *utils.GossipPacket){
	fmt.Fprintln(os.Stderr,"PrivateMessage received")
	pm := packet.Private
	if pm.Destination == g.Name {
		utils.LogPrivate(pm)
		return
	}
	go g.sendPointToPoint(packet, pm.Destination)

}

func (g *Gossiper) peerDataRequestHandler(packet *utils.GossipPacket){
	fmt.Fprintln(os.Stderr,"DataRequest received")
	request := packet.DataRequest
	if request.Destination != g.Name {
		g.sendPointToPoint(packet, request.Destination)
		return
	}
	go g.replyData(packet.DataRequest)
}

func (g *Gossiper) peerDataReplyHandler(packet *utils.GossipPacket){
	fmt.Fprintln(os.Stderr,"DataReply received")
	reply := packet.DataReply
	if reply.Destination != g.Name {
		g.sendPointToPoint(packet, reply.Destination)
		return
	}
	dd := g.lookupDownloader(reply.HashValue)
	if dd == nil {
		fmt.Fprintln(os.Stderr,"Didn't found the corresponding downloader")
		return
	}
	dd.replies <- reply
}

func (g *Gossiper) peerSearchRequestHandler(packet *utils.GossipPacket){
	fmt.Fprintln(os.Stderr,"SearchRequest received")
	request := packet.SearchRequest
	if (!g.lookupSearchRequest(request.Origin,request.Keywords)) {
		go g.NewSearchReplier(request)
		return
	}
	fmt.Fprintln(os.Stderr,"Duplicate SearchRequest received")
}

func (g *Gossiper) peerSearchReplyHandler(packet *utils.GossipPacket){
	fmt.Fprintln(os.Stderr,"SearchReply received")
	reply := packet.SearchReply
	if reply.Destination != g.Name {
		g.sendPointToPoint(packet, reply.Destination)
		return
	}
	searcher := g.getFileSearcher()
	if searcher.running {
		searcher.replies <- reply
	} else {
		fmt.Fprintln(os.Stderr,"Search not in progress")
	}
}

func (g *Gossiper) peerTLCMessageHandler(packet *utils.GossipPacket, address string){
	fmt.Fprintln(os.Stderr,"TLCMessage received")
	//ultimately send it to the rumormonger
	defer g.peerRumorStatusHandler(packet, address)
	msg := packet.TLCMessage
	if msg.Origin == g.Name {
		return
	}
	utils.LogTLCGossip(msg)

	if msg.Confirmed != -1 {
		g.tlcStorage.addMessage(msg)

		if g.hw3ex4 && g.consensus.running {
			g.consensus.msg <- msg
			return
		}
		publisher := g.checkPublisher(uint32(msg.Confirmed))
		if (publisher != nil) {
			publisher.msg <- msg
		}
		return
	}
	if g.checkBlockValidity(&msg.TxBlock){
		g.tlcStorage.addMessage(msg)
		g.TLCAck(packet)
	} else {
		fmt.Fprintln(os.Stderr,"mapping already exists")
		//don't ack the message, for now nothing
	}
}

func (g *Gossiper) peerTLCAckHandler(packet *utils.GossipPacket){
	fmt.Fprintln(os.Stderr,"TLCACK received")
	ack := packet.Ack
	if ack.Destination == g.Name {
		if p := g.lookupPublisher(ack.ID); p != nil {
			p.acks <- ack
		} else {
			fmt.Fprintln(os.Stderr,"Publisher not found, dropping packet")
		}
		return
	}
	go g.sendPointToPoint(packet, ack.Destination)

}

func (g *Gossiper) peerGCSearchRequestHandler(packet *utils.GossipPacket){

}

func (g *Gossiper) peerGCSearchReplyHandler(packet *utils.GossipPacket){
	
}


// func (g *Gossiper) peerCloveHandler(packet *utils.GossipPacket){
// 	fmt.Fprintln(os.Stderr,"Clove received")
// 	clove := packet.Clove
// 	g.addClove(clove)
// }

