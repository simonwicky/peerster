package gossiper

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/simonwicky/Peerster/utils"
	"go.dedis.ch/protobuf"
)

type Settings struct {
	SessionKeySize uint
	Buffering      uint
	Redundancy     uint          // number of cloves sent for any threshold
	DiscoveryRate  time.Duration // period of proxy discovery
}

type Gossiper struct {
	//network
	addressPeer   *net.UDPAddr
	connPeer      *net.UDPConn
	addressClient *net.UDPAddr
	connClient    *net.UDPConn

	//name + peers
	Name       string
	knownPeers []string

	//status
	currentStatus      utils.StatusPacket
	currentStatus_lock sync.RWMutex

	//Counter
	rumor_counter          uint32
	rumor_counter_lock     sync.Mutex
	TLC_counter            uint32
	TLC_counter_lock       sync.Mutex
	TLC_round_counter      uint32
	TLC_round_counter_lock sync.Mutex

	//timer
	antiEntropyTicker *time.Ticker
	rTimerTicker      *time.Ticker

	//rumormongers
	workers      map[string]*Rumormonger
	workers_lock sync.RWMutex

	//datadownloader
	downloader      map[string]*Datadownloader
	downloader_lock sync.RWMutex

	//filereplier
	searchreplier      map[string][]*SearchReplier
	searchreplier_lock sync.RWMutex

	//DSDV
	DSDV      map[string]string
	DSDV_lock sync.RWMutex

	//UI
	uiBuffer     chan utils.GossipPacket
	latestRumors *utils.RumorKeyQueue

	//storage
	fileStorage *FileStorage
	messages    map[utils.RumorMessageKey]*utils.GossipPacket
	tlcStorage  *TLCstorage

	//publishers
	publishers      map[uint32]*TLCPublisher
	publisherBuffer []*utils.FileInfo

	//search file
	fileSearcher *FileSearcher

	//clovestorage
	//secretSharer *SecretSharer

	//constant value
	hopLimit        uint32
	stubbornTimeout time.Duration
	peersNumber     uint32
	hw3ex2          bool
	hw3ex3          bool
	hw3ex4          bool

	//blockChain
	blockChain []*utils.BlockPublish
	consensus  *Consensus

	//cloves; can only receive one clove from particular predecessor for a particular sequence number
	cloves          map[string]map[string]*utils.Clove
	newProxies      chan *Proxy
	settings        *Settings
	proxyPool       *ProxyPool
	clovesCollector *ClovesCollector
}

func NewGossiper(clientAddress, address, name, peers string, antiEntropy, rtimer, hoplimit, peersNumber, stubbornTimeout int, hw3ex2, hw3ex3, hw3ex4 bool) *Gossiper {
	rand.Seed(time.Now().Unix())
	udpAddrPeer, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to resolve UDP address")
		return nil
	}

	udpConnPeer, err := net.ListenUDP("udp4", udpAddrPeer)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to listen")
		fmt.Fprintln(os.Stderr, err)
		return nil
	}

	udpAddrClient, err := net.ResolveUDPAddr("udp4", clientAddress)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to resolve UDP address")
		return nil
	}

	udpConnClient, err := net.ListenUDP("udp4", udpAddrClient)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to listen")
		fmt.Fprintln(os.Stderr, err)
		return nil
	}

	if name == "" {
		fmt.Fprintln(os.Stderr, "Name must be specified")
		return nil
	}

	var peersArray []string
	if peers != "" {
		peersArray = strings.Split(peers, ",")
	}

	if antiEntropy < 0 || antiEntropy > 100 {
		fmt.Fprintln(os.Stderr, "Anti Entropy too small or too high, fallback to 10")
		antiEntropy = 10
	}

	var antiEntropyTicker *time.Ticker
	if antiEntropy == 0 {
		fmt.Fprintln(os.Stderr, "Disabling anti entropy")
		antiEntropyTicker = nil
	} else {
		antiEntropy_duration, _ := time.ParseDuration(strconv.Itoa(antiEntropy) + "s")
		antiEntropyTicker = time.NewTicker(antiEntropy_duration)
	}

	if rtimer < 0 || rtimer > 1000 {
		fmt.Fprintln(os.Stderr, "Rtimer too small or too high, disabling...")
		rtimer = 0
	}
	var rTimerTicker *time.Ticker
	if rtimer == 0 {
		//fmt.Fprintln(os.Stderr, "Route rumors disabled")
		rTimerTicker = nil
	} else {
		duration, _ := time.ParseDuration(strconv.Itoa(rtimer) + "s")
		rTimerTicker = time.NewTicker(duration)
	}

	if peersNumber == 0 {
		fmt.Println(peers)
		fmt.Fprintln(os.Stderr, "Number of peers should be > 0")
		return nil
	}

	g := &Gossiper{
		addressPeer:       udpAddrPeer,
		connPeer:          udpConnPeer,
		addressClient:     udpAddrClient,
		connClient:        udpConnClient,
		Name:              name,
		knownPeers:        peersArray,
		rumor_counter:     0,
		TLC_counter:       0,
		messages:          make(map[utils.RumorMessageKey]*utils.GossipPacket, 10),
		workers:           make(map[string]*Rumormonger),
		DSDV:              make(map[string]string),
		antiEntropyTicker: antiEntropyTicker,
		rTimerTicker:      rTimerTicker,
		uiBuffer:          make(chan utils.GossipPacket, 10),
		latestRumors:      utils.NewRumorKeyQueue(50),
		fileStorage:       NewFileStorage(),
		tlcStorage:        NewTLCstorage(),
		downloader:        make(map[string]*Datadownloader),
		searchreplier:     make(map[string][]*SearchReplier),
		publishers:        make(map[uint32]*TLCPublisher),
		fileSearcher:      nil,
		hopLimit:          uint32(hoplimit),
		peersNumber:       uint32(peersNumber),
		stubbornTimeout:   time.Duration(stubbornTimeout),
		hw3ex2:            hw3ex2,
		hw3ex3:            hw3ex3,
		hw3ex4:            hw3ex4,
		consensus:         NewConsensus(),
		settings: &Settings{
			SessionKeySize: 32,
			Buffering:      10,
			Redundancy:     10,
			DiscoveryRate:  8,
		},
		newProxies: make(chan *Proxy),
		proxyPool:  &ProxyPool{proxies: make([]*Proxy, 0)},
		//secretSharer : NewSecretSharer(),
	}
	g.clovesCollector = NewClovesCollector(g)
	return g
}

