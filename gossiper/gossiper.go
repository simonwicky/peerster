package gossiper

import ("net"
		"fmt"
		"os"
		"strings"
		"github.com/dedis/protobuf"
		"github.com/simonwicky/Peerster/utils"
		"math/rand"
		"time"
		"sync"
		"strconv"
		"encoding/hex"
)


type Gossiper struct {
	//network
	addressPeer *net.UDPAddr
	connPeer *net.UDPConn
	addressClient *net.UDPAddr
	connClient *net.UDPConn

	//name + peers
	Name string
	knownPeers []string

	//status
	currentStatus utils.StatusPacket
	currentStatus_lock sync.RWMutex
	//Counter
	counter uint32
	counter_lock sync.Mutex

	//timer
	antiEntropyTicker *time.Ticker
	rTimerTicker *time.Ticker

	//rumormongers
	workers map[string] *Rumormonger
	workers_lock sync.RWMutex

	//datadownloader
	downloader map[string] *Datadownloader
	downloader_lock sync.RWMutex

	//DSDV
	DSDV map[string] string
	DSDV_lock sync.RWMutex

	//UI
	uiBuffer chan utils.GossipPacket
	latestRumors *utils.RumorKeyQueue

	//storage
	fileStorage *FileStorage
	messages map[utils.RumorMessageKey]utils.RumorMessage

}


func NewGossiper(clientAddress, address, name, peers string, antiEntropy, rtimer int) *Gossiper {
	rand.Seed(time.Now().Unix())
	udpAddrPeer, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		fmt.Fprintln(os.Stderr,"Unable to resolve UDP address")
		return nil
	}

	udpConnPeer, err := net.ListenUDP("udp4",udpAddrPeer)
	if err != nil {
		fmt.Fprintln(os.Stderr,"Unable to listen")
		fmt.Fprintln(os.Stderr,err)
		return nil
	}

	udpAddrClient, err := net.ResolveUDPAddr("udp4", clientAddress)
	if err != nil {
		fmt.Fprintln(os.Stderr,"Unable to resolve UDP address")
		return nil
	}

	udpConnClient, err := net.ListenUDP("udp4",udpAddrClient)
	if err != nil {
		fmt.Fprintln(os.Stderr,"Unable to listen")
		fmt.Fprintln(os.Stderr,err)
		return nil
	}
	var peersArray []string
	if peers != ""{
		peersArray = strings.Split(peers, ",")
	}

	if antiEntropy < 0 || antiEntropy > 100 {
		fmt.Fprintln(os.Stderr,"Anti Entropy too small or too high, fallback to 10")
		antiEntropy = 10
	}

	var antiEntropyTicker *time.Ticker
	if antiEntropy == 0 {
		fmt.Fprintln(os.Stderr,"Disabling anti entropy")
		antiEntropyTicker = nil
	} else {
		antiEntropy_duration, _ := time.ParseDuration(strconv.Itoa(antiEntropy) + "s")
		antiEntropyTicker = time.NewTicker(antiEntropy_duration)
	}

	if rtimer < 0 || rtimer > 1000 {
		fmt.Fprintln(os.Stderr,"Rtimer too small or too high, disabling...")
		rtimer = 0
	}
	var rTimerTicker *time.Ticker
	if rtimer == 0 {
		fmt.Fprintln(os.Stderr,"Route rumors disabled")
		rTimerTicker = nil
	} else {
		duration,_ := time.ParseDuration(strconv.Itoa(rtimer) + "s")
		rTimerTicker = time.NewTicker(duration)
	}
	return &Gossiper{
		addressPeer: udpAddrPeer,
		connPeer: udpConnPeer,
		addressClient: udpAddrClient,
		connClient: udpConnClient,
		Name: name,
		knownPeers: peersArray,
		counter: 1,
		messages : make(map[utils.RumorMessageKey]utils.RumorMessage,10),
		workers : make(map[string]*Rumormonger),
		DSDV : make(map[string] string),
		antiEntropyTicker : antiEntropyTicker,
		rTimerTicker : rTimerTicker,
		uiBuffer : make(chan utils.GossipPacket, 10),
		latestRumors : utils.NewRumorKeyQueue(50),
		fileStorage : NewFileStorage(),
		downloader : make(map[string]*Datadownloader),
	}
}
//================================
//STARTUP and ROUTINE functions
//================================
func (g *Gossiper) Start(simple bool){
	go g.ClientHandle(simple)
	if !simple {
		go g.antiEntropy()
	}
	go g.rumorRoute()
	go g.HttpServerHandler()
	g.PeersHandle(simple) 
}

