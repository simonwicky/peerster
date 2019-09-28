package main

import ("net"
		"fmt"
		"github.com/dedis/protobuf"
		"github.com/simonwicky/Peerster/utils"
)


type Gossiper struct {
	addressPeer *net.UDPAddr
	connPeer *net.UDPConn
	addressClient *net.UDPAddr
	connClient *net.UDPConn
	Name string
	knownPeers string
	// callbackNewPeers func()
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

	return &Gossiper{
		addressPeer: udpAddrPeer,
		connPeer: udpConnPeer,
		addressClient: udpAddrClient,
		connClient: udpConnClient,
		Name: name,
		knownPeers: peers,
	}
}

func (g *Gossiper) HandleClient(){
	go func() {
		fmt.Println("Listening on " + g.addressClient.String())
		var packetBytes []byte = make([]byte, 1024)	
		for {
			var packet utils.GossipPacket
			n,address,err := g.connClient.ReadFromUDP(packetBytes)
			if err != nil {
				fmt.Println("Error!")
				return
			}
			_ = address

			if n > 0 {
				protobuf.Decode(packetBytes, &packet)
				fmt.Println(packet.Simple.Contents)	
			}		
			fmt.Println(n)

		}
	}()
}

// go func (g *Gossiper) HandlePeers(){
// 	for {

// 	}
// }