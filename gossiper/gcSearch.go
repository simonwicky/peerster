/*
Author: Boubacar Camara

Contains various functions related to the Garlic cast search 
*/
package gossiper
import (
	"fmt"
	"github.com/simonwicky/Peerster/utils"
	"os"
	"strings"
	"math/rand"
)
//Packets handlers
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
			if ips :=packet.GCSearchRequest.ProxiesIP; ips != nil /*&& len(keywords) == 1*/ {
				foundFiles = append(foundFiles, g.fileStorage.lookupFile(kw)...)

				if  len(foundFiles) > 0 && foundFiles[0].name == kw{
					fmt.Println(1)

					fmt.Println("Deliver file ", kw)
					g.deliver(kw, *ips)
				}else {
					g.FilesRouting.Lock()
					if len(g.FilesRouting.filesRoutes[kw].ProxyOwnerPaths) == 2{
						data := &utils.DataFragment{
							GCSearchRequest: packet.GCSearchRequest,
						}
						fmt.Println(2)

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

			fmt.Println(3)


		}
		fmt.Println(4)
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
		//utils.LogGCSearchReply(reply)
		g.sendPointToPoint(&utils.GossipPacket{GCSearchReply:reply}, packet.GCSearchRequest.Origin)
	}
}

func (g *Gossiper) peerGCSearchReplyHandler(packet *utils.GossipPacket){
	g.getGCFileSearcher().receiveReply(packet.GCSearchReply)
}

//Client handler


func(g *Gossiper) clientGCFileSearchHandler(message *utils.Message) {
	searcher := g.getGCFileSearcher()
	d := uint(len(g.proxyPool.proxies))

	providerProxies, err := g.proxyPool.GetD(d)

	if message.UseProxy != nil && *message.UseProxy{
		data := utils.DataFragment{
			Query: &utils.Query{Keywords: strings.Split(*message.Keywords,",")},
		}
		logger := utils.LogObj.Named("postFile")

		//get file from filestorage
		//providerProxies, err := g.proxyPool.GetD(d)
		if err != nil {
			logger.Fatal(err.Error())
			return
		}
		for _, proxy := range providerProxies {
			sessionKey := make([]byte, g.settings.SessionKeySize)
			rand.Read(sessionKey)
			data.Query.SessionKey = sessionKey
			cloves, err := data.Split(2, 2) // normally we should collect all cloves to make sure that no error occurs
			if err != nil {
				logger.Fatal(err.Error())
				return
			}
			for j, clove := range cloves {
				g.sendToPeer(proxy.Paths[j], clove.Wrap())
			}
		}
		
	}else {
		if !searcher.running {
			keywords := strings.Split(*message.Keywords,",")
			ips := make([]string, len(providerProxies))
			for _, p := range providerProxies{
				ips =append(ips, p.IP)

			}
			searcher.proxiesMux.Lock()
			searcher.proxiesIP = &ips
			searcher.proxiesMux.Unlock()
			go searcher.startSearch(keywords, nil)
		} else {
			fmt.Fprintln(os.Stderr,"File search already running")
		}
	}
	



}