func (g *Gossiper) antiEntropy(){
	if g.antiEntropyTicker == nil {
		return
	}
	for {
		_ = <- g.antiEntropyTicker.C
		g.currentStatus_lock.RLock()
		g.sendToRandomPeer(&utils.GossipPacket{Status : &g.currentStatus})
		g.currentStatus_lock.RUnlock()
		fmt.Fprintln(os.Stderr,"Sending antientropy")
	}
}

func (g *Gossiper) rumorRoute() {
	for {
		rumor := g.generateRumor("")
		g.sendToRandomPeer(&utils.GossipPacket{Rumor : &rumor})
		fmt.Fprintln(os.Stderr,"Sending route rumors.")
		//if no timer this is the only route rumors sent
		if g.rTimerTicker == nil {
			break
		}
		_ = <- g.rTimerTicker.C
	}
}

//================================
//ALL PURPOSE
//================================

//================================
//sending functions
//================================
func (g *Gossiper) sendToKnowPeers(exception string, packet *utils.GossipPacket){
	for _,peer := range g.knownPeers {
		if peer == exception {
			continue
		}
		g.sendToPeer(peer, packet)
	}
}
func (g *Gossiper) sendToRandomPeer(packet *utils.GossipPacket) string{
	if len(g.knownPeers) > 0 {
		nextPeerAddr := g.knownPeers[rand.Intn(len(g.knownPeers))]
		g.sendToPeer(nextPeerAddr, packet)
		return nextPeerAddr
	} else {
		fmt.Fprintln(os.Stderr,"No known peers")
		return ""
	}
}


func (g *Gossiper) sendToPeer(peer string, packet *utils.GossipPacket){
		address, err := net.ResolveUDPAddr("udp4",peer)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Unable to resolve adress " + peer)
			return
		}
		packetBytes, err := protobuf.Encode(packet)
		if err != nil {
			fmt.Fprintln(os.Stderr,"Could not serialize packet")
			return
		}
		n,err := g.connPeer.WriteToUDP(packetBytes,address)
		if n == 0 {
			fmt.Fprintln(os.Stderr,err)
		}
		fmt.Fprintln(os.Stderr,"Packet sent to " + address.String() + " size: ",n)
}

func (g *Gossiper) sendPointToPoint(packet *utils.GossipPacket, destination string){
	var hoplimit *uint32
	switch {
	case packet.Private != nil:
		hoplimit = &packet.Private.HopLimit
	case packet.DataRequest != nil:
		hoplimit = &packet.DataRequest.HopLimit
	case packet.DataReply != nil:
		hoplimit = &packet.DataReply.HopLimit
	default:
		fmt.Fprintln(os.Stderr,"Packet which should not be sent point to point, exiting")
		return
	}
	*hoplimit -= 1
	if *hoplimit <= 0 {
		fmt.Fprintln(os.Stderr,"No more hop, dropping packet")
		return
	}
	address := g.lookupDSDV(destination)
	if address == "" {
		fmt.Fprintln(os.Stderr,"Next hop not found, aborting")
		return
	}
	g.sendToPeer(address, packet)
}

//================================
//KNOWN PEERS
//================================
func (g *Gossiper) addToKnownPeers(address string) bool {
	for _, peer := range g.knownPeers {
		if peer == address {
			return false
		}
	}
	fmt.Fprintf(os.Stderr,"Adding peer %s to known peers\n", address)
	g.knownPeers = append(g.knownPeers, address)
	return true
}

//================================
//STATUS
//================================
func (g *Gossiper) updateStatus(status utils.PeerStatus, index int){
	fmt.Fprintln(os.Stderr,"Status update")
	g.currentStatus_lock.Lock()
	if index == -1 {
		g.currentStatus.Want = append(g.currentStatus.Want, status)
	} else {
		g.currentStatus.Want[index].NextID += 1
	}
	g.currentStatus_lock.Unlock()
}
//================================
//DSDV
//================================
func (g *Gossiper) updateDSDV(name,addr string){
	fmt.Fprintln(os.Stderr,"DSDV update")
	g.DSDV_lock.Lock()
	g.DSDV[name] = addr
	g.DSDV_lock.Unlock()
}
func (g *Gossiper) lookupDSDV(name string) string {
	g.DSDV_lock.RLock()
	address, ok := g.DSDV[name]
	g.DSDV_lock.RUnlock()
	if ok {
		return address
	} else {
		//Address not found, shouldn't trigger but, you never know
		return ""
	}
}

