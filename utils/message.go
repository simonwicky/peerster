package utils

import (
	"crypto/rand"

	"go.dedis.ch/protobuf"
)

// import ("math/big")

type Message struct {
	Text        string
	Destination *string
	File        *string
	Request     *[]byte
	Budget      *uint64
	Keywords    *string
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
	Message *AnonymousMessage
	Proxy   *ProxyRequest
	Query   *Query
	Content *Content
}

/*
NewDataFragment returns a DataFragment recovered from k cloves
*/
func NewDataFragment(cloves []*Clove) *DataFragment {
	threshold := len(cloves) //cloves[0].Threshold
	xs := make([]int, threshold)
	data := make([][]byte, threshold)
	for i, clove := range cloves {
		xs[i] = int(clove.Index)
		data[i] = clove.Data
	}
	marshalled := recoverSecret(data, xs)
	var df DataFragment
	err := protobuf.Decode(marshalled, &df)
	if err != nil {
		LogObj.Fatal(err.Error())
		return nil
	}
	return &df
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
func (df *DataFragment) Split(k uint, n uint) []*Clove {
	if n == 0 {
		return nil
	}
	marshal, err := protobuf.Encode(df)
	if err != nil {
		return nil
	}
	secrets := splitSecret(marshal, int(k), int(n))
	sn := make([]byte, 8)
	rand.Read(sn)
	cloves := make([]*Clove, len(secrets))
	for i, secret := range secrets {
		//generate uuid sequence number
		cloves[i] = &Clove{Threshold: uint32(k), Index: uint32(i) + 1, Data: secret, SequenceNumber: sn}
	}
	return cloves
}

type ProxyRequest struct {
	Forward    bool
	SessionKey *[]byte
}

type Query struct {
}

type Content struct {
}

type AnonymousMessage struct {
}

type GossipPacket struct {
	Simple        *SimpleMessage
	Rumor         *RumorMessage
	Status        *StatusPacket
	Private       *PrivateMessage
	DataRequest   *DataRequest
	DataReply     *DataReply
	SearchRequest *SearchRequest
	SearchReply   *SearchReply
	TLCMessage    *TLCMessage
	Ack           *TLCAck
	Clove         *Clove
}
