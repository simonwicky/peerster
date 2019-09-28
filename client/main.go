package main

import ("flag"
		"fmt"
		"net"
		"github.com/simonwicky/Peerster/utils"
		"github.com/dedis/protobuf"
)

func main() {
	var uiPort = flag.String("UIPort", "8080", "port for the UI client")
	var msg = flag.String("msg", "", "message to be sent")
	flag.Parse()

	message := utils.SimpleMessage{Contents : *msg}
	packetToSend := utils.GossipPacket{Simple: &message}
	send(&packetToSend, "127.0.0.1:" + *uiPort)

}

func send(packet *utils.GossipPacket, addr string) {
	udpAddr,err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		fmt.Println("Could not resolve UDP address")
		return
	}
	conn, err := net.DialUDP("udp4",nil, udpAddr)
	if err != nil {
		fmt.Println("Could not connect to server %s: %s\n", addr, err)
		return
	}
	defer conn.Close()

	packetBytes, err := protobuf.Encode(packet)
	if err != nil {
		fmt.Println("Could not serialize packet")
		return
	}
	fmt.Println(packet.Simple.Contents)
	n,_ := conn.Write(packetBytes)

	fmt.Println("Packet sent to " + udpAddr.String() + " size: ",n)
	return 


}