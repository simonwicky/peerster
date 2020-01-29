package utils

import (
	"crypto/rand"

	"go.dedis.ch/protobuf"
)

// import ("math/big")

type Message struct {
	Text        string
	Destination *string
	File *string
	Request *[]byte
	Budget *uint64
	Keywords *string
	GC *bool
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

/*
Clove is the backbone of secret sharing
*/
type Clove struct {
	Index          uint32
	Threshold      uint32
	SequenceNumber []byte
	Data           []byte
	AssociatedData []byte // for 2-threshold data only; threshold is assumed to be 2 and index is % 2 + 1
}

/*
Wrap wraps a clove into a gossipPacket
*/
func (clove *Clove) Wrap() *GossipPacket {
	return &GossipPacket{Clove: clove}
}

/*
DataFragment is a generic type to hold data that can be split to cloves
	Different types can be obtained by calling the type method
	@Fordward - non-nil indicates that this is a proxyrequest. `true` means that the request is from initiator to proxy.
	@SessionKey - reserved to send the session key at the end of the
*/
type DataFragment struct {
	Proxy    *ProxyRequest
	Query    *Query
	Content  *Content
	Delivery *Delivery
}

/*
NewDataFragment returns a DataFragment recovered from k cloves
*/
func NewDataFragment(cloves []*Clove) (*DataFragment, error) {
	threshold := len(cloves) //cloves[0].Threshold
	xs := make([]int, threshold)
	data := make([][]byte, threshold)
	for i, clove := range cloves {
		xs[i] = int(clove.Index)
		data[i] = clove.Data
	}
	marshalled, err := recoverSecret(data, xs)
	if err != nil {
		return nil, err
	}
	var df DataFragment
	err = protobuf.Decode(marshalled, &df)
	if err != nil {
		return nil, err
	}
	return &df, nil
}

func NewProxyInit() *DataFragment {
	return &DataFragment{Proxy: &ProxyRequest{Forward: true}}
}

func NewProxyAccept() *DataFragment {
	return &DataFragment{Proxy: &ProxyRequest{Forward: false}}
}

/*
	NewProxyAck creates a DataFragment containing an acknowledgement message consisting of a session key to the proxy
*/
func NewProxyAck(sessionKey []byte) *DataFragment {
	//generate Session Key
	return &DataFragment{Proxy: &ProxyRequest{Forward: true, SessionKey: &sessionKey}}
}

/*
	Split splits a DataFragment into n cloves
*/
func (df *DataFragment) Split(k uint, n uint) ([]*Clove, error) {
	if n == 0 {
		return nil, nil
	}
	marshal, err := protobuf.Encode(df)
	if err != nil {
		return nil, err
	}
	secrets, err := splitSecret(marshal, int(k), int(n))
	if err != nil {
		return nil, err
	}
	sn := make([]byte, 8)
	rand.Read(sn)
	cloves := make([]*Clove, len(secrets))
	for i, secret := range secrets {
		//generate uuid sequence number
		cloves[i] = &Clove{Threshold: uint32(k), Index: uint32(i) + 1, Data: secret, SequenceNumber: sn}
	}
	return cloves, nil
}

type ProxyRequest struct {
	Forward    bool
	SessionKey *[]byte
}

type Query struct {
	Keywords []string
}

type Content struct {
	Key  []byte
	Data []byte
}

/*
Content describes file delivery
*/
type Delivery struct {
	IP     string    // the provider only sends one initiator proxy per provider proxy
	Cloves [2][]byte // 2 protobofed(or other byte representation) cloves (because Delivery is meant to be split by 2-threshold)
}

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
