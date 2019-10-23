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
	g.fileStorage.addFromSystem(fileName)
}

func (g *Gossiper) uiFileDownloadHandler(request *utils.DataRequest,fileName string){
	g.NewDatadownloader(request, fileName)
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


func (g *Gossiper) getLatestRumors() utils.RumorMessages{
	list := []utils.RumorMessage{}
	for _, key := range g.latestRumors.Container{
		list = append(list, g.messages[key])
	}
	return list
}



