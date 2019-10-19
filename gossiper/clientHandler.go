package gossiper

import ("fmt"
		"github.com/dedis/protobuf"
		"github.com/simonwicky/Peerster/utils"
		"os"
)

//================================================================
//CLIENT SIDE
//================================================================

//loop handling the client side
func (g *Gossiper) ClientHandle(simple bool){
		fmt.Fprintln(os.Stderr,"Listening on " + g.addressClient.String())
		var packetBytes []byte = make([]byte, 1024)	
		for {
			var message utils.Message
			n,_,err := g.connClient.ReadFromUDP(packetBytes)
			if err != nil {
				fmt.Fprintln(os.Stderr,"Error!")
				return
			}

			if n > 0 {
				protobuf.Decode(packetBytes[:n], &message)
				switch {
					case simple:
						g.clientSimpleMessageHandler(&message)
					case message.Destination != "":
						g.clientPrivateMessageHandler(&message)
					default:
						g.clientRumorHandler(&message)
				}

			}		

		}
}

func (g *Gossiper) clientSimpleMessageHandler(message *utils.Message) {
	utils.LogClient(message.Text)

	var simple utils.SimpleMessage
	simple.OriginalName = g.Name
	simple.RelayPeerAddr = g.addressPeer.String()
	simple.Contents = message.Text
	//sending to known peers
	g.sendToKnowPeers("", &utils.GossipPacket{Simple: &simple})
}

func (g *Gossiper) clientRumorHandler(message *utils.Message) {
	utils.LogClient(message.Text)
	rumor := g.generateRumor(message.Text)
	g.sendToRandomPeer(&utils.GossipPacket{Rumor : &rumor})

}

func (g *Gossiper) clientPrivateMessageHandler(message *utils.Message){
	pm := utils.PrivateMessage{
		Origin: g.Name,
		ID: 0,
		Text : message.Text,
		Destination: message.Destination,
		HopLimit: 10,
	}
	address := g.lookupDSDV(message.Destination)
	if address == "" {
		fmt.Fprintln(os.Stderr,"Next hop not found, aborting")
		return
	}

	g.sendToPeer(address, &utils.GossipPacket{Private: &pm})
}
