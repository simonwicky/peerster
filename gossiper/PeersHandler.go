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
		var packetBytes []byte = make([]byte, 1024)	
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
				if simple {
					g.peersSimpleMessageHandler(&packet)
				} else if packet.Private != nil {
					g.peerPrivateMessageHandler(&packet)
				} else {
					if worker, ok := g.lookupWorkers(address.String()); ok {
						worker.Buffer <- *utils.CopyGossipPacket(&packet)
					} else {
						new := *utils.CopyGossipPacket(&packet)
						g.createAndRunWorker(address.String(), false, nil, &new)
					}
				}
			}
		}
}

func (g *Gossiper) peersSimpleMessageHandler(packet *utils.GossipPacket) {

	utils.LogSimpleMessage(packet.Simple)
	relayPeer := packet.Simple.RelayPeerAddr
	packet.Simple.RelayPeerAddr = g.addressPeer.String()
	g.addToKnownPeers(relayPeer)
	utils.LogPeers(g.knownPeers)
	g.sendToKnowPeers(relayPeer, packet)
}

func (g *Gossiper) peerPrivateMessageHandler(packet *utils.GossipPacket){
	pm := packet.Private
	if pm.Destination == g.Name {
		utils.LogPrivate(pm)
		return
	}
	pm.HopLimit -= 1
	if pm.HopLimit <= 0 {
		fmt.Fprintln(os.Stderr,"No more hop, dropping packet")
		return
	}
	address := g.lookupDSDV(pm.Destination)
	if address == "" {
		fmt.Fprintln(os.Stderr,"Next hop not found, aborting")
		return
	}
	g.sendToPeer(address, packet)

}