/*type Proxy struct {
	Paths     [2]*net.UDPAddr
	PublicKey crypto.PublicKey
}*/

func getTuple(n uint, pathsTaken map[string]bool, peers []string) ([]string, map[string]bool, error) {
	//shuffle known peers
	rand.Shuffle(len(peers), func(i, j int) {
		tmp := peers[i]
		peers[i] = peers[j]
		peers[j] = tmp
	})
	tuple := make([]string, n)
	var i uint = 0
	for _, peer := range peers {
		if taken, ok := pathsTaken[peer]; !ok || !taken {
			tuple[i] = peer
			i++
			pathsTaken[peer] = true
			if i >= n {
				return tuple, pathsTaken, nil
			}
		}
	}
	return tuple[:i], pathsTaken, fmt.Errorf("Not enough available paths!")
}

/*
Proxy describes a proxy in an abstract manner
*/
type Proxy struct {
	Paths      [2]string
	SessionKey []byte
	ProxySN    []byte
}

/*
ProxyPool is a thread-safe store for proxies with convenience methods
*/
type ProxyPool struct {
	sync.RWMutex
	proxies []*Proxy
}

/*
Cover returns a map of all the paths taken by the pool's proxies in aggregate
*/
func (pool *ProxyPool) Cover() map[string]bool {
	pathsTaken := map[string]bool{}
	for _, proxy := range pool.proxies {
		pathsTaken[proxy.Paths[0]] = true
		pathsTaken[proxy.Paths[1]] = true
	}
	return pathsTaken
}

/*
Add adds a new proxy to the pool
*/
func (pool *ProxyPool) Add(proxy *Proxy) {
	pool.Lock()
	defer pool.Unlock()
	pool.proxies = append(pool.proxies, proxy)
}

/*
GetD returns d random proxies from the ProxyPool
*/
func (pool *ProxyPool) GetD(d uint) ([]*Proxy, error) {
	pool.Lock()
	defer pool.Unlock()
	if uint(len(pool.proxies)) < d {
		return nil, errors.New("could not find d proxies")
	}
	rand.Shuffle(len(pool.proxies), func(i, j int) {
		tmp := pool.proxies[i]
		pool.proxies[i] = pool.proxies[j]
		pool.proxies[j] = tmp
	})
	return pool.proxies[:d], nil
}

