package main

import ("flag"
		"fmt"
		"net"
		"github.com/simonwicky/Peerster/utils"
		"github.com/dedis/protobuf"
		"os"
)

func main() {
	var uiPort = flag.String("UIPort", "8080", "port for the UI client")
	var msg = flag.String("msg", "", "message to be sent")
	flag.Parse()

	message := utils.Message{Text : *msg}
	//no tag simple here, so we send simple + rumor
	packetToSend := &message
	send(packetToSend, "127.0.0.1:" + *uiPort)

}

func send(packet *utils.Message, addr string) {
	udpAddr,err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		fmt.Fprintln(os.Stderr,"Could not resolve UDP address")
		return
	}
	conn, err := net.DialUDP("udp4",nil, udpAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr,"Could not connect to server %s: %s\n", addr, err)
		return
	}
	defer conn.Close()

	packetBytes, err := protobuf.Encode(packet)
	if err != nil {
		fmt.Fprintln(os.Stderr,"Could not serialize packet")
		return
	}
	n,_ := conn.Write(packetBytes)

	fmt.Fprintln(os.Stderr,"Packet sent to " + udpAddr.String() + " size: ",n)
	return 


}