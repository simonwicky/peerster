
package gossiper

import (
	//"bytes"
	crand"crypto/rand"
	"encoding/hex"
	"encoding/binary"
	"github.com/simonwicky/Peerster/utils"
	"fmt"
	"os"
	"time"
	"sync"

)

//Worker managing the file search in the Garlic Cast extension
type GCFileSearcher struct{
	running bool
	replies chan *utils.GCSearchReply
	keywords []string
	matchThreshold uint32
	g *Gossiper
	matches []*Match
	repliesDispatcher map[uint32]chan*utils.GCSearchReply
	repliesMux sync.Mutex
}

func NewGCFileSearcher(g *Gossiper) *GCFileSearcher {
	return &GCFileSearcher{
		g : g,
		replies : make(chan *utils.GCSearchReply),
		running : false,
		matchThreshold: 2,
		repliesDispatcher:make(map[uint32]chan*utils.GCSearchReply),
	}
}

//Start the file search given the specified keywords
func (s *GCFileSearcher) startSearch(keywords []string){
	s.running = true
	s.keywords = keywords
	s.matches = make([]*Match,0)
	s.search(keywords)
	s.running = false

	
	//s.handleReply()
}

func (s *GCFileSearcher) search(keyword []string){
	randBytes := make([]byte, 4)
	_, err := crand.Read(randBytes)
	if err != nil {
		fmt.Println("Error generating random sequence number for the search ID.")
		return 
	}
	
	searchRequest := &utils.GCSearchRequest{
			ID: binary.LittleEndian.Uint32(randBytes),
			Origin : s.g.Name,
			Keywords : s.keywords,
			HopLimit:s.g.GCSearchHopLimit,
	}
	fmt.Println("New GC search ID ", searchRequest.ID)
	if !s.running {
		return
	}
	s.manageRequest(searchRequest)
}

func contains(haystack []*utils.SearchResult, needle *utils.SearchResult) bool{
	for _, result := range haystack {
		//check chunkmap
		return  result.FileName == needle.FileName && hex.EncodeToString(result.MetafileHash) == hex.EncodeToString(needle.MetafileHash)
	}
	return false
}

func (s *GCFileSearcher) receiveReply(reply *utils.GCSearchReply){

	s.repliesMux.Lock()
	if channel, ok := s.repliesDispatcher[reply.ID]; ok{
		channel<-reply
	}
	s.repliesMux.Unlock()
}

func (s *GCFileSearcher) processReply(channel chan *utils.GCSearchReply, ticker *time.Ticker ){
	//TODO: modularize select in managerequest here
}

/*
	Manages the file search. Sends search requests in the appropriate order
	depending on the gossiper files routing table. 
*/ 
func (s *GCFileSearcher) manageRequest(searchRequest *utils.GCSearchRequest){
	var receivedResults []*utils.SearchResult
	fRoutesSorted := s.g.FilesRouting.RoutesSorted(searchRequest.Keywords)
	
	ticker := time.NewTicker(time.Second * time.Duration(5))
	replyChannel := make(chan *utils.GCSearchReply, 20)
	s.repliesMux.Lock()
	s.repliesDispatcher[searchRequest.ID] = replyChannel
	s.repliesMux.Unlock()

	
	for _, fRoute := range fRoutesSorted {
		if len(receivedResults) < int(s.matchThreshold){
			//should we send the request to other peers if the first does not respond
			if s.SendRequest(*searchRequest, s.g.lookupDSDV(fRoute.Routes[0])){

				select{
					case reply := <- replyChannel:
							
						if !reply.Failure { 
							utils.LogGCSearchReply(reply)

							s.g.FilesRouting.UpdateRouting(*reply)
							for _, newResult := range reply.Results {
								if !contains(receivedResults, newResult){
									receivedResults = append(receivedResults, newResult)
								}
							}
						}
					case <- ticker.C:
				}
			}
			
		}else{
			ticker.Stop()
		}
	} 
	if len(receivedResults) < int(s.matchThreshold){
		var restingPeers []string
		for _, peer := range s.g.knownPeers{
			peerFound := false
			if peer != s.g.lookupDSDV(searchRequest.Origin){
				for _, fRoutes := range fRoutesSorted {
					for _, routePeer := range fRoutes.Routes{
						if s.g.lookupDSDV(routePeer) == peer {
							peerFound = true
						}
					}
				}
				if !peerFound {
					restingPeers = append(restingPeers, peer)
				}
			}
			
			
		}
		fmt.Println("resting peers", restingPeers)
		for _, peer := range restingPeers {
			
			if len(receivedResults) < int(s.matchThreshold){
				if s.SendRequest(*searchRequest, peer){
					select{
						case reply := <- replyChannel:
							
							if ! reply.Failure{
								utils.LogGCSearchReply(reply)
								
								s.g.FilesRouting.UpdateRouting(*reply)
								for _, newResult := range reply.Results {
									if !contains(receivedResults, newResult){
										receivedResults = append(receivedResults, newResult)
									}
								}
							}
						case <- ticker.C:
					}
				}
			}else{
				ticker.Stop()
			}
		}
	}
	s.g.FilesRouting.dump()
	return 
}	

func (s *GCFileSearcher) SendRequest(searchRequest utils.GCSearchRequest, peer string) bool{
	fmt.Println(peer, searchRequest.Origin)

	if peer == searchRequest.Origin{
		return false
	}
	searchRequest.Origin = s.g.Name
	//Update the origin so peers only know the direct upstream and downstream nodes in the chain
	pkt := &utils.GossipPacket {
		GCSearchRequest: &searchRequest,
	}

	fmt.Fprintf(os.Stderr,"Sending Garlic Cast search to %s\n", peer)
	s.g.sendToPeer(peer, pkt)
	return true
}