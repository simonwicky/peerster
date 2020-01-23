
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


)
//Worker managing the file search in the Garlic Cast extension
type GCFileSearcher struct{
	running bool
	replies chan *utils.GCSearchReply
	keywords []string
	matchThreshold uint32
	g *Gossiper
	matches []*Match
}

func NewGCFileSearcher(g *Gossiper) *GCFileSearcher {
	return &GCFileSearcher{
		g : g,
		replies : make(chan *utils.GCSearchReply),
		running : false,
		matchThreshold: 2,
	}
}

//Start the file search given the specified keywords
func (s *GCFileSearcher) Start(keywords []string){
	s.running = true
	s.keywords = keywords
	s.matches = make([]*Match,0)
	go s.search(keywords)
	//s.handleReply()
	s.running = false
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
	}
	if !s.running {
		return
	}
	fmt.Fprintf(os.Stderr,"Sending Garlic Cast search")
	s.manageRequest(searchRequest)
}

func contains(haystack []*utils.SearchResult, needle *utils.SearchResult) bool{
	for _, result := range haystack {
		return  result.FileName == needle.FileName && hex.EncodeToString(result.MetafileHash) == hex.EncodeToString(needle.MetafileHash)
	}
	return false
}

/*
	Manages the file search. Sends search requests in the appropriate order
	depending on the gossiper files routing table. 
*/ 
func (s *GCFileSearcher) manageRequest(searchRequest *utils.GCSearchRequest){
	ticker := time.NewTicker(time.Second * time.Duration(5))
	var receivedResults []*utils.SearchResult
	peersOrdering := s.g.FilesRouting.RoutesSorted(searchRequest.Keywords)

	for _, peer := range peersOrdering {
		if len(receivedResults) < int(s.matchThreshold){
			s.SendRequest(searchRequest, peer)
			select{
				case reply := <- s.replies:
					s.g.FilesRouting.UpdateRouting(reply)
					for _, newResult := range reply.Results {
						if !contains(receivedResults, newResult){
							receivedResults = append(receivedResults, newResult)
						}
					}
				case <- ticker.C:
			}
		}else{
			ticker.Stop()
		}
	}
}



func (s *GCFileSearcher) SendRequest(searchRequest *utils.GCSearchRequest, peer string){
	pkt := &utils.GossipPacket {
		GCSearchRequest: searchRequest,
	}
	s.g.sendToPeer(peer, pkt)
}