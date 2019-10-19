package utils

import ("fmt"
		"strings")

func LogRumor(packet *RumorMessage, address string){
	fmt.Printf("RUMOR origin %s from %s ID %d contents %s\n",packet.Origin,address,packet.ID,packet.Text)
}

func LogStatus(want []PeerStatus, address string){
	fmt.Printf("STATUS from %s ", address)
	for _, status := range want {
		fmt.Printf("peer %s nextID %d ", status.Identifer, status.NextID)
	}
	fmt.Printf("\n")
}

func LogSimpleMessage(packet *SimpleMessage){
	fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s\n",packet.OriginalName, packet.RelayPeerAddr, packet.Contents)
}

func LogPeers(peers []string){
	fmt.Printf("PEERS %s\n", strings.Join(peers,","))
}

func LogSync(address string){
	fmt.Printf("IN SYNC WITH %s\n",address)
}

func LogFlip(address string){
	fmt.Printf("FLIPPED COIN sending rumor to %s\n",address)
}

func LogMongering(address string){
	fmt.Printf("MONGERING with %s\n",address)
}

func LogDSDV(name, address string){
	fmt.Printf("DSDV %s %s\n",name,address)
}

func LogPrivate(packet *PrivateMessage){
	fmt.Printf("PRIVATE origin %s hop-limit %d contents %s\n",packet.Origin,packet.HopLimit,packet.Text)
}

func LogClient(text string){
	fmt.Printf("CLIENT MESSAGE %s\n",text)
}
