package gossiper

import ("github.com/simonwicky/Peerster/utils"
		_"encoding/hex"
		"fmt"
		"os"
		"time"
)


type SearchReplier struct {
	origin string
	keywords []string
	g *Gossiper
}

func (g *Gossiper) NewSearchReplier(request *utils.SearchRequest){
	sr := &SearchReplier{
		origin : request.Origin,
		keywords : request.Keywords,
		g : g,
	}
	g.addSearchReplier(sr)
	sr.replyRequest()
	g.relayRequest(request)
	time.Sleep(500 * time.Millisecond)
	fmt.Fprintln(os.Stderr,"Removing replier")
	g.removeSearchReplier(sr)
}

func (sr *SearchReplier) replyRequest(){
	storage := sr.g.fileStorage
	results := make([]*utils.SearchResult,0)
	for _, keyword := range sr.keywords {
		matchingFile := storage.lookupFile(keyword)
		for _, file := range matchingFile {
			if file.local {
				result := &utils.SearchResult{
					FileName : file.name,
					MetafileHash : file.metafileHash,
					ChunkMap : file.chunkmap,
					ChunkCount : uint64(len(file.metafile)),
				}
				results = append(results,result)
			}
		}
	}
	if len(results) == 0 || sr.origin == sr.g.Name {
		fmt.Fprintln(os.Stderr,"No results, or own request")
		return
	}
	reply := &utils.SearchReply{
		Origin: sr.g.Name,
		Destination: sr.origin,
		HopLimit : sr.g.hopLimit,
		Results : results,
	}
	sr.g.sendPointToPoint(&utils.GossipPacket{SearchReply: reply}, reply.Destination)
}

func (g *Gossiper) relayRequest(request *utils.SearchRequest){
	request.Budget -= 1
	if request.Budget <= 0 {
		return
	}
	budget := int(request.Budget)
	nbPeers := len(g.knownPeers)
	//if less budget than peers, send to val(budget) random peer
	if budget < nbPeers {
		for i := 0 ; i < budget; i++ {
			request.Budget = 1
			g.sendToRandomPeer(&utils.GossipPacket{SearchRequest: request})
		}
		return
	}
	//if budget is multiple of peers number, send to everyone, budget is budget / peers number
	if budget % nbPeers == 0 {
		request.Budget = uint64(budget / nbPeers)
		g.sendToKnownPeers("",&utils.GossipPacket{SearchRequest : request})
	}

	//if budget is not multiple of peers number, send to everyone but one, budget is budget / peers number or budget % peers number for the last one
	if budget % nbPeers != 0 {
		request.Budget = uint64(budget / nbPeers)
		g.sendToKnownPeers(g.knownPeers[0],&utils.GossipPacket{SearchRequest : request})
		request.Budget = uint64(budget % nbPeers)
		g.sendToPeer(g.knownPeers[0],&utils.GossipPacket{SearchRequest : request})
	}

}