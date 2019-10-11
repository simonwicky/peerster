package main

import ("flag"
		"github.com/simonwicky/Peerster/gossiper"
		"fmt"
		"os"
)



func main() {
	var udpPort = flag.String("UIPort", "8080", "port for the UI client")
	var gossipAddr = flag.String("gossipAddr", "127.0.0.1:5000", "ip:port for the gossiper")
	var name = flag.String("name","","name of the gossiper")
	var peers = flag.String("peers", "","comma separated list of peers of the form ip:port")
	var simple = flag.Bool("simple", false, "run gossiper in simple broadcast mode")
	var antiEntropy = flag.Int("antientropy",10, "time between antiEntropy checks")
	flag.Parse()
	 
	gossiper := gossiper.NewGossiper("127.0.0.1:" + *udpPort, *gossipAddr, *name, *peers, *antiEntropy)
	if gossiper == nil {
		fmt.Fprintln(os.Stderr,"Problem initializing gossiper")
		return
	}
	gossiper.Start(*simple)
}



