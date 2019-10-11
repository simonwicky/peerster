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
			var packet utils.Message
			n,_,err := g.connClient.ReadFromUDP(packetBytes)
			if err != nil {
				fmt.Fprintln(os.Stderr,"Error!")
				return
			}

			if n > 0 {
				protobuf.Decode(packetBytes[:n], &packet)
				switch {
					case simple:
						g.clientSimpleMessageHandler(&packet)
					default:
						g.clientRumorHandler(&packet)
				}

			}		

		}
}

func (g *Gossiper) clientSimpleMessageHandler(packet *utils.Message) {
	fmt.Println("CLIENT MESSAGE " + packet.Text)

	var simple utils.SimpleMessage
	simple.OriginalName = g.Name
	simple.RelayPeerAddr = g.addressPeer.String()
	simple.Contents = packet.Text
	//sending to known peers
	g.sendToKnowPeers("", &utils.GossipPacket{Simple: &simple})
}

func (g *Gossiper) clientRumorHandler(packet *utils.Message) {
	fmt.Println("CLIENT MESSAGE " + packet.Text)
	var rumor utils.RumorMessage
	rumor.Origin = g.Name
	rumor.ID = g.counter
	rumor.Text = packet.Text
	statusIndex := -1
	for index,status := range g.currentStatus.Want {
		if status.Identifer == rumor.Origin{
			statusIndex = index
		}
	}
	g.updateStatus(utils.PeerStatus{Identifer : rumor.Origin, NextID : rumor.ID + 1}, statusIndex)
	g.counter_lock.Lock()
	g.counter += 1
	g.counter_lock.Unlock()
	g.sendToRandomPeer(&utils.GossipPacket{Rumor : &rumor})
	//add the message to storage
	g.addMessage(&rumor)

}
