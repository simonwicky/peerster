package gossiper

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"reflect"
	"runtime"
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
			} else {
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
				case packet.GCSearchRequest != nil:
					go g.peerGCSearchRequestHandler(&packet)
				case packet.GCSearchReply != nil:
					go g.peerGCSearchReplyHandler(&packet)
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
	handler      chan IncomingClove
	directs      chan *utils.Clove
	cloves       map[string]map[string]map[uint32]*utils.Clove
	routingTable map[string]string // records previous hop -> next hop
}

func NewClovesCollector(g *Gossiper) *ClovesCollector {
	cc := &ClovesCollector{
		handler:      make(chan IncomingClove),
		directs:      make(chan *utils.Clove),
		cloves:       make(map[string]map[string]map[uint32]*utils.Clove),
		routingTable: make(map[string]string),
	}
	cc.manage(g)
	return cc
}

func CloneValue(source interface{}, destin interface{}) {
	x := reflect.ValueOf(source)
	if x.Kind() == reflect.Ptr {
		starX := x.Elem()
		y := reflect.New(starX.Type())
		starY := y.Elem()
		starY.Set(starX)
		reflect.ValueOf(destin).Elem().Set(y.Elem())
	} else {
		destin = x.Interface()
	}
}

/*
Add is a thread unsafe method to add a clove to the collector(it is meant to be used in a isolated context)
*/
func (cc *ClovesCollector) Add(clove *utils.Clove, predecessor string) bool {
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
			Threshold:      clove.Threshold,
			Data:           make([]byte, len(clove.Data)),
			SequenceNumber: make([]byte, len(clove.SequenceNumber)),
		}
		copy(cc.cloves[sequenceNumber][predecessor][idx].SequenceNumber[:], clove.SequenceNumber[:])
		copy(cc.cloves[sequenceNumber][predecessor][idx].Data[:], clove.Data[:])
		return true
	}
	//check if the threshold is met for that sequence numnber
	return false
}

/*
MeetsThreshold checks if there are k cloves matching the given sequence-number in the collector
		Basically we have to check if there are k ways to choose cloves with both distinct
		predecessors and indices. Generally, this is NP-complete
*/
func (cc *ClovesCollector) MeetsThreshold(sn string, k uint32) (bool, []*utils.Clove, []string) {
	if seq, ok := cc.cloves[sn]; ok {
		if uint32(len(seq)) >= k {
			ids, paths, cover := pathsCovered(seq, k)
			return getKIndependentCloves(k, seq, paths, ids, cover, make([]*utils.Clove, 0), []string{})
		}
		// there are less paths than k
		return false, []*utils.Clove{}, []string{}
	}
	utils.LogObj.Fatal("sequence number ", sn, " not found")
	return false, []*utils.Clove{}, []string{}
}

/*
pathsCovered given the cloves received for a particular sequence number returns a list of all the clove indices, paths that are available (where a proxy wasn't already discovered
and the "inverted" sequence of cloves from index to a list of predecessors). Its purpose is
to help find k cloves coming from different paths and having different indices
*/
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
getKIndependentCloves returns a tuple of a boolean indicating whether k different cloves have come from
k different paths, the list of cloves if it exists and the list of paths if they exist.
The subtlety is that if many cloves came from the same path (which can happen with loops),
then we get to choose which clove of that path contribute to recovering the message.
-
*/
func getKIndependentCloves(k uint32, seq map[string]map[uint32]*utils.Clove, pathIsAvailable map[string]bool, indices []uint32, inv map[uint32][]string, resa []*utils.Clove, resb []string) (bool, []*utils.Clove, []string) {
	if k == 0 {
		return true, resa, resb
	}
	//fmt.Println(res, k, indices)
	for _, index := range indices {
		for _, predecessor := range inv[index] {
			if pathIsAvailable[predecessor] {
				pathIsAvailable[predecessor] = false
				//check if seen[string(seq[predecessor][index].Data)]
				newIndices := make([]uint32, 0)
				for _, other := range indices {
					if other != index {
						newIndices = append(newIndices, other)
					}
				}
				if ok, cloves, paths := getKIndependentCloves(k-1, seq, pathIsAvailable, newIndices, inv, append(resa, seq[predecessor][index]), append(resb, predecessor)); ok {
					return true, cloves, paths
				}
			}
		}
	}
	return false, []*utils.Clove{}, []string{}
}

