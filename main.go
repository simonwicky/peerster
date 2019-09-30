package main

import ("flag"
		_ "fmt"
		_ "github.com/simonwicky/Peerster/utils"
		_ "net"
)



func main() {
	var udpPort = flag.String("UIPort", "8080", "port for the UI client")
	var gossipAddr = flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper")
	var name = flag.String("name","","name of the gossiper")
	var peers = flag.String("peers", "","comma separated list of peers of the form ip:port")
	var simple = flag.Bool("simple", false, "run gossiper in simple broadcast mode")
	flag.Parse()
	_ = simple
	 
	gossiper := NewGossiper("127.0.0.1:" + *udpPort, *gossipAddr, *name, *peers)
	gossiper.HandlePeers() 
	gossiper.HandleClient()
}





func receiveClient(){

}

func receivePeers(){

}
