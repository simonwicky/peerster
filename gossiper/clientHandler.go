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
					case message.Request != nil:
						//request for a file
						g.clientFileRequestHandler(&message)
					case message.File != nil:
						//indexing a file
						g.clientFileIndexHandler(&message)
					case message.Destination != nil && *message.Destination != "":
						//private message
						g.clientPrivateMessageHandler(&message)
					case message.Text != "":
						//rumor message
						g.clientRumorHandler(&message)
					default :
						fmt.Fprintln(os.Stderr,"Type of message unknown, dropping message.")
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
		Destination: *message.Destination,
		HopLimit: 10,
	}

	packet := &utils.GossipPacket{Private: &pm}

	g.sendPointToPoint(packet, pm.Destination)
}

func (g *Gossiper) clientFileRequestHandler(message *utils.Message) {
	dr := utils.DataRequest {
		Origin: g.Name,
		Destination: *message.Destination,
		HopLimit: 10,
		HashValue : *message.Request,
	}
	g.NewDatadownloader(&dr, *message.File)
}

func (g *Gossiper) clientFileIndexHandler(message *utils.Message) {
	g.fileStorage.addFromSystem(*message.File)
}
