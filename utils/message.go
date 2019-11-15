package utils

type Message struct {
	Text string
	Destination *string
	File *string
	Request *[]byte
	Budget *uint64
	Keywords *string
}

type SimpleMessage struct {
	OriginalName string
	RelayPeerAddr string
	Contents string
}

type RumorMessage struct {
	Origin string
	ID	uint32
	Text string
}

type PrivateMessage struct {
	Origin string
	ID	uint32
	Text string
	Destination string
	HopLimit uint32
}
type RumorMessageKey struct {
	Origin string
	ID uint32
}

type RumorMessages []RumorMessage

type PeerStatus struct {
	Identifer string
	NextID uint32
}

type StatusPacket struct {
	Want []PeerStatus
}

type DataRequest struct {
	Origin string
	Destination string
	HopLimit uint32
	HashValue []byte
}

type DataReply struct {
	Origin string
	Destination string
	HopLimit uint32
	HashValue []byte
	Data []byte
}

type SearchRequest struct {
	Origin string
	Budget uint64
	Keywords []string
}

type SearchReply struct {
	Origin string
	Destination string
	HopLimit uint32
	Results []*SearchResult
}

type SearchResult struct {
	FileName string
	MetafileHash []byte
	ChunkMap []uint64
	ChunkCount uint64
}

type GossipPacket struct {
	Simple *SimpleMessage
	Rumor *RumorMessage
	Status *StatusPacket
	Private *PrivateMessage
	DataRequest *DataRequest
	DataReply *DataReply
	SearchRequest *SearchRequest
	SearchReply *SearchReply
}