func (cc *ClovesCollector) cloveHandler(g *Gossiper, clove *utils.Clove, predecessor string) {
	//rec := utils.LogObj.Named("rec")
	var sequenceNumber string = string(clove.SequenceNumber)
	logger := utils.LogObj.Named("rec")

	//store clove by sequence number
	cc.Add(clove, predecessor)
	forwarding := false
	p := 0.8
	if met, cloves, paths := cc.MeetsThreshold(sequenceNumber, clove.Threshold); met {
		//logger.Debug("recovered clove from", paths)
		df, err := utils.NewDataFragment(cloves)
		if err == nil {
			logger.Debug(g.Name, "recovered clove")
			switch {
			case df.Proxy != nil:
				if df.Proxy.Forward {
					if df.Proxy.SessionKey == nil {
						output, err := utils.NewProxyAccept(g.directProxyPort).Split(2, 2)
						if err == nil {
							//accept to be a proxy
							for i, path := range paths {
								//logger.Debug("sent accept clove to ", path)
								logger.Debug("sending ACCEPT to ", path)
								g.sendToPeer(path, output[i].Wrap())
							}
						}
					} else {
						// register session key and id
						logger.Debug("registering session key")
					}
				} else {
					var fixPaths [2]string
					copy(fixPaths[:], paths[:2])
					// record proxy and send session key
					if df.Proxy.IP != nil && df.Proxy.SessionKey != nil {
						g.newProxies <- &Proxy{Paths: fixPaths, IP: *df.Proxy.IP, SessionKey: *df.Proxy.SessionKey}
					}
				}
			case df.Delivery != nil: // this is read by a provider proxy
				//directly connect by TCP to proxy provided
				atTcp, err := net.ResolveTCPAddr("tcp", df.Delivery.IP)
				if err != nil {
					utils.LogObj.Fatal(err.Error(), " dropping cloves")
					return
				}
				connect, err := net.DialTCP("tcp", nil, atTcp)
				if err != nil {
					utils.LogObj.Fatal(err.Error(), " dropping cloves")
					return
				}
				for _, dataClove := range df.Delivery.Cloves {
					_, err := connect.Write(dataClove)
					if err != nil {
						utils.LogObj.Fatal(err.Error())
					}
				}
			case df.Content != nil:
				//index file
			case df.Query != nil :
				searcher := g.getGCFileSearcher()
			go searcher.startSearch(df.Query.Keywords, &df.Query.SessionKey)
			case df.FileInfo != nil:
				g.FilesRouting.addOwnerPath(*df.FileInfo,paths)
			}
			
			
		} else {
			data := []string{}
			for _, clove := range cloves {
				data = append(data, fmt.Sprintf("%d::%s ", clove.Index, string(clove.Data)))
			}
			logger.Fatal(err.Error(), data, cc.cloves[sequenceNumber])
			forwarding = true
		}
	} else {
		forwarding = true
	}
	if forwarding { // !full
		if successor, ok := cc.routingTable[predecessor]; ok && successor != g.addressPeer.String() {
			g.sendToPeer(successor, clove.Wrap())
		} else {
			//forward to one random neighbour
			if rand.Float64() < p {
				if successor := g.getRandomPeer(predecessor); successor != nil {
					utils.LogObj.Named("fwd").Debug("forwarding clove to ", *successor)
					cc.routingTable[predecessor] = *successor
					cc.routingTable[*successor] = predecessor
					g.sendToPeer(*successor, clove.Wrap())
				} else {
					logger.Warn("could not get no successor!")
				}
			}
		}

	}
}

