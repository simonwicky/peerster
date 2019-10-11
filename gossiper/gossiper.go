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
)


type Gossiper struct {
	addressPeer *net.UDPAddr
	connPeer *net.UDPConn
	addressClient *net.UDPAddr
	connClient *net.UDPConn
	Name string
	knownPeers []string
	currentStatus utils.StatusPacket
	currentStatus_lock sync.RWMutex
	counter uint32
	counter_lock sync.Mutex
	messages map[utils.RumorMessageKey]utils.RumorMessage
	ticker *time.Ticker
	workers map[string] *Rumormonger
	uiBuffer chan utils.GossipPacket
	latestRumors *utils.RumorKeyQueue

}


func NewGossiper(clientAddress, address, name, peers string, antiEntropy int) *Gossiper {
	rand.Seed(time.Now().Unix())
	udpAddrPeer, err := net.ResolveUDPAddr("udp4", address)
	if err != nil {
		fmt.Fprintln(os.Stderr,"Unable to resolve UDP address")
		return nil
	}

	udpConnPeer, err := net.ListenUDP("udp4",udpAddrPeer)
	if err != nil {
		fmt.Fprintln(os.Stderr,"Unable to listen")
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
		return nil
	}
	var peersArray []string
	if peers != ""{
		peersArray = strings.Split(peers, ",")
	}

	if antiEntropy <= 0 || antiEntropy > 100 {
		fmt.Fprintln(os.Stderr,"Anti Entropy too small or too high, fallback to 10")
		antiEntropy = 10
	}
	antiEntropy_duration, _ := time.ParseDuration(strconv.Itoa(antiEntropy) + "s")

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
		ticker : time.NewTicker(antiEntropy_duration),
		uiBuffer : make(chan utils.GossipPacket, 10),
		latestRumors : utils.NewRumorKeyQueue(50),
	}
}

func (g *Gossiper) Start(simple bool){
	go g.ClientHandle(simple)
	if !simple {
		go g.antiEntropy()
	}
	//go g.HttpServerHandler()
	g.PeersHandle(simple) 
}

func (g *Gossiper) antiEntropy(){
	for {
		_ = <- g.ticker.C
		g.currentStatus_lock.RLock()
		g.sendToRandomPeer(&utils.GossipPacket{Status : &g.currentStatus})
		g.currentStatus_lock.RUnlock()
		fmt.Fprintln(os.Stderr,"Sending antientropy")
	}
}

//================================
//ALL PURPOSE
//================================


func (g *Gossiper) addToKnownPeers(address string) bool {
	fmt.Fprintf(os.Stderr,"Adding peer %s to known peers\n", address)
	for _, peer := range g.knownPeers {
		if peer == address {
			return false
		}
	}
	g.knownPeers = append(g.knownPeers, address)
	return true
}

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
		n,_ := g.connPeer.WriteToUDP(packetBytes,address)
		fmt.Fprintln(os.Stderr,"Packet sent to " + address.String() + " size: ",n)
}

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

func (g *Gossiper) addMessage(rumor *utils.RumorMessage){
	key := utils.RumorMessageKey{Origin : rumor.Origin, ID : rumor.ID}
	g.messages[key] = *rumor
	g.latestRumors.Push(key)
}

//================================================================
//PEERS SIDE
//================================================================

//loop handling the peer side
func (g *Gossiper) PeersHandle(simple bool){
		fmt.Fprintln(os.Stderr,"Listening on " + g.addressPeer.String())
		var packetBytes []byte = make([]byte, 1024)	
		for {
			var packet utils.GossipPacket
			n,address,err := g.connPeer.ReadFromUDP(packetBytes)
			if err != nil {
				fmt.Fprintln(os.Stderr,"Error!")
				return
			}
			if n > 0 {
				err = protobuf.Decode(packetBytes[:n], &packet)
				if err != nil {
					fmt.Fprintln(os.Stderr,err.Error())
				}
				if simple {
					g.peersSimpleMessageHandler(&packet)
				} else {
					if worker, ok := g.workers[address.String()]; ok {
						worker.Buffer <- *utils.CopyGossipPacket(&packet)
					} else {
						worker = NewRumormonger(g, address.String(), make(chan utils.GossipPacket, 20))
						g.workers[address.String()] = worker
						new := *utils.CopyGossipPacket(&packet)
						worker.Buffer <- new 
						go func(){
							worker.Start()
							defer delete(g.workers, address.String())
						}()
					}
				}
			}
		}
}

func (g *Gossiper) peersSimpleMessageHandler(packet *utils.GossipPacket) {

	utils.LogSimpleMessage(packet.Simple)
	relayPeer := packet.Simple.RelayPeerAddr
	packet.Simple.RelayPeerAddr = g.addressPeer.String()
	g.addToKnownPeers(relayPeer)
	utils.LogPeers(g.knownPeers)
	g.sendToKnowPeers(relayPeer, packet)
}

