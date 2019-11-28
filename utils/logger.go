package utils

import ("fmt"
		"strings"
		"encoding/hex")

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

func LogMetafile(filename, peer string){
	fmt.Printf("DOWNLOADING metafile of %s from %s\n",filename,peer)
}

func LogChunk(filename, peer string, index int){
	fmt.Printf("DOWNLOADING %s chunk %d from %s\n",filename,index,peer)
}

func LogReconstruct(filename string) {
	fmt.Printf("RECONSTRUCTED file %s\n",filename)
}
func LogSearchFinished(){
	fmt.Printf("SEARCH FINISHED\n");
}
func LogFileFound(name, peer, metafile string, chunkMap []uint64){
	fmt.Printf("FOUND match %s at %s\n",name,peer)
	fmt.Printf("metafile=%s chunks=",metafile)
	for index,n := range chunkMap {
		fmt.Printf("%d",n)
		if index != len(chunkMap)-1 {
			fmt.Printf(",")
		}
	}
	fmt.Printf("\n")

}

func LogTLCGossip(message *TLCMessage){
	if message.Confirmed != -1{
		fmt.Printf("CONFIRMED ")
	} else {
		fmt.Printf("UNCONFIRMED ")
	}
	fmt.Printf("GOSSIP origin %s ID %d file name %s size %d metahash %s\n",message.Origin, message.ID, message.TxBlock.Transaction.Name,message.TxBlock.Transaction.Size,hex.EncodeToString(message.TxBlock.Transaction.MetafileHash))
}

func LogConfirmedID(id uint32, witnesses []string){
	fmt.Printf("RE-BROADCAST ID %d WITNESSES %s\n", id, strings.Join(witnesses, ","))
}

func LogAck(origin string, id uint32){
	fmt.Printf("SENDING ACK origin %s ID %d\n", origin,id)
}

func LogNextRound(id uint32,msgs []*TLCMessage){
	fmt.Printf("ADVANCING TO round ​%d BASED ON CONFIRMED MESSAGES ",id)
	for index,msg := range msgs {
		fmt.Printf("origin%d %s ID%d %d",index+1,msg.Origin, index+1, msg.ID)
		if index != len(msgs)-1 {
			fmt.Printf(", ")
		}
	}
	fmt.Printf("\n")
}

