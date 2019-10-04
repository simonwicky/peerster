package gossiper

import ("fmt"
		"github.com/simonwicky/Peerster/utils"
		"strings"
)

//================================================================
//UI SIDE
//================================================================

func (g *Gossiper) uiRumorHandler(packet *utils.GossipPacket) {
	fmt.Println("CLIENT MESSAGE " + packet.Rumor.Text)
	packet.Rumor.Origin = g.Name
	packet.Rumor.ID = g.counter
	statusIndex := -1
	for index,status := range g.currentStatus.Want {
		if status.Identifer == packet.Rumor.Origin{
			statusIndex = index
		}
	}
	g.updateStatus(utils.PeerStatus{Identifer : packet.Rumor.Origin, NextID : packet.Rumor.ID + 1}, statusIndex)
	g.counter_lock.Lock()
	g.counter += 1
	g.counter_lock.Unlock()
	g.sendToRandomPeer(packet)
	//add the message to storage
	g.addMessage(packet.Rumor)
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



func (g *Gossiper) getLatestRumors() utils.RumorMessages{
	list := []utils.RumorMessage{}
	for _, key := range g.latestRumors.Container{
		list = append(list, g.messages[key])
	}
	return list
}



