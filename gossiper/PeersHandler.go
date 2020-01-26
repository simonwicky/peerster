package gossiper

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/simonwicky/Peerster/utils"
	"go.dedis.ch/protobuf"
)

//================================================================
//PEER SIDE
//================================================================

//loop handling the peer side
func (g *Gossiper) PeersHandle(simple bool) {
	//fmt.Fprintln(os.Stderr, "Listening on "+g.addressPeer.String())
	var packetBytes []byte = make([]byte, 10000)
	for {
		var packet utils.GossipPacket
		n, address, err := g.connPeer.ReadFromUDP(packetBytes)
		g.addToKnownPeers(address.String())
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error!")
			return
		}
		if n > 0 {
			err = protobuf.Decode(packetBytes[:n], &packet)
			if err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
			}
			switch {
			case simple:
				g.peerSimpleMessageHandler(&packet)
			case packet.Private != nil:
				g.peerPrivateMessageHandler(&packet)
			case packet.DataRequest != nil:
				g.peerDataRequestHandler(&packet)
			case packet.DataReply != nil:
				g.peerDataReplyHandler(&packet)
			case packet.SearchRequest != nil:
				g.peerSearchRequestHandler(&packet)
			case packet.SearchReply != nil:
				g.peerSearchReplyHandler(&packet)
			case packet.Rumor != nil || packet.Status != nil:
				g.peerRumorStatusHandler(&packet, address.String())
			case packet.TLCMessage != nil:
				g.peerTLCMessageHandler(&packet, address.String())
			case packet.Ack != nil:
				g.peerTLCAckHandler(&packet)
			case packet.Clove != nil:
				g.clovesCollector.handler <- IncomingClove{predecessor: address.String(), clove: packet.Clove}
			default:
				fmt.Fprintln(os.Stderr, "Message unknown, dropping packet")
			}
		}
	}
}
func (g *Gossiper) getRandomPeer(exclude string) *string {
	if len(g.knownPeers) <= 0 {
		return nil
	}
	peer := g.knownPeers[rand.Intn(len(g.knownPeers))]
	if peer == exclude {
		return g.getRandomPeer(exclude)
	}
	return &peer
}

type IncomingClove struct {
	predecessor string
	clove       *utils.Clove
}

/*
ClovesCollector is a LRU store for cloves with a handler and a "garbage collector"
cloves are stored in a map[sequence-number]map[predecessor]map[index] to provide a good mix
of fast lookup insertion deletion, and to be able to store multiple cloves from a predecessor
but not store many of the same exact cloves from the same exact predecessor
*/
type ClovesCollector struct {
	handler chan IncomingClove
	cloves  map[string]map[string]map[uint32]*utils.Clove
}

func NewClovesCollector(g *Gossiper) *ClovesCollector {
	cc := &ClovesCollector{handler: make(chan IncomingClove), cloves: make(map[string]map[string]map[uint32]*utils.Clove)}
	cc.manage(g)
	return cc
}

func (cc *ClovesCollector) Add(clove *utils.Clove, predecessor string) {
	var sequenceNumber string = string(clove.SequenceNumber)
	//make sure there is storage for that sequence number
	if _, ok := cc.cloves[sequenceNumber]; !ok {
		cc.cloves[sequenceNumber] = make(map[string]map[uint32]*utils.Clove)
	}
	//make sure there is storage for that predecessor
	if _, ok := cc.cloves[predecessor]; !ok {
		cc.cloves[sequenceNumber][predecessor] = make(map[uint32]*utils.Clove)
	}
	//store the clove; make sure to deep copy clove data
	idx := clove.Index
	if _, ok := cc.cloves[sequenceNumber][predecessor][idx]; !ok {
		cc.cloves[sequenceNumber][predecessor][idx] = &utils.Clove{
			Index:          clove.Index,
			SequenceNumber: clove.SequenceNumber,
			Threshold:      clove.Threshold,
			Data:           make([]byte, len(clove.Data)),
		}
		copy(cc.cloves[sequenceNumber][predecessor][idx].Data, clove.Data)
	}
	//check if the threshold is met for that sequence numnber

}

/*
MeetsThreshold checks if there are k cloves matching the given sequence-number in the collector
		Basically we have to check if there are k ways to choose cloves with both distinct
		predecessors and indices. Generally, this is NP-complete
*/
func (cc *ClovesCollector) MeetsThreshold(sn string, k uint32) (bool, []*utils.Clove) {
	if seq, ok := cc.cloves[sn]; ok {
		if uint32(len(seq)) >= k {
			ids, paths, cover := pathsCovered(seq, k)
			return getKIndependentCloves(k, seq, paths, ids, cover, make([]*utils.Clove, 0))
		}
		// there are less paths than
		return false, []*utils.Clove{}
	} else {
		utils.LogObj.Fatal("sequence number ", sn, " not found")
		return false, []*utils.Clove{}
	}

}

