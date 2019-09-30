package main

import ("net"
		"fmt"
		"os"
		"strings"
		"github.com/dedis/protobuf"
		"github.com/simonwicky/Peerster/utils"
)


type Gossiper struct {
	addressPeer *net.UDPAddr
	connPeer *net.UDPConn
	addressClient *net.UDPAddr
	connClient *net.UDPConn
	Name string
	knownPeers []string
}


func NewGossiper(clientAddress, address, name, peers string) *Gossiper {
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
	}
}

func (g *Gossiper) addToKnownPeers(address string){
	g.knownPeers = append(g.knownPeers, address)
}

func (g *Gossiper) sendToKnowPeers(exception string, packet *utils.GossipPacket){
	for _,peer := range g.knownPeers {
		if peer == exception {
			continue
		}
		address, err := net.ResolveUDPAddr("udp4",peer)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Unable to resolve adress " + peer)
			return
		}
		connexion, err := net.DialUDP("udp4",nil, address)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Unable to connect to " + peer)
			return
		}
		packetBytes, err := protobuf.Encode(packet)
		if err != nil {
			fmt.Println("Could not serialize packet")
			return
		}
		n,_ := connexion.Write(packetBytes)

		fmt.Println("Packet sent to " + address.String() + " size: ",n)
	}
}

func (g *Gossiper) HandleClient(){
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
				fmt.Println("CLIENT MESSAGE " + packet.Simple.Contents)

				packet.Simple.OriginalName = g.Name
				packet.Simple.RelayPeerAddr = g.addressPeer.String()
				//sending to known peers
				g.sendToKnowPeers("", &packet)
			}		

		}
	//}()
}

func (g *Gossiper) HandlePeers(){
	go func(){
		fmt.Println("Listening on " + g.addressPeer.String())
		var packetBytes []byte = make([]byte, 1024)	
		for {
			var packet utils.GossipPacket
			n,_,err := g.connPeer.ReadFromUDP(packetBytes)
			if err != nil {
				fmt.Println("Error!")
				return
			}
			if n > 0 {
				protobuf.Decode(packetBytes, &packet)
				fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s\n",packet.Simple.OriginalName, packet.Simple.RelayPeerAddr, packet.Simple.Contents)
				relayPeer := packet.Simple.RelayPeerAddr
				packet.Simple.RelayPeerAddr = g.addressPeer.String()
				g.addToKnownPeers(relayPeer)
				g.sendToKnowPeers(relayPeer, &packet)
			}
		}
	}()
}