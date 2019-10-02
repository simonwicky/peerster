package gossiper

import ("fmt"
		"github.com/dedis/protobuf"
		"github.com/simonwicky/Peerster/utils"
)

//================================================================
//CLIENT SIDE
//================================================================

//loop handling the client side
func (g *Gossiper) ClientHandle(simple bool){
		fmt.Println("Listening on " + g.addressClient.String())
		var packetBytes []byte = make([]byte, 1024)	
		for {
			var packet utils.GossipPacket
			n,_,err := g.connClient.ReadFromUDP(packetBytes)
			if err != nil {
				fmt.Println("Error!")
				return
			}

			if n > 0 {
				protobuf.Decode(packetBytes, &packet)
				switch {
					case simple:
						g.clientSimpleMessageHandler(&packet)
					case packet.Rumor != nil :
						g.clientRumorHandler(&packet)
					case packet.Status != nil :
						g.clientStatusHandler(&packet)
				}

			}		

		}
}

func (g *Gossiper) clientSimpleMessageHandler(packet *utils.GossipPacket) {
	fmt.Println("CLIENT MESSAGE " + packet.Simple.Contents)

	packet.Simple.OriginalName = g.Name
	packet.Simple.RelayPeerAddr = g.addressPeer.String()
	//sending to known peers
	g.sendToKnowPeers("", packet)
}

func (g *Gossiper) clientRumorHandler(packet *utils.GossipPacket) {
	fmt.Println("CLIENT MESSAGE " + packet.Rumor.Text)
	packet.Rumor.Origin = g.Name
	packet.Rumor.ID = g.counter
	statusIndex := -1
	for index,status := range g.currentStatus.Want {
		if status.Identifer == packet.Rumor.Origin{
			statusIndex = index
		}
	}
	g.updateStatus(utils.PeerStatus{Identifer : packet.Rumor.Origin, NextID : packet.Rumor.ID + 1}, statusIndex)
	g.counter += 1
	g.sendToRandomPeer(packet)
	//add the message to storage
	key := utils.RumorMessageKey{Origin : packet.Rumor.Origin, ID : packet.Rumor.ID}
	g.messages[key] = *packet.Rumor

}

func (g *Gossiper) clientStatusHandler(packet *utils.GossipPacket) {

}