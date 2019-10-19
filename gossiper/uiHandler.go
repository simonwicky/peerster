package gossiper

import ("github.com/simonwicky/Peerster/utils"
		"strings"
		"fmt"
		"os"
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
	address := g.lookupDSDV(packet.Private.Destination)
	if address == "" {
		fmt.Fprintln(os.Stderr,"Next hop not found, aborting")
		return
	}

	g.sendToPeer(address, packet)
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