/*
manage is a forwarder/handler of cloves coupled with a state resetter that will delete all the cloves
every x seconds
*/
func (cc *ClovesCollector) manage(g *Gossiper) {
	if g == nil {
		return
	}
	logger := utils.LogObj.Named("man")
	cleaningTime := time.NewTicker(15 * time.Second)
	go func() {
		for {
			select {
			//case clove := <-cc.directs:
			// look up "cloves routing table" and forward
			case newClove := <-cc.handler:
				cc.cloveHandler(g, newClove.clove, newClove.predecessor)
			case <-cleaningTime.C:
				logger.Debug("clearing cloves", len(cc.cloves))
				cc.cloves = make(map[string]map[string]map[uint32]*utils.Clove)
				runtime.GC()
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

func (g *Gossiper) peerGCSearchRequestHandler(packet *utils.GossipPacket){
	logger := utils.LogObj.Named("GCSearch")
	request := packet.GCSearchRequest
	searcher := g.getGCFileSearcher()
	searcher.repliesMux.Lock()
	_, alreadyReceived := searcher.repliesDispatcher[request.ID]; 
	//Send failure because we already received the request
	if alreadyReceived {
		fmt.Printf("Already received GCSearchRequest with ID %d from relay %s\n", packet.GCSearchRequest.ID, packet.GCSearchRequest.Origin)
		reply := &utils.GCSearchReply{
			ID: request.ID,
			Origin: g.Name, 
			Failure:true,
			HopLimit:g.GCSearchHopLimit,
		}
		g.sendPointToPoint(&utils.GossipPacket{GCSearchReply:reply}, packet.GCSearchRequest.Origin)

	}
	searcher.repliesMux.Unlock()
	
	if !alreadyReceived{

		var foundFiles []*FileData
		keywords := packet.GCSearchRequest.Keywords
		for _, kw := range keywords {
			//Assuming there is consensus over the file names 
			if ips :=packet.GCSearchRequest.ProxiesIP; ips != nil && len(keywords) == 1 {
				foundFiles = append(foundFiles, g.fileStorage.lookupFile(kw)...)

				if  foundFiles[0].name == keywords[0]{

					fmt.Println("Deliver file ", kw)
					g.deliver(kw, *ips)
				}else {
					g.FilesRouting.Lock()
					if len(g.FilesRouting.filesRoutes[keywords[0]].ProxyOwnerPaths) == 2{
						data := &utils.DataFragment{
							GCSearchRequest: packet.GCSearchRequest,
						}

						cloves, err := data.Split(2,2)
						if err != nil {
							fmt.Println("Error forwarding GCSearchRequest to content holder")
						}else {
							paths :=  g.FilesRouting.filesRoutes[keywords[0]].ProxyOwnerPaths
							for i, path := range paths{
								g.sendToPeer(path, cloves[i].Wrap())
							}
							logger.Debug("Forwarding search request to content holder via anonymous paths", paths)

						}
					}
					g.FilesRouting.Unlock()
				}

			}



		}
		if len(foundFiles) < int(searcher.matchThreshold) {
			searcher.manageRequest(packet.GCSearchRequest)
		}
		routingResults := g.FilesRouting.asSearchResults()
		accessibleFiles := append(routingResults, g.fileStorage.asSearchResults()... )
		reply := &utils.GCSearchReply{
			ID:packet.GCSearchRequest.ID,
			Origin: g.Name,
			AccessibleFiles:accessibleFiles,
			Failure: false,
			HopLimit:g.GCSearchHopLimit,
		}
		utils.LogGCSearchReply(reply)
		g.sendPointToPoint(&utils.GossipPacket{GCSearchReply:reply}, packet.GCSearchRequest.Origin)
	}
}

func (g *Gossiper) peerGCSearchReplyHandler(packet *utils.GossipPacket){
	g.getGCFileSearcher().receiveReply(packet.GCSearchReply)
}


// func (g *Gossiper) peerCloveHandler(packet *utils.GossipPacket){
// 	fmt.Fprintln(os.Stderr,"Clove received")
// 	clove := packet.Clove
// 	g.addClove(clove)
// }
