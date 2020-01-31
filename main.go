package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/simonwicky/Peerster/gossiper"
	"github.com/simonwicky/Peerster/utils"
)

func main() {
	var udpPort = flag.String("UIPort", "8080", "port for the UI client")
	var guiport = flag.String("GUIPort", "8080", "port for the GUI client")
	var gossipAddr = flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper")
	var name = flag.String("name", "", "name of the gossiper")
	var peers = flag.String("peers", "", "comma separated list of peers of the form ip:port")
	var simple = flag.Bool("simple", false, "run gossiper in simple broadcast mode")
	var antiEntropy = flag.Int("antientropy", 10, "time between antiEntropy checks. 0 disables it")
	var rtimer = flag.Int("rtimer", 0, "Timeout in seconds to send route rumors. 0 (default) means disable sending route rumors.")
	var hoplimit = flag.Int("hoplimit", 10, "HopLimit for TLCAcks")
	var numberPeers = flag.Int("N", 0, "Number of peers in the network")
	var stubbornTimeout = flag.Int("stubbornTimeout", 5, "Duration of the stubbornTimeout")
	var hw3ex2 = flag.Bool("hw3ex2", false, "Enable TLCMessage when storing file")
	var hw3ex3 = flag.Bool("hw3ex3", false, "Enable TLC Round")
	var hw3ex4 = flag.Bool("hw3ex4", false, "Enable QSC")
	var filters = flag.String("filter", "", "activated log tags")
	var proxy = flag.String("proxy", ":666", "direct proxy port")
	var _ = flag.Bool("ackall", true, "Ack everything")
	flag.Parse()
	utils.LogObj.Filter(*filters)
	gossiper := gossiper.NewGossiper("127.0.0.1:"+*udpPort, *gossipAddr, *name, *peers, *antiEntropy, *rtimer, *hoplimit, *numberPeers, *stubbornTimeout, *hw3ex2, *hw3ex3, *hw3ex4, *proxy)
	if gossiper == nil {
		fmt.Println("Could not initialize ",name)
		fmt.Fprintln(os.Stderr, "Problem initializing gossiper")
		return
	}
	//fmt.Fprintln(os.Stderr,"GUI on port " + *guiport)
	gossiper.Start(*simple, *guiport)
}
