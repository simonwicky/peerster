package gossiper

import ("net"
		"fmt"
		"os"
		"strings"
		"github.com/dedis/protobuf"
		"github.com/simonwicky/Peerster/utils"
		"math/rand"
		"time"
)


type Gossiper struct {
	addressPeer *net.UDPAddr
	connPeer *net.UDPConn
	addressClient *net.UDPAddr
	connClient *net.UDPConn
	Name string
	knownPeers []string
	currentStatus utils.StatusPacket
	counter uint32
	messages map[utils.RumorMessageKey]utils.RumorMessage

}


func NewGossiper(clientAddress, address, name, peers string) *Gossiper {
	rand.Seed(time.Now().Unix())
	udpAddrPeer, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		fmt.Println("Unable to resolve UDP address")
		return nil
	}

	udpConnPeer, err := net.ListenUDP("udp4",udpAddrPeer)
	if err != nil {
		fmt.Println("Unable to listen")
		return nil
	}

	udpAddrClient, err := net.ResolveUDPAddr("udp4", clientAddress)
	if err != nil {
		fmt.Println("Unable to resolve UDP address")
		return nil
	}

	udpConnClient, err := net.ListenUDP("udp4",udpAddrClient)
	if err != nil {
		fmt.Println("Unable to listen")
		return nil
	}
	var peersArray []string
	if peers != ""{
		peersArray = strings.Split(peers, ",")
	}

	return &Gossiper{
		addressPeer: udpAddrPeer,
		connPeer: udpConnPeer,
		addressClient: udpAddrClient,
		connClient: udpConnClient,
		Name: name,
		knownPeers: peersArray,
		counter: 1,
	}
}

//================================
//ALL PURPOSE
//================================


func (g *Gossiper) addToKnownPeers(address string) bool {
	for _, peer := range g.knownPeers {
		if peer == address {
			return false
		}
	}
	g.knownPeers = append(g.knownPeers, address)
	return true
}

func (g *Gossiper) sendToKnowPeers(exception string, packet *utils.GossipPacket){
	for _,peer := range g.knownPeers {
		if peer == exception {
			continue
		}
		g.sendToPeer(peer, packet)
	}
}

func (g *Gossiper) sendToPeer(peer string, packet *utils.GossipPacket){
		address, err := net.ResolveUDPAddr("udp4",peer)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Unable to resolve adress " + peer)
			return
		}
		// connexion, err := net.DialUDP("udp4",g.addressPeer, address)
		// if err != nil {
		// 	fmt.Fprintln(os.Stderr, "Unable to connect to " + peer + err.Error())
		// 	return
		// }
		// defer connexion.Close()
		packetBytes, err := protobuf.Encode(packet)
		if err != nil {
			fmt.Println("Could not serialize packet")
			return
		}
		n,_ := g.connPeer.WriteToUDP(packetBytes,address)

		fmt.Println("Packet sent to " + address.String() + " size: ",n)
}
//================================================================
//CLIENT SIDE
//================================================================

//loop handling the client side
func (g *Gossiper) ClientHandle(simple bool){
	//go func() {
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
	//}()
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
	g.counter += 1
	nextPeerAddr := g.knownPeers[rand.Intn(len(g.knownPeers))]
	g.sendToPeer(nextPeerAddr, packet)
	//add the message to storage
	key := utils.RumorMessageKey{Origin : packet.Rumor.Origin, ID : packet.Rumor.ID}
	g.messages[key] = *packet.Rumor

}

func (g *Gossiper) clientStatusHandler(packet *utils.GossipPacket) {

}

//================================================================
//PEERS SIDE
//================================================================

//loop handling the peer side
func (g *Gossiper) PeersHandle(simple bool){
	go func(){
		fmt.Println("Listening on " + g.addressPeer.String())
		var packetBytes []byte = make([]byte, 1024)	
		for {
			var packet utils.GossipPacket
			n,address,err := g.connPeer.ReadFromUDP(packetBytes)
			if err != nil {
				fmt.Println("Error!")
				return
			}
			if n > 0 {
				protobuf.Decode(packetBytes, &packet)
				switch {
					case simple:
						g.peersSimpleMessageHandler(&packet)
					case packet.Rumor != nil :
						g.peersRumorHandler(&packet,address.String())
					case packet.Status != nil :
						g.peersStatusHandler(&packet,address.String())
				}
			}
		}
	}()
}

func (g *Gossiper) peersSimpleMessageHandler(packet *utils.GossipPacket) {

	fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s\n",packet.Simple.OriginalName, packet.Simple.RelayPeerAddr, packet.Simple.Contents)
	relayPeer := packet.Simple.RelayPeerAddr
	packet.Simple.RelayPeerAddr = g.addressPeer.String()
	g.addToKnownPeers(relayPeer)
	fmt.Println("PEERS " + strings.Join(g.knownPeers,","))
	g.sendToKnowPeers(relayPeer, packet)
}

func (g *Gossiper) peersRumorHandler(packet *utils.GossipPacket, address string) {
	fmt.Printf("RUMOR MESSAGE origin %s from %s ID %d contents %s\n",packet.Rumor.Origin,address,packet.Rumor.ID,packet.Rumor.Text)
	newGossiper := g.addToKnownPeers(address)
	newMessage := false
	fmt.Println( "PEERS " + strings.Join(g.knownPeers,","))

	if newGossiper {
		g.currentStatus.Want = append(g.currentStatus.Want, utils.PeerStatus{Identifer : packet.Rumor.Origin, NextID : packet.Rumor.ID + 1})
	} else {

		//check if new message from known gossiper
		for index,status := range g.currentStatus.Want {
			if packet.Rumor.Origin == status.Identifer && packet.Rumor.ID == status.NextID{
				//update status
				g.currentStatus.Want[index].NextID += 1
				newMessage = true
			}
		}
	}

	if newMessage || newGossiper {
		nextPeerAddr := g.knownPeers[rand.Intn(len(g.knownPeers))]
		g.sendToPeer(nextPeerAddr, packet)
		//TODO: implement the no response case
	}
	//acknowledge the message
	ack := utils.GossipPacket{Status : &g.currentStatus}
	g.sendToPeer(address, &ack)
}

func (g *Gossiper) peersStatusHandler(packet *utils.GossipPacket, address string) {
	fmt.Printf("STATUS from %s ", address)
	for _, status := range packet.Status.Want {
		fmt.Printf("peer %s nextID %d ", status.Identifer, status.NextID)
	}
	fmt.Printf("\n")

	for _, localStatus := range g.currentStatus.Want {
		for _,extStatus := range packet.Status.Want {
			//both knows the origin
			if localStatus.Identifer == extStatus.Identifer {
				if localStatus.NextID < extStatus.NextID {
					status := utils.GossipPacket{Status : &g.currentStatus}
					g.sendToPeer(address, &status)
					return
				}
				if localStatus.NextID > extStatus.NextID {
					key := utils.RumorMessageKey{Origin: extStatus.Identifer, ID: extStatus.NextID}
					msg := g.messages[key]
					g.sendToPeer(address, &utils.GossipPacket{Rumor : &msg})
					return
				}	
			} else {

			}
		}
	}
	//handle unknown gossiper
	//in sync, flip a coin to continue
	fmt.Printf("IN SYNC WITH %s\n",address)

}