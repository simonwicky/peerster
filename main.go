package main

import ("flag"
		"github.com/simonwicky/Peerster/gossiper"
		"fmt"
		"os"
)



func main() {
	var udpPort = flag.String("UIPort", "8080", "port for the UI client")
	var guiport = flag.String("GUIPort", "8080", "port for the GUI client")
	var gossipAddr = flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper")
	var name = flag.String("name","","name of the gossiper")
	var peers = flag.String("peers", "","comma separated list of peers of the form ip:port")
	var simple = flag.Bool("simple", false, "run gossiper in simple broadcast mode")
	var antiEntropy = flag.Int("antientropy",10, "time between antiEntropy checks. 0 disables it")
	var rtimer = flag.Int("rtimer", 0, "Timeout in seconds to send route rumors. 0 (default) means disable sending route rumors.")
	var hoplimit = flag.Int("hoplimit",10,"HopLimit for TLCAcks")
	var numberPeers = flag.Int("N", 0, "Number of peers in the network")
	var stubbornTimeout = flag.Int("stubbornTimeout",5,"Duration of the stubbornTimeout")
	var hw3ex2 = flag.Bool("hw3ex2",false,"Enable TLCMessage when storing file")
	flag.Parse()

	gossiper := gossiper.NewGossiper("127.0.0.1:" + *udpPort, *gossipAddr, *name, *peers, *antiEntropy, *rtimer, *hoplimit, *numberPeers, *stubbornTimeout, *hw3ex2)
	if gossiper == nil {
		fmt.Fprintln(os.Stderr,"Problem initializing gossiper")
		return
	}
	gossiper.Start(*simple, *guiport)
}



