package utils

import (
	"fmt"
	"strings"
	"encoding/hex"
)

func LogRumor(packet *RumorMessage, address string){
	//fmt.Printf("RUMOR origin %s from %s ID %d contents %s\n",packet.Origin,address,packet.ID,packet.Text)
}

func LogStatus(want []PeerStatus, address string){
	/*fmt.Printf("STATUS from %s ", address)
	for _, status := range want {
		fmt.Printf("peer %s nextID %d ", status.Identifer, status.NextID)
	}
	fmt.Printf("\n")*/
}

func LogSimpleMessage(packet *SimpleMessage){
	//fmt.Printf("SIMPLE MESSAGE origin %s from %s contents %s\n",packet.OriginalName, packet.RelayPeerAddr, packet.Contents)
}

func LogPeers(peers []string){
	//fmt.Printf("PEERS %s\n", strings.Join(peers,","))
}

func LogSync(address string){
	//fmt.Printf("IN SYNC WITH %s\n",address)
}

func LogFlip(address string){
	//fmt.Printf("FLIPPED COIN sending rumor to %s\n",address)
}

func LogMongering(address string){
	//fmt.Printf("MONGERING with %s\n",address)
}

func LogDSDV(name, address string){
	//fmt.Printf("DSDV %s %s\n",name,address)
}

func LogPrivate(packet *PrivateMessage){
	//fmt.Printf("PRIVATE origin %s hop-limit %d contents %s\n",packet.Origin,packet.HopLimit,packet.Text)
}

func LogClient(text string){
	//fmt.Printf("CLIENT MESSAGE %s\n",text)
}

func LogMetafile(filename, peer string){
	//fmt.Printf("DOWNLOADING metafile of %s from %s\n",filename,peer)
}

func LogChunk(filename, peer string, index int){
	//fmt.Printf("DOWNLOADING %s chunk %d from %s\n",filename,index,peer)
}

func LogReconstruct(filename string) {
	//fmt.Printf("RECONSTRUCTED file %s\n",filename)
}
func LogSearchFinished(){
	//fmt.Printf("SEARCH FINISHED\n");
}
func LogFileFound(name, peer, metafile string, chunkMap []uint64) {
	/*fmt.Printf("FOUND match %s at %s\n", name, peer)
	fmt.Printf("metafile=%s chunks=", metafile)
	for index, n := range chunkMap {
		fmt.Printf("%d", n)
		if index != len(chunkMap)-1 {
			fmt.Printf(",")
		}
	}
	fmt.Printf("\n")*/

}

func LogTLCGossip(message *TLCMessage) {
	/*if message.Confirmed != -1 {
		fmt.Printf("CONFIRMED ")
	} else {
		fmt.Printf("UNCONFIRMED ")
	}
	fmt.Printf("GOSSIP origin %s ID %d file name %s size %d metahash %s\n", message.Origin, message.ID, message.TxBlock.Transaction.Name, message.TxBlock.Transaction.Size, hex.EncodeToString(message.TxBlock.Transaction.MetafileHash))*/
}

func LogConfirmedID(id uint32, witnesses []string) {
	//fmt.Printf("RE-BROADCAST ID %d WITNESSES %s\n", id, strings.Join(witnesses, ","))
}

func LogAck(origin string, id uint32) {
	//fmt.Printf("SENDING ACK origin %s ID %d\n", origin, id)
}

func LogNextRound(id uint32, msgs []*TLCMessage) {
	/*fmt.Printf("ADVANCING TO round ​%d BASED ON CONFIRMED MESSAGES ", id)
	for index, msg := range msgs {
		fmt.Printf("origin%d %s ID%d %d", index+1, msg.Origin, index+1, msg.ID)
		if index != len(msgs)-1 {
			fmt.Printf(", ")
		}
	}
	fmt.Printf("\n")*/
}

func LogConsensus(id uint32, msg *TLCMessage, nameList []string) {
	/*fmt.Printf("CONSENSUS ON QSC round %d message origin %s ID %d ​", id, msg.Origin, msg.ID)
	fmt.Printf("file names ")
	fmt.Printf(strings.Join(nameList, " "))
	fmt.Printf(" size %d metahash %s\n", msg.TxBlock.Transaction.Size, hex.EncodeToString(msg.TxBlock.Transaction.MetafileHash))*/
}

func LogGCSearchReply(reply *GCSearchReply){
	fmt.Printf("Log Garlic Cast Search Reply from %s with ID %dAccessible files:\n", reply.Origin, reply.ID)
	for i, file := range reply.AccessibleFiles {
		fmt.Printf("%d) %s %s\n", i + 1, file.FileName, hex.EncodeToString(file.MetafileHash))
	}
	fmt.Printf("\n")
}

/*
Log prints messages to the std output prepended by a [DEBUG] flag
*/
func Log(msg ...interface{}) {
	//fmt.Println("[\033[0;36mDEBUG\033[0m]", fmt.Sprint(msg...))
}

/*
Logger contrarily to glog and Logger aims at having a backend for console
on the client side and backdoor for testing
*/
type Logger struct {
	warnings bool
	debugs   bool
	fatals   bool
	named    bool
	name     string
	filtered bool
	filter   map[string]bool
}

var (
	/*
		LogObj is the Base logger instance aimed at being mutated to suit needs
	*/
	LogObj = defaultLogger()
)

func (logger *Logger) getName() string {
	return logger.name
}

func (logger *Logger) getname() string {
	if logger.named {
		return fmt.Sprintf("\033[1;38;5;%dm<%s>\033[0m", 225, logger.name)
	}
	return ""
}

/*
Debug logs debug statement to the stdout
*/
func (logger *Logger) Debug(msg ...interface{}) {
	if logger.debugs {
		if !logger.filtered {
			fmt.Println("[\033[0;36mDEBUG\033[0m]", logger.getname(), fmt.Sprint(msg...))
		} else {
			if _, ok := logger.filter[logger.name]; ok {
				fmt.Println("[\033[0;36mDEBUG\033[0m]", logger.getname(), fmt.Sprint(msg...))
			}
		}
	}
}

func (logger *Logger) Warn(msg ...interface{}) {
	if logger.warnings {
		fmt.Println("[\033[1;33mWARN\033[0m]", fmt.Sprint(msg...))
	}
}

func (logger *Logger) Fatal(msg ...interface{}) {
	if logger.fatals {
		fmt.Println("[\033[1;31mFATAL\033[0m]", fmt.Sprint(msg...))
	}
}

func (logger *Logger) Named(name string) *Logger {
	copy := *logger
	copy.name = name //fmt.Sprintf("<\033[1;%dm%s\033[0m>", logger.nameColor, name)
	copy.named = true
	return &copy
}

func (logger *Logger) Filter(filters string) {
	for _, filter := range strings.Split(filters, ",") {
		logger.filter[filter] = true
	}
	logger.filtered = true
}

//func log(msg ...interface{}, )

func defaultLogger() *Logger {
	return &Logger{
		warnings: true,
		debugs:   true,
		fatals:   true,
		name:     "default",
		named:    false,
		filter:   make(map[string]bool),
		filtered: false,
	}
}