/*
initiate creates a new proxy init message,
splits it in n cloves, gets n paths from the known peers
and sends a clove to each path
*/
func (g *Gossiper) initiate(n uint, knownPeers []string, pathsTaken map[string]bool) map[string]bool {
	tuple, pathsStillAvailable, err := getTuple(n, pathsTaken, knownPeers)
	if err == nil {
		cloves, err := utils.NewProxyInit().Split(2, g.settings.Redundancy)
		//test
		if err == nil {
			for i, clove := range cloves {
				utils.LogObj.Named("init").Debug("sending to ", tuple[i], clove.Data)
				g.sendToPeer(tuple[i], clove.Wrap())
			}
		} else {
			utils.LogObj.Fatal(err.Error())
		}
	} else {
		utils.LogObj.Warn(err.Error())
	}
	return pathsStillAvailable
}

/*
initiator maintains a proxy pool by periodically trying to discover new ones
BIG QUESTION: is it enough to take distincts pairs or do _ALL_ the paths have to be distinct
	- distinct pairs:
	- distinct paths:

	let's assume distinct paths(one path = one and only one proxy)
*/
func (g *Gossiper) initiator(n uint, peersAtBootstrap []string, peersUpdates chan []string) {
	logger := utils.LogObj.Named("init")
	knownPeers := peersAtBootstrap
	pathsTaken := map[string]bool{}
	pool := g.proxyPool
	g.initiate(n, knownPeers, pathsTaken)
	ini := time.NewTicker(time.Second * g.settings.DiscoveryRate)
	for {
		select {
		case <-ini.C:
			pathsTaken = pool.Cover()
			//initiate proxy search
			pathsTaken = g.initiate(n, knownPeers, pathsTaken)
		case peersUpdate := <-peersUpdates:
			knownPeers = peersUpdate
		case newProxy := <-g.newProxies:
			logger.Debug("New proxy")
			pool.Add(newProxy)
			sessionKey := make([]byte, g.settings.SessionKeySize)
			rand.Read(sessionKey) // always return nil error per documentation
			cloves, err := utils.NewProxyAck(sessionKey).Split(2, 2)
			if err == nil {
				for i, clove := range cloves {
					logger.Debug("sent ack ", string(clove.SequenceNumber), " to ", newProxy.Paths[i])
					g.sendToPeer(newProxy.Paths[i], clove.Wrap())
				}
			} else {
				utils.LogObj.Fatal(err.Error())
			}
		}
	}
}

//================================
//STARTUP and ROUTINE functions
//================================
func (g *Gossiper) Start(simple bool, port string) {
	go g.ClientHandle(simple)
	if !simple {
		go g.antiEntropy()
	}
	go g.rumorRoute()
	go g.HttpServerHandler(port)
	go g.initiator(3, g.knownPeers, nil)
	g.PeersHandle(simple)
}

func (g *Gossiper) antiEntropy() {
	if g.antiEntropyTicker == nil {
		return
	}
	for {
		_ = <-g.antiEntropyTicker.C
		g.currentStatus_lock.RLock()
		g.sendToRandomPeer(&utils.GossipPacket{Status: &g.currentStatus})
		g.currentStatus_lock.RUnlock()
		//fmt.Fprintln(os.Stderr, "Sending antientropy")
	}
}

func (g *Gossiper) rumorRoute() {
	if g.rTimerTicker == nil {
		return
	}
	for {
		rumor := g.generateRumor("")
		g.sendToRandomPeer(&utils.GossipPacket{Rumor: &rumor})
		//fmt.Fprintln(os.Stderr, "Sending route rumors.")
		_ = <-g.rTimerTicker.C
	}
}

//================================
//ALL PURPOSE
//================================

//================================
//sending functions
//================================
func (g *Gossiper) sendToKnownPeers(exception string, packet *utils.GossipPacket) {
	for _, peer := range g.knownPeers {
		if peer == exception {
			continue
		}
		g.sendToPeer(peer, packet)
	}
}
func (g *Gossiper) sendToRandomPeer(packet *utils.GossipPacket) string {
	if len(g.knownPeers) > 0 {
		nextPeerAddr := g.knownPeers[rand.Intn(len(g.knownPeers))]
		g.sendToPeer(nextPeerAddr, packet)
		return nextPeerAddr
	} else {
		//fmt.Fprintln(os.Stderr, "No known peers")
		return ""
	}
}

func (g *Gossiper) sendToPeer(peer string, packet *utils.GossipPacket) {
	address, err := net.ResolveUDPAddr("udp4", peer)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Unable to resolve adress "+peer)
		return
	}
	packetBytes, err := protobuf.Encode(packet)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	n, err := g.connPeer.WriteToUDP(packetBytes, address)
	if n == 0 {
		fmt.Fprintln(os.Stderr, err)
	}
	//fmt.Fprintln(os.Stderr, "Packet sent to "+address.String()+" size: ", n)
}

