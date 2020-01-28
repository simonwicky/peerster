package gossiper

import ("fmt"
		"github.com/dedis/protobuf"
		"github.com/simonwicky/Peerster/utils"
		"os"
		"strings"
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
					case message.Keywords != nil && len(*message.Keywords) > 0:
						if message.GC != nil && *message.GC  {
							g.clientGCFileSearchHandler(&message)

						}else {
							g.clientFileSearchHandler(&message)
						}
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
	g.sendToKnownPeers("", &utils.GossipPacket{Simple: &simple})
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
		HopLimit: g.hopLimit,
	}

	packet := &utils.GossipPacket{Private: &pm}

	g.sendPointToPoint(packet, pm.Destination)
}

func (g *Gossiper) clientFileRequestHandler(message *utils.Message) {
	dr := utils.DataRequest {
		Origin: g.Name,
		Destination: *message.Destination,
		HopLimit: g.hopLimit,
		HashValue : *message.Request,
	}
	g.NewDatadownloader(&dr, *message.File)
}

func (g *Gossiper) clientFileIndexHandler(message *utils.Message) {
	g.fileStorage.addFromSystem(g, *message.File)
}

func(g *Gossiper) clientFileSearchHandler(message *utils.Message) {
	searcher := g.getFileSearcher()
	if !searcher.running {
		keywords := strings.Split(*message.Keywords,",")
		go searcher.Start(*message.Budget,keywords)
	} else {
		fmt.Fprintln(os.Stderr,"File search already running")
	}
}


func(g *Gossiper) clientGCFileSearchHandler(message *utils.Message) {
	searcher := g.getGCFileSearcher()
	if !searcher.running {
		keywords := strings.Split(*message.Keywords,",")
		go searcher.startSearch(keywords)
	} else {
		fmt.Fprintln(os.Stderr,"File search already running")
	}
}