func pathsCovered(seq map[string]map[uint32]*utils.Clove, k uint32) ([]uint32, map[string]bool, map[uint32][]string) {
	invertedSeq := make(map[uint32][]string) //map[index][]predecessor
	availablePaths := make(map[string]bool)
	ids := make([]uint32, 0)
	for predecessor, indices := range seq {
		availablePaths[predecessor] = true
		for index := range indices {
			ids = append(ids, index)
			if _, ok := invertedSeq[index]; !ok {
				invertedSeq[index] = make([]string, 0)
			}
			invertedSeq[index] = append(invertedSeq[index], predecessor)
		}
	}
	return ids, availablePaths, invertedSeq
}

/*
removeAtI removes an element in place
Adapted from https://yourbasic.org/golang/delete-element-slice/
*/
func removeAtI(i int, a []uint32) []uint32 {
	// Remove the element at index i from a.
	tmp := a[i]
	a[i] = a[len(a)-1] // Copy last element to index i.
	a[len(a)-1] = tmp
	//a[len(a)-1] = uint32(0)   // Erase last element (write zero value).
	a = a[:len(a)-1] // Truncate slice.
	return a
}

/*
getKIndependentCloves
-
*/
func getKIndependentCloves(k uint32, seq map[string]map[uint32]*utils.Clove, pathIsAvailable map[string]bool, indices []uint32, inv map[uint32][]string, res []*utils.Clove) (bool, []*utils.Clove) {
	if k == 0 {
		return true, res
	}
	fmt.Println(res, k, indices)
	for i, index := range indices {
		for _, predecessor := range inv[index] {
			if pathIsAvailable[predecessor] {
				pathIsAvailable[predecessor] = false
				if ok, cloves := getKIndependentCloves(k-1, seq, pathIsAvailable, removeAtI(i, indices), inv, append(res, seq[predecessor][index])); ok {
					return true, cloves
				}
			}
		}
	}
	return false, []*utils.Clove{}
}

func (cc *ClovesCollector) cloveHandler(g *Gossiper, clove *utils.Clove, predecessor string) {
	rec := utils.LogObj.Named("rec")
	var sequenceNumber string = string(clove.SequenceNumber)
	logger := utils.LogObj.Named(fmt.Sprintf("clove_handler@%s", g.Name))
	//store clove by sequence number

	full := false
	if _, ok := cc.cloves[sequenceNumber][predecessor]; !ok {
		if len(cc.cloves[sequenceNumber]) >= int(clove.Threshold) {
			rec.Debug("recovered cloves", cc.cloves[sequenceNumber])
			//flip a coin?
			//clove can be reconstituted, call recover and handle type
			/*full = true
			//reconstitute chain of bytes
			cloves := make([]*utils.Clove, 0)
			paths := [2]string{"", ""}
			i := 0
			for path, collected := range cc.cloves[sequenceNumber] {
				rec.Debug(path, ": ", string(collected.Data))
				cloves = append(cloves, collected)
				paths[i] = path
				i++
			}*/
			/*df := utils.NewDataFragment(cloves[:clove.Threshold])
			switch {
			case df.Proxy != nil:
				if df.Proxy.Forward {
					if df.Proxy.SessionKey == nil {
						cloves := utils.NewProxyAccept().Split(2, 2)
						//accept to be a proxy
						for i, path := range paths {
							g.sendToPeer(path, cloves[i].Wrap())
						}
					} else {
						// register session key
					}
				} else {
					// record proxy and send session key
					g.newProxies <- &Proxy{Paths: paths}
				}
			default:
				logger.Warn("unimplemented type of data fragment")
			}*/
		}
	} else {
		logger.Warn("received two cloves from the same predecessor. dropping!")
	}
	p := 0.8
	if !full { // !full
		if rand.Float64() < p {
			if successor := g.getRandomPeer(predecessor); successor != nil {
				utils.LogObj.Named("fwd").Debug("forwarding clove to ", *successor)
				g.sendToPeer(*successor, clove.Wrap())
			} else {
				logger.Warn("could not get a successor!")
			}

			//forward to one random neighbour
		}
	}
}
func (cc *ClovesCollector) manage(g *Gossiper) {
	if g == nil {
		return
	}
	logger := utils.LogObj.Named(fmt.Sprintf("collector@%s", g.Name))
	go func() {
		for {
			select {
			case newClove := <-cc.handler:
				fmt.Println(newClove)
				cc.cloveHandler(g, newClove.clove, newClove.predecessor)
			case <-time.After(10 * time.Second):
				logger.Debug("clearing cloves", len(cc.cloves))
				//cc.cloves = make(map[string]map[string]*utils.Clove)
				//runtime.GC()
			}
		}
	}()
}