func (g *Gossiper) sendPointToPoint(packet *utils.GossipPacket, destination string) {
	var hoplimit *uint32
	switch {
	case packet.Private != nil:
		hoplimit = &packet.Private.HopLimit
	case packet.DataRequest != nil:
		hoplimit = &packet.DataRequest.HopLimit
	case packet.DataReply != nil:
		hoplimit = &packet.DataReply.HopLimit
	case packet.SearchReply != nil:
		hoplimit = &packet.SearchReply.HopLimit
	case packet.Ack != nil:
		hoplimit = &packet.Ack.HopLimit
	default:
		fmt.Fprintln(os.Stderr, "Packet which should not be sent point to point, exiting")
		return
	}
	if *hoplimit <= 0 {
		fmt.Fprintln(os.Stderr, "No more hop, dropping packet")
		return
	}
	*hoplimit -= 1
	address := g.lookupDSDV(destination)
	if destination == g.Name {
		address = g.addressPeer.String()
	}
	if address == "" {
		fmt.Fprintln(os.Stderr, "Next hop not found, aborting")
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
	//fmt.Fprintf(os.Stderr, "Adding peer %s to known peers\n", address)
	g.knownPeers = append(g.knownPeers, address)
	return true
}

//================================
//STATUS
//================================
func (g *Gossiper) updateStatus(status utils.PeerStatus, index int) {
	fmt.Fprintln(os.Stderr, "Status update")
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
func (g *Gossiper) updateDSDV(name, addr string) {
	if name != g.Name {
		fmt.Fprintln(os.Stderr, "DSDV update")
		g.DSDV_lock.Lock()
		g.DSDV[name] = addr
		g.DSDV_lock.Unlock()
	}
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
	for k := range g.DSDV {
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
func (g *Gossiper) addMessage(packet *utils.GossipPacket) {
	tlc := packet.TLCMessage
	rumor := packet.Rumor
	var key utils.RumorMessageKey
	if rumor != nil {
		key = utils.RumorMessageKey{Origin: rumor.Origin, ID: rumor.ID}
	} else if tlc != nil {
		key = utils.RumorMessageKey{Origin: tlc.Origin, ID: tlc.ID}
	}
	if _, new := g.messages[key]; !new {
		g.messages[key] = packet
		if rumor != nil && rumor.Text != "" {
			g.latestRumors.Push(key)
		}
	}
}

func (g *Gossiper) getMessage(origin string, ID uint32) *utils.GossipPacket {
	key := utils.RumorMessageKey{Origin: origin, ID: ID}
	packet, ok := g.messages[key]
	if ok {
		return packet
	}
	//rumor not stored, might be a route rumor
	rumor := utils.RumorMessage{
		Origin: origin,
		ID:     ID,
		Text:   "",
	}
	return &utils.GossipPacket{
		Rumor: &rumor,
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

func (g *Gossiper) createAndRunWorker(address string, waitingForAck bool, currentRumor *utils.GossipPacket, bootstrapPacket *utils.GossipPacket) {
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
		g.addWorker(worker, address)
		go func() {
			worker.Start()
			defer g.deleteWorker(address)
		}()
	}
}

//================================
//Downloader
//================================
func (g *Gossiper) addDownloader(dd *Datadownloader) {
	g.downloader_lock.Lock()
	g.downloader[dd.id] = dd
	g.downloader_lock.Unlock()
	go func() {
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
	for _, downloader := range g.downloader {
		if hex.EncodeToString(downloader.waitingFor) == hex.EncodeToString(waitingFor) {
			return downloader
		}
	}
	return nil
}

//================================
//SearchReplier
//================================
func (g *Gossiper) addSearchReplier(sr *SearchReplier) {
	g.searchreplier_lock.Lock()
	g.searchreplier[sr.origin] = append(g.searchreplier[sr.origin], sr)
	g.searchreplier_lock.Unlock()
}

func (g *Gossiper) lookupSearchRequest(origin string, keywords []string) bool {
	g.searchreplier_lock.Lock()
	defer g.searchreplier_lock.Unlock()
	for _, replier := range g.searchreplier[origin] {
		if utils.ArrayEquals(replier.keywords, keywords) {
			return true
		}
	}
	return false
}

func (g *Gossiper) removeSearchReplier(sr *SearchReplier) {
	g.searchreplier_lock.Lock()
	defer g.searchreplier_lock.Unlock()
	for index, replier := range g.searchreplier[sr.origin] {
		if utils.ArrayEquals(replier.keywords, sr.keywords) {
			g.searchreplier[sr.origin] = append(g.searchreplier[sr.origin][:index], g.searchreplier[sr.origin][index+1:]...)
		}
	}
}

//================================
//File Searcher
//================================
func (g *Gossiper) getFileSearcher() *FileSearcher {
	if g.fileSearcher == nil {
		g.fileSearcher = NewFileSearcher(g)
	}
	return g.fileSearcher
}

func (g *Gossiper) deleteFileSearcher() {
	g.fileSearcher = nil
}

//==============================
//TLCPublisher
//==============================
func (g *Gossiper) addPublisher(p *TLCPublisher) {
	g.publishers[p.id] = p
}

func (g *Gossiper) deletePublisher(id uint32) {
	g.publishers[id] = nil
}

func (g *Gossiper) lookupPublisher(id uint32) *TLCPublisher {
	p, ok := g.publishers[id]
	if ok {
		return p
	}
	return nil
}

func (g *Gossiper) checkPublisher(roundID uint32) *TLCPublisher {
	for _, p := range g.publishers {
		if p != nil && p.roundID == roundID {
			return p
		}
	}
	return nil
}

func (g *Gossiper) bufferInfos(infos *utils.FileInfo) {
	g.publisherBuffer = append(g.publisherBuffer, infos)
}

func (g *Gossiper) getNextPublishInfos() *utils.FileInfo {
	if len(g.publisherBuffer) == 0 {
		return nil
	}
	infos := g.publisherBuffer[0]
	g.publisherBuffer = g.publisherBuffer[1:]
	return infos
}

//==============================
//Blcokchain
//==============================
func (g *Gossiper) getLastHash() [32]byte {
	if len(g.blockChain) == 0 {
		return [32]byte{}
	}
	return g.blockChain[len(g.blockChain)-1].Hash()
}

func (g *Gossiper) addBlock(block *utils.BlockPublish) {
	g.blockChain = append(g.blockChain, block)
}

func (g *Gossiper) checkBlockValidity(block *utils.BlockPublish) bool {
	if !g.hw3ex4 {
		return true
	}
	for _, name := range g.dumpBlockChain() {
		if block.Transaction.Name == name {
			return false
		}
	}

	lastHash := g.getLastHash()
	if hex.EncodeToString(block.PrevHash[:]) != hex.EncodeToString(lastHash[:]) {
		return false
	}
	return true
}
func (g *Gossiper) dumpBlockChain() []string {
	result := []string{}
	for _, block := range g.blockChain {
		result = append(result, block.Transaction.Name)
	}
	return result
}

//==============================
//SecretSharer
//==============================

// func (g *Gossiper) addClove(clove *utils.Cloves) {
// 	g.secretSharer.cloves[clove.Id] = append(g.secretSharer.cloves[clove.Id], clove)
// 	go g.secretSharer.checkSecret(clove.Id)
// }

//================================
//NO CATEGORY
//================================
func (g *Gossiper) generateRumor(text string) utils.RumorMessage {
	var rumor utils.RumorMessage
	rumor.Origin = g.Name
	rumor.ID = g.getRumorID()
	rumor.Text = text
	statusIndex := -1
	for index, status := range g.currentStatus.Want {
		if status.Identifer == rumor.Origin {
			statusIndex = index
		}
	}
	g.updateStatus(utils.PeerStatus{Identifer: rumor.Origin, NextID: rumor.ID + 1}, statusIndex)
	//add the message to storage
	g.addMessage(&utils.GossipPacket{Rumor: &rumor})
	return rumor
}

func (g *Gossiper) getRumorID() uint32 {
	g.rumor_counter_lock.Lock()
	g.TLC_counter_lock.Lock()
	g.rumor_counter += 1
	id := g.rumor_counter
	id += g.TLC_counter
	g.TLC_counter_lock.Unlock()
	g.rumor_counter_lock.Unlock()
	return id
}

func (g *Gossiper) getTLCID() uint32 {
	g.rumor_counter_lock.Lock()
	g.TLC_counter_lock.Lock()
	g.TLC_counter += 1
	id := g.TLC_counter
	id += g.rumor_counter
	g.TLC_counter_lock.Unlock()
	g.rumor_counter_lock.Unlock()
	return id
}

func (g *Gossiper) getTLCRound() uint32 {
	g.TLC_round_counter_lock.Lock()
	id := g.TLC_round_counter
	g.TLC_round_counter_lock.Unlock()
	return id
}

func (g *Gossiper) incrementTLCRound() {
	g.TLC_round_counter_lock.Lock()
	g.TLC_round_counter += 1
	g.TLC_round_counter_lock.Unlock()
}
