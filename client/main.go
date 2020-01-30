package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/simonwicky/Peerster/utils"
	"go.dedis.ch/protobuf"
)

var GOSSIPER_ADDRESS string = "127.0.0.1"

func main() {
	var uiPort = flag.String("UIPort", "8080", "port for the UI client")
	var destination = flag.String("dest", "", "destination for the private message; can be omitted")
	var msg = flag.String("msg", "", "message to be sent; if the -dest flag is present, this is a private message, otherwise it's a rumor message")
	var file = flag.String("file", "", "file to be indexed by the gossiper")
	var request = flag.String("request", "", "request a chunk or metafile of this hash")
	var keywords = flag.String("keywords","","Keywords for perforimg a search")
	var budget = flag.Uint64("budget",2,"budget for the file search")
	var gc  = flag.Bool("garlic", false, "set to true with the search arguments to activate the garlic cast search")
	var useProxy  = flag.Bool("useProxy", false, "set to true with the search arguments to delegate the search the proxies")
	flag.Parse()

	//flag for file request
	if *request != "" && *msg == "" && *keywords == "" {
		if (*file == "" && *destination == "") || (*file != "" && *destination != "") {
			request_bytes, err := hex.DecodeString(*request)
			if err != nil {
				fmt.Println("ERROR (Unable to decode hex hash)")
				os.Exit(1)
			}
			message := utils.Message{
				File:        file,
				Destination: destination,
				Request:     &request_bytes,
			}
			fmt.Fprintln(os.Stderr, "Sending file request")
			send(&message, GOSSIPER_ADDRESS+":"+*uiPort)
			return
		}
	}

	//flag for file indexing
	if *file != "" && *request == "" && *destination == "" && *msg == "" && *keywords == "" {
		message := utils.Message{
			File: file,
		}
		send(&message, GOSSIPER_ADDRESS+":"+*uiPort)
		return
	}

	//flag for private message / rumor message
	if *msg != "" && *file == "" && *request == "" && *keywords == "" {
		message := utils.Message{Text: *msg, Destination: destination}
		send(&message, GOSSIPER_ADDRESS+":"+*uiPort)
		return
	}
	//flag for file search
	if *msg == "" && *file == "" && *request == "" && *keywords != ""{
		var message utils.Message
		if !*gc {
			message = utils.Message{Budget : budget, Keywords: keywords}
		}else {
			message = utils.Message{GC : gc, Keywords: keywords, UseProxy: useProxy}
		}
		send(&message, GOSSIPER_ADDRESS + ":" + *uiPort)
		return
	
	}

	fmt.Println("ERROR (Bad argument combination)​")
	os.Exit(1)

}

func send(packet *utils.Message, addr string) {
	udpAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not resolve UDP address")
		return
	}
	conn, err := net.DialUDP("udp4", nil, udpAddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not connect to server %s: %s\n", addr, err)
		return
	}
	defer conn.Close()

	packetBytes, err := protobuf.Encode(packet)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	n, _ := conn.Write(packetBytes)

	fmt.Fprintln(os.Stderr,"Packet sent to " + udpAddr.String() + " size: ",n)
	return

}