func (g *Gossiper) dumpDSDV() string {
	g.DSDV_lock.RLock()
	str := ""
	for k := range g.DSDV{
		str += k
		str += ","
	}
	g.DSDV_lock.RUnlock()
	if len(str) > 0 {
		str = str[:len(str)-1]
	}
	return str
}

//================================
//MESSAGE STORAGE
//================================
func (g *Gossiper) addMessage(rumor *utils.RumorMessage){
	key := utils.RumorMessageKey{Origin : rumor.Origin, ID : rumor.ID}
	if _, new := g.messages[key]; !new{
		g.messages[key] = *rumor
		if rumor.Text != "" {
			g.latestRumors.Push(key)		
		}
	}
}

func (g *Gossiper) getMessage(origin string, ID uint32) utils.RumorMessage {
	key := utils.RumorMessageKey{Origin: origin, ID: ID}
	msg, ok := g.messages[key]
	if ok {
		return msg
	}
	//rumor not sotred, might be a route rumor
	return utils.RumorMessage{
		Origin: origin,
		ID: ID,
		Text: "",
	}
}
//================================
//Workers
//================================
func (g *Gossiper) lookupWorkers(address string) (*Rumormonger, bool) {
	g.workers_lock.RLock()
	worker, ok := g.workers[address]
	g.workers_lock.RUnlock()
	return worker, ok
}

func (g *Gossiper) addWorker(worker *Rumormonger, address string) {
	g.workers_lock.Lock()
	g.workers[address] = worker
	g.workers_lock.Unlock()
}

func (g *Gossiper) deleteWorker(address string) {
	g.workers_lock.Lock()
	delete(g.workers, address)
	g.workers_lock.Unlock()
}

func (g *Gossiper) createAndRunWorker(address string, waitingForAck bool, currentRumor *utils.GossipPacket, bootstrapPacket *utils.GossipPacket){
	if worker, ok := g.lookupWorkers(address); ok {
		if waitingForAck {
			worker.timer = time.NewTimer(10 * time.Second)
		}
		worker.waitingForAck = waitingForAck
		worker.currentRumor = currentRumor
		if bootstrapPacket != nil {
			worker.Buffer <- *bootstrapPacket
		}
	} else {
		buffer := make(chan utils.GossipPacket, 20)
		if bootstrapPacket != nil {
			buffer <- *bootstrapPacket
		}
		worker := NewRumormonger(g, address, buffer, waitingForAck, currentRumor)
		g.addWorker(worker,address)
		go func(){
			worker.Start()
			defer g.deleteWorker(address)
		}()
	}
}
//================================
//Workers
//================================
func (g *Gossiper) addDownloader(dd *Datadownloader){
	g.downloader_lock.Lock()
	g.downloader[dd.id] = dd
	g.downloader_lock.Unlock()
	go func(){
		dd.Start()
		g.removeDownloader(dd.id) 
	}()
}

func (g *Gossiper) removeDownloader(id string) {
	g.downloader_lock.Lock()
	delete(g.downloader, id)
	g.downloader_lock.Unlock()
}
func (g *Gossiper) lookupDownloader(waitingFor []byte) *Datadownloader {
	g.downloader_lock.RLock()
	defer g.downloader_lock.RUnlock()
	for _,downloader := range g.downloader {
		if hex.EncodeToString(downloader.waitingFor) == hex.EncodeToString(waitingFor) {
			return downloader
		}
	}
	return nil
}

//================================
//NO CATEGORY
//================================
func (g *Gossiper) generateRumor(text string) utils.RumorMessage{
	var rumor utils.RumorMessage
	rumor.Origin = g.Name
	g.counter_lock.Lock()
	rumor.ID = g.counter
	g.counter += 1
	g.counter_lock.Unlock()
	rumor.Text = text
	statusIndex := -1
	for index,status := range g.currentStatus.Want {
		if status.Identifer == rumor.Origin{
			statusIndex = index
		}
	}
	g.updateStatus(utils.PeerStatus{Identifer : rumor.Origin, NextID : rumor.ID + 1}, statusIndex)
	//add the message to storage
	g.addMessage(&rumor)
	return rumor
}



