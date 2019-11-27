package gossiper

import ("github.com/simonwicky/Peerster/utils"
		"strings"
)

//================================================================
//UI SIDE
//================================================================

func (g *Gossiper) uiRumorHandler(packet *utils.GossipPacket) {
	utils.LogClient(packet.Rumor.Text)
	rumor := g.generateRumor(packet.Rumor.Text)
	packet.Rumor = &rumor
	g.sendToRandomPeer(packet)
}

func (g *Gossiper) uiPrivateMessageHandler(packet *utils.GossipPacket) {
	packet.Private.Origin = g.Name
	packet.Private.HopLimit = 10

	g.sendPointToPoint(packet, packet.Private.Destination)
}

func (g *Gossiper) uiFileIndexHandler(fileName string){
	g.fileStorage.addFromSystem(g,fileName)
}

func (g *Gossiper) uiFileDownloadHandler(hash []byte, destination ,fileName string){
	dr := utils.DataRequest{
		Origin: g.Name,
		Destination: destination,
		HopLimit:10,
		HashValue: make([]byte,len(hash)),
	}
	copy(dr.HashValue,hash)
	g.NewDatadownloader(&dr, fileName)
}

func (g *Gossiper) uiAddPeer(peer string) {
	g.addToKnownPeers(peer)
}

func (g *Gossiper) getName() string {
	return g.Name
}


func (g *Gossiper) getKnownPeers() string {
	return strings.Join(g.knownPeers, ",")
}

func(g *Gossiper) getIdentifiers() string {
	return g.dumpDSDV()
}

func (g *Gossiper) uiFileSearchHandler(keywords string) string{
	searcher := g.getFileSearcher()
	if !searcher.running {
		keywords := strings.Split(keywords,",")
		searcher.Start(2,keywords)
	}
	matches := searcher.getMatches()
	if len(matches) == 0 {
		return ""
	}
	var names []string
	for _,m := range matches {
		names = append(names,m.name)
	}
	return strings.Join(names,",")
}

func (g *Gossiper) uiGetTLCMessages() string {
	msgs := g.tlcStorage.getConfirmedMessages()
	var names []string
	for _,m := range msgs {
		names = append(names,m.TxBlock.Transaction.Name)
	}
	return strings.Join(names, ",")
}



func (g *Gossiper) getLatestRumors() utils.RumorMessages{
	list := []utils.RumorMessage{}
	for _, key := range g.latestRumors.Container{
		list = append(list, g.messages[key])
	}
	return list
}



