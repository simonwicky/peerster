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
					case packet.Rumor != nil || packet.Status != nil:
						g.peerRumorStatusHandler(&packet,address.String())
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
	g.sendToKnowPeers(relayPeer, packet)
}

func (g *Gossiper) peerRumorStatusHandler(packet *utils.GossipPacket, address string){
	if worker, ok := g.lookupWorkers(address); ok {
		worker.Buffer <- *utils.CopyGossipPacket(packet)
	} else {
		new := *utils.CopyGossipPacket(packet)
		g.createAndRunWorker(address, false, nil, &new)
	}
}

func (g *Gossiper) peerPrivateMessageHandler(packet *utils.GossipPacket){
	pm := packet.Private
	if pm.Destination == g.Name {
		utils.LogPrivate(pm)
		return
	}
	go g.sendPointToPoint(packet, pm.Destination)

}

func (g *Gossiper) peerDataRequestHandler(packet *utils.GossipPacket){
	request := packet.DataRequest
	if request.Destination != g.Name {
		g.sendPointToPoint(packet, request.Destination)
		return
	}
	go g.replyData(packet.DataRequest)
}

func (g *Gossiper) peerDataReplyHandler(packet *utils.GossipPacket){
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
