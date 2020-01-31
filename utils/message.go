package utils

// import ("math/big")

type Message struct {
	Text        string
	Destination *string
	File *string
	Request *[]byte
	Budget *uint64
	Keywords *string
	GC *bool
	UseProxy *bool
}

type SimpleMessage struct {
	OriginalName  string
	RelayPeerAddr string
	Contents      string
}

type RumorMessage struct {
	Origin string
	ID     uint32
	Text   string
}

type PrivateMessage struct {
	Origin      string
	ID          uint32
	Text        string
	Destination string
	HopLimit    uint32
}

type PeerStatus struct {
	Identifer string
	NextID    uint32
}

type StatusPacket struct {
	Want []PeerStatus
}

type DataRequest struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte
}

type DataReply struct {
	Origin      string
	Destination string
	HopLimit    uint32
	HashValue   []byte
	Data        []byte
}

type SearchRequest struct {
	Origin   string
	Budget   uint64
	Keywords []string
}

type SearchReply struct {
	Origin      string
	Destination string
	HopLimit    uint32
	Results     []*SearchResult
}
type GCSearchReply struct {
	ID uint32
	Origin string
	Results []*SearchResult
	AccessibleFiles []*SearchResult
	Failure bool
	HopLimit uint32
}
type GCSearchRequest struct {
	ID uint32
	Origin string
	Keywords []string
	ProxiesIP *[]string
	HopLimit uint32
	SessionKey []byte
}



type GCDownloadRequest struct {
	ID uint32
	ProxiesIP []string
	FileMetaHash []byte
}

type SearchResult struct {
	FileName     string
	MetafileHash []byte
	ChunkMap     []uint64
	ChunkCount   uint64
}

type TxPublish struct {
	Name         string
	Size         int64
	MetafileHash []byte
}

type BlockPublish struct {
	PrevHash    [32]byte
	Transaction TxPublish
}

type TLCMessage struct {
	Origin      string
	ID          uint32
	Confirmed   int
	TxBlock     BlockPublish
	VectorClock *StatusPacket
	Fitness     float32
}

type TLCAck PrivateMessage
type GossipPacket struct {
	Simple        *SimpleMessage
	Rumor         *RumorMessage
	Status        *StatusPacket
	Private       *PrivateMessage
	DataRequest   *DataRequest
	DataReply     *DataReply
	SearchRequest *SearchRequest
	SearchReply 	*SearchReply
	GCSearchRequest *GCSearchRequest
	GCSearchReply 	*GCSearchReply
	TLCMessage 		*TLCMessage
	Ack *TLCAck
	Clove         *Clove

}