func (g *Gossiper) peerSimpleMessageHandler(packet *utils.GossipPacket) {

	utils.LogSimpleMessage(packet.Simple)
	relayPeer := packet.Simple.RelayPeerAddr
	packet.Simple.RelayPeerAddr = g.addressPeer.String()
	g.addToKnownPeers(relayPeer)
	utils.LogPeers(g.knownPeers)
	g.sendToKnownPeers(relayPeer, packet)
}

func (g *Gossiper) peerRumorStatusHandler(packet *utils.GossipPacket, address string) {
	//fmt.Fprintln(os.Stderr, "Rumor or Status received")
	if worker, ok := g.lookupWorkers(address); ok {
		worker.Buffer <- *utils.CopyGossipPacket(packet)
	} else {
		new := *utils.CopyGossipPacket(packet)
		g.createAndRunWorker(address, false, nil, &new)
	}
}

func (g *Gossiper) peerPrivateMessageHandler(packet *utils.GossipPacket) {
	fmt.Fprintln(os.Stderr, "PrivateMessage received")
	pm := packet.Private
	if pm.Destination == g.Name {
		utils.LogPrivate(pm)
		return
	}
	go g.sendPointToPoint(packet, pm.Destination)

}

func (g *Gossiper) peerDataRequestHandler(packet *utils.GossipPacket) {
	fmt.Fprintln(os.Stderr, "DataRequest received")
	request := packet.DataRequest
	if request.Destination != g.Name {
		g.sendPointToPoint(packet, request.Destination)
		return
	}
	go g.replyData(packet.DataRequest)
}

func (g *Gossiper) peerDataReplyHandler(packet *utils.GossipPacket) {
	fmt.Fprintln(os.Stderr, "DataReply received")
	reply := packet.DataReply
	if reply.Destination != g.Name {
		g.sendPointToPoint(packet, reply.Destination)
		return
	}
	dd := g.lookupDownloader(reply.HashValue)
	if dd == nil {
		fmt.Fprintln(os.Stderr, "Didn't found the corresponding downloader")
		return
	}
	dd.replies <- reply
}

func (g *Gossiper) peerSearchRequestHandler(packet *utils.GossipPacket) {
	fmt.Fprintln(os.Stderr, "SearchRequest received")
	request := packet.SearchRequest
	if !g.lookupSearchRequest(request.Origin, request.Keywords) {
		go g.NewSearchReplier(request)
		return
	}
	fmt.Fprintln(os.Stderr, "Duplicate SearchRequest received")
}

func (g *Gossiper) peerSearchReplyHandler(packet *utils.GossipPacket) {
	fmt.Fprintln(os.Stderr, "SearchReply received")
	reply := packet.SearchReply
	if reply.Destination != g.Name {
		g.sendPointToPoint(packet, reply.Destination)
		return
	}
	searcher := g.getFileSearcher()
	if searcher.running {
		searcher.replies <- reply
	} else {
		fmt.Fprintln(os.Stderr, "Search not in progress")
	}
}

func (g *Gossiper) peerTLCMessageHandler(packet *utils.GossipPacket, address string) {
	fmt.Fprintln(os.Stderr, "TLCMessage received")
	//ultimately send it to the rumormonger
	defer g.peerRumorStatusHandler(packet, address)
	msg := packet.TLCMessage
	if msg.Origin == g.Name {
		return
	}
	utils.LogTLCGossip(msg)

	if msg.Confirmed != -1 {
		g.tlcStorage.addMessage(msg)

		if g.hw3ex4 && g.consensus.running {
			g.consensus.msg <- msg
			return
		}
		publisher := g.checkPublisher(uint32(msg.Confirmed))
		if publisher != nil {
			publisher.msg <- msg
		}
		return
	}
	if g.checkBlockValidity(&msg.TxBlock) {
		g.tlcStorage.addMessage(msg)
		g.TLCAck(packet)
	} else {
		fmt.Fprintln(os.Stderr, "mapping already exists")
		//don't ack the message, for now nothing
	}
}

func (g *Gossiper) peerTLCAckHandler(packet *utils.GossipPacket) {
	fmt.Fprintln(os.Stderr, "TLCACK received")
	ack := packet.Ack
	if ack.Destination == g.Name {
		if p := g.lookupPublisher(ack.ID); p != nil {
			p.acks <- ack
		} else {
			fmt.Fprintln(os.Stderr, "Publisher not found, dropping packet")
		}
		return
	}
	go g.sendPointToPoint(packet, ack.Destination)

}

// func (g *Gossiper) peerCloveHandler(packet *utils.GossipPacket){
// 	fmt.Fprintln(os.Stderr,"Clove received")
// 	clove := packet.Clove
// 	g.addClove(clove)
// }
