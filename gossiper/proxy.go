//Author: Frédéric Gessler

package gossiper

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/simonwicky/Peerster/utils"
	"go.dedis.ch/protobuf"
)

type Settings struct {
	SessionKeySize uint
	Buffering      uint
	Redundancy     uint          // number of cloves sent for any threshold
	DiscoveryRate  time.Duration // period of proxy discovery
	Connectivity   uint          // d used in the paper = number of proxies for search for example
	ProxyMocking   bool
}

/*
getTuple returns n paths
*/
func getTuple(n uint, pathsTaken map[string]bool, peers []string) ([]string, map[string]bool, error) {
	//shuffle known peers
	rand.Shuffle(len(peers), func(i, j int) {
		tmp := peers[i]
		peers[i] = peers[j]
		peers[j] = tmp
	})
	tuple := make([]string, n)
	var i uint = 0
	for _, peer := range peers {
		if taken, ok := pathsTaken[peer]; !ok || !taken {
			tuple[i] = peer
			i++
			pathsTaken[peer] = true
			if i >= n {
				return tuple, pathsTaken, nil
			}
		}
	}
	return tuple[:i], pathsTaken, errors.New(fmt.Sprint("Not enough available paths in", peers, "of", pathsTaken, "!"))
}

/*
Proxy describes a proxy in an abstract manner
*/
type Proxy struct {
	Paths      [2]string
	SessionKey []byte
	ProxySN    []byte
	IP         string
}

/*
ProxyPool is a thread-safe store for proxies with convenience methods
*/
type ProxyPool struct {
	sync.RWMutex
	proxies []*Proxy
}

/*
Cover returns a map of all the paths taken by the pool's proxies in aggregate
*/
func (pool *ProxyPool) Cover() map[string]bool {
	pathsTaken := map[string]bool{}
	for _, proxy := range pool.proxies {
		pathsTaken[proxy.Paths[0]] = true
		pathsTaken[proxy.Paths[1]] = true
	}
	return pathsTaken
}

/*
Add adds a new proxy to the pool
*/
func (pool *ProxyPool) Add(proxy *Proxy) {
	pool.Lock()
	defer pool.Unlock()
	pool.proxies = append(pool.proxies, proxy)
}

/*
GetD returns d random proxies from the ProxyPool
*/
func (pool *ProxyPool) GetD(d uint) ([]*Proxy, error) {
	pool.Lock()
	defer pool.Unlock()
	if uint(len(pool.proxies)) < d {
		return nil, errors.New("could not find d proxies")
	}
	rand.Shuffle(len(pool.proxies), func(i, j int) {
		tmp := pool.proxies[i]
		pool.proxies[i] = pool.proxies[j]
		pool.proxies[j] = tmp
	})
	return pool.proxies[:d], nil
}

/*
initiate creates a new proxy init message,
splits it in n cloves, gets n paths from the known peers
and sends a clove to each path
*/
func (g *Gossiper) initiate(pathsTaken map[string]bool) map[string]bool {
	knownPeers := g.knownPeers
	//series := utils.LogObj.Named("init")
	tuple, pathsStillAvailable, err := getTuple(g.settings.Redundancy, pathsTaken, knownPeers)
	if err == nil {
		cloves, err := utils.NewProxyInit().Split(2, g.settings.Redundancy)
		//test
		if err == nil {
			for i, clove := range cloves {
				//series.Debug(string(clove.SequenceNumber), "(", i, ") = ", string(clove.Data))
				g.sendToPeer(tuple[i], clove.Wrap())
			}
		} else {
			utils.LogObj.Fatal(err.Error())
		}
	} else {
		//utils.LogObj.Warn(err.Error())
	}
	return pathsStillAvailable
}

/*
initiator maintains a proxy pool by periodically trying to discover new ones
BIG QUESTION: is it enough to take distincts pairs or do _ALL_ the paths have to be distinct
	- distinct pairs:
	- distinct paths:

	let's assume distinct paths(one path = one and only one proxy)
*/
func (g *Gossiper) initiator(n uint, peersAtBootstrap []string, peersUpdates chan []string) {
	logger := utils.LogObj.Named("init")
	pathsTaken := map[string]bool{}
	pool := g.proxyPool
	g.initiate(pathsTaken)
	ini := time.NewTicker(time.Second * g.settings.DiscoveryRate)
	for {
		select {
		case <-ini.C:
			pathsTaken = pool.Cover()
			//initiate proxy search
			pathsTaken = g.initiate(pathsTaken)
		case newProxy := <-g.newProxies:
			logger.Debug("New proxy")
			pool.Add(newProxy)
			sessionKey := make([]byte, g.settings.SessionKeySize)
			rand.Read(sessionKey) // always return nil error per documentation
			cloves, err := utils.NewProxyAck(sessionKey).Split(2, 2)
			if err == nil {
				for i, clove := range cloves {
					logger.Debug("sent ack ", string(clove.SequenceNumber), " to ", newProxy.Paths[i])
					g.sendToPeer(newProxy.Paths[i], clove.Wrap())
				}
			} else {
				utils.LogObj.Fatal(err.Error())
			}
		}
	}
}

/*
NewContent creates a new deliverable content
*/
func NewDeliveries(data []byte, initiatorProxies []string, k, nPrime uint) ([]*utils.DataFragment, error) {
	//assert len(initiatorProxies) == nPrime / 2
	dPrime := nPrime / 2
	deliveries := make([]*utils.DataFragment, dPrime)
	// assert nPrime is even
	content, err := NewContent(data)
	if err != nil {
		return nil, err
	}
	contentCloves, err := content.Split(k, nPrime)
	if err != nil {
		return nil, err
	}
	for i := 0; i < int(dPrime); i++ {
		encodedCloves := [2][]byte{}
		for j := 0; j < 2; j++ {
			encoded, err := protobuf.Encode(contentCloves[i+j])
			if err != nil {
				return nil, err
			}
			encodedCloves[j] = encoded
		}
		delivery := &utils.Delivery{IP: initiatorProxies[i], Cloves: encodedCloves}
		deliveries[i] = &utils.DataFragment{Delivery: delivery}
	}
	//cihpherText := gcm.Seal(nonce, nonce, data, nil)
	// data is
	return deliveries, nil
}

func NewContent(data []byte) (*utils.DataFragment, error) {
	//generate a key K
	key := make([]byte, 32) // needs to be 32 bytes?
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	encrypter, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//encrypt F with AES K
	//encrypt
	cbc := cipher.NewCBCEncrypter(encrypter, []byte(utils.AES_IV))
	cipherText := make([]byte, len(data))
	cbc.CryptBlocks(cipherText, data)
	df := &utils.DataFragment{Content: &utils.Content{
		Key:  key,
		Data: cipherText,
	}}
	return df, nil
}

func (g *Gossiper) deliver(filename string, proxies []string) {
	logger := utils.LogObj.Named("delivery")
	d := uint(len(proxies))
	//get file from filestorage
	providerProxies, err := g.proxyPool.GetD(d)
	if err != nil {
		logger.Fatal(err.Error())
		return
	}
	file, ok := g.fileStorage.data[filename]
	if !ok {
		return
	}
	n := 2 * d
	deliveries, err := NewDeliveries(file.data, proxies, 4, n)
	if err != nil {
		logger.Fatal(err.Error())
		return
	}
	for i, delivery := range deliveries {
		cloves, err := delivery.Split(2, 2) // normally we should collect all cloves to make sure that no error occurs
		if err != nil {
			logger.Fatal(err.Error())
			return
		}
		for j, clove := range cloves {
			//clove.SequenceNumber.encryp(proxies[j].SessionKey)
			//clove.Encrypted = true
			g.sendToPeer(providerProxies[i].Paths[j], clove.Wrap())
		}
	}
	//split F
	//split K

	//generate sequence number

}

func (g *Gossiper) proxySrv() {
	var buf []byte = make([]byte, 8000)
	atTCP, err := net.ResolveTCPAddr("tcp", g.directProxyPort)
	if err != nil {
		utils.LogObj.Fatal(err.Error(), " dropping cloves")
		return
	}
	connect, err := net.DialTCP("tcp", nil, atTCP)
	for {
		_, err := connect.Read(buf)
		if err == nil {
			var clove utils.Clove
			err = protobuf.Decode(buf, &clove)
			if err == nil {
				// forward clove to clove handler
				g.clovesCollector.directs <- &clove
			} else {
				utils.LogObj.Fatal(err.Error())
			}
		} else {
			utils.LogObj.Fatal(err.Error())
		}
	}
}

func (g *Gossiper) proxyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	//utils.LogObj.Warn("proxies ", g.proxyPool)
	paths := [][]string{}
	var proxiesJson []map[string]string = make([]map[string]string, 0)
	var proxies []*Proxy
	if g.settings.ProxyMocking {
		if g.Name == "Alice" {
			mockProxies := []*Proxy{&Proxy{
				IP:    "127.0.0.1:5008",
				Paths: [2]string{"127.0.0.1:5001", "127.0.0.1:5001"},
			}, &Proxy{
				IP:    "127.0.0.1:5009",
				Paths: [2]string{"127.0.0.1:5003", "127.0.0.1:5004"},
			}}
			proxies = mockProxies
		} else {
			proxies = []*Proxy{}
		}
	} else {
		proxies = g.proxyPool.proxies
	}
	for _, proxy := range proxies {
		paths = append(paths, proxy.Paths[:])
		proxiesJson = append(proxiesJson, map[string]string{
			"IP": proxy.IP,
			"_1": proxy.Paths[0],
			"_2": proxy.Paths[1],
		})
	}
	data, err := json.Marshal(map[string][]map[string]string{
		"proxies": proxiesJson,
	})
	if err != nil {
		utils.LogObj.Fatal(err.Error())
	} else {
		w.Write(data)
	}

}

func (g *Gossiper) getRandomPeer(exclude string) *string {
	if len(g.knownPeers) <= 0 {
		return nil
	}
	peer := g.knownPeers[rand.Intn(len(g.knownPeers))]
	if peer == exclude {
		return g.getRandomPeer(exclude)
	}
	return &peer
}

type IncomingClove struct {
	predecessor string
	clove       *utils.Clove
}

/*
ClovesCollector is a LRU store for cloves with a handler and a "garbage collector"
cloves are stored in a map[sequence-number]map[predecessor]map[index] to provide a good mix
of fast lookup insertion deletion, and to be able to store multiple cloves from a predecessor
but not store many of the same exact cloves from the same exact predecessor
*/
type ClovesCollector struct {
	sync.Mutex
	handler      chan IncomingClove
	directs      chan *utils.Clove
	cloves       map[string]map[string]map[uint32]*utils.Clove
	routingTable map[string]string // records previous hop -> next hop
}

func NewClovesCollector(g *Gossiper) *ClovesCollector {
	cc := &ClovesCollector{
		handler:      make(chan IncomingClove),
		directs:      make(chan *utils.Clove),
		cloves:       make(map[string]map[string]map[uint32]*utils.Clove),
		routingTable: make(map[string]string),
	}
	cc.manage(g)
	return cc
}

func CloneValue(source interface{}, destin interface{}) {
	x := reflect.ValueOf(source)
	if x.Kind() == reflect.Ptr {
		starX := x.Elem()
		y := reflect.New(starX.Type())
		starY := y.Elem()
		starY.Set(starX)
		reflect.ValueOf(destin).Elem().Set(y.Elem())
	} else {
		destin = x.Interface()
	}
}

/*
Add is a thread unsafe method to add a clove to the collector(it is meant to be used in a isolated context)
*/
func (cc *ClovesCollector) Add(clove *utils.Clove, predecessor string) bool {
	str := string(clove.Data)
	cc.Lock()
	defer cc.Unlock()
	var sequenceNumber string = string(clove.SequenceNumber)
	//make sure there is storage for that sequence number
	if _, ok := cc.cloves[sequenceNumber]; !ok {
		cc.cloves[sequenceNumber] = make(map[string]map[uint32]*utils.Clove)
	}
	//make sure there is storage for that predecessor
	if _, ok := cc.cloves[predecessor]; !ok {
		cc.cloves[sequenceNumber][predecessor] = make(map[uint32]*utils.Clove)
	}
	//store the clove; make sure to deep copy clove data
	idx := clove.Index
	if _, ok := cc.cloves[sequenceNumber][predecessor][idx]; !ok {
		cc.cloves[sequenceNumber][predecessor][idx] = &utils.Clove{
			Index:          clove.Index,
			Threshold:      clove.Threshold,
			Data:           make([]byte, len(clove.Data)),
			SequenceNumber: make([]byte, len(clove.SequenceNumber)),
			Canary:         clove.Canary,
		}
		copy(cc.cloves[sequenceNumber][predecessor][idx].SequenceNumber[:], clove.SequenceNumber[:])
		copy(cc.cloves[sequenceNumber][predecessor][idx].Data[:], clove.Data[:])
		if str != string(cc.cloves[sequenceNumber][predecessor][idx].Data) {
			utils.LogObj.Fatal("storing error encountered!")
		}
		return true
	}
	//check if the threshold is met for that sequence numnber
	return false
}

/*
MeetsThreshold checks if there are k cloves matching the given sequence-number in the collector
		Basically we have to check if there are k ways to choose cloves with both distinct
		predecessors and indices. Generally, this is NP-complete
*/
func (cc *ClovesCollector) MeetsThreshold(sn string, k uint32) (bool, []*utils.Clove, []string) {
	if seq, ok := cc.cloves[sn]; ok {
		if uint32(len(seq)) >= k {
			ids, paths, cover := pathsCovered(seq, k)
			return getKIndependentCloves(k, seq, paths, ids, cover, make([]*utils.Clove, 0), []string{})
		}
		// there are less paths than k
		return false, []*utils.Clove{}, []string{}
	}
	utils.LogObj.Fatal("sequence number ", sn, " not found")
	return false, []*utils.Clove{}, []string{}
}

/*
pathsCovered given the cloves received for a particular sequence number returns a list of all the clove indices, paths that are available (where a proxy wasn't already discovered
and the "inverted" sequence of cloves from index to a list of predecessors). Its purpose is
to help find k cloves coming from different paths and having different indices
*/
func pathsCovered(seq map[string]map[uint32]*utils.Clove, k uint32) ([]uint32, map[string]bool, map[uint32][]string) {
	invertedSeq := make(map[uint32][]string) //map[index][]predecessor
	availablePaths := make(map[string]bool)
	ids := make([]uint32, 0)
	for predecessor, indices := range seq {
		availablePaths[predecessor] = true
		for index := range indices {
			ids = append(ids, index)
			if _, ok := invertedSeq[index]; !ok {
				invertedSeq[index] = make([]string, 0)
			}
			invertedSeq[index] = append(invertedSeq[index], predecessor)
		}
	}
	return ids, availablePaths, invertedSeq
}

/*
removeAtI removes an element in place
Adapted from https://yourbasic.org/golang/delete-element-slice/
*/
func removeAtI(i int, a []uint32) []uint32 {
	// Remove the element at index i from a.
	tmp := a[i]
	a[i] = a[len(a)-1] // Copy last element to index i.
	a[len(a)-1] = tmp
	//a[len(a)-1] = uint32(0)   // Erase last element (write zero value).
	a = a[:len(a)-1] // Truncate slice.
	return a
}

/*
getKIndependentCloves returns a tuple of a boolean indicating whether k different cloves have come from
k different paths, the list of cloves if it exists and the list of paths if they exist.
The subtlety is that if many cloves came from the same path (which can happen with loops),
then we get to choose which clove of that path contribute to recovering the message.
-
*/
func getKIndependentCloves(k uint32, seq map[string]map[uint32]*utils.Clove, pathIsAvailable map[string]bool, indices []uint32, inv map[uint32][]string, resa []*utils.Clove, resb []string) (bool, []*utils.Clove, []string) {
	if k == 0 {
		return true, resa, resb
	}
	//fmt.Println(res, k, indices)
	for _, index := range indices {
		for _, predecessor := range inv[index] {
			if pathIsAvailable[predecessor] {
				pathIsAvailable[predecessor] = false
				//check if seen[string(seq[predecessor][index].Data)]
				newIndices := make([]uint32, 0)
				for _, other := range indices {
					if other != index {
						newIndices = append(newIndices, other)
					}
				}
				if ok, cloves, paths := getKIndependentCloves(k-1, seq, pathIsAvailable, newIndices, inv, append(resa, seq[predecessor][index]), append(resb, predecessor)); ok {
					return true, cloves, paths
				}
			}
		}
	}
	return false, []*utils.Clove{}, []string{}
}

func (cc *ClovesCollector) cloveHandler(g *Gossiper, clove *utils.Clove, predecessor string) {
	//rec := utils.LogObj.Named("rec")
	var sequenceNumber string = string(clove.SequenceNumber)
	logger := utils.LogObj.Named("rec")

	//store clove by sequence number
	cc.Add(clove, predecessor)
	forwarding := false
	p := 0.8
	if met, cloves, paths := cc.MeetsThreshold(sequenceNumber, clove.Threshold); met {
		//logger.Debug("recovered clove from", paths)
		df, err := utils.NewDataFragment(cloves)
		if err == nil {
			logger.Debug(g.Name, "recovered clove")
			switch {
			case df.Proxy != nil:
				if df.Proxy.Forward {
					if df.Proxy.SessionKey == nil {
						logger.Debug("ProxyInit")
						output, err := utils.NewProxyAccept(g.directProxyPort).Split(2, 2)
						if err == nil {
							//accept to be a proxy
							for i, path := range paths {
								//logger.Debug("sent accept clove to ", path)
								logger.Debug("sending ACCEPT to ", path)
								g.sendToPeer(path, output[i].Wrap())
							}
						}
					} else {
						logger.Debug("ProxyAck")
						// register session key and id
						logger.Debug("registering session key")
					}
				} else {
					logger.Debug("ProxyAccept")
					var fixPaths [2]string
					copy(fixPaths[:], paths[:2])
					// record proxy and send session key
					if df.Proxy.IP != nil {
						g.newProxies <- &Proxy{Paths: fixPaths, IP: *df.Proxy.IP}
					}
				}
			case df.Delivery != nil: // this is read by a provider proxy
				logger.Debug("delivery")
				//directly connect by TCP to proxy provided
				atTcp, err := net.ResolveTCPAddr("tcp", df.Delivery.IP)
				if err != nil {
					utils.LogObj.Fatal(err.Error(), " dropping cloves")
					return
				}
				connect, err := net.DialTCP("tcp", nil, atTcp)
				if err != nil {
					utils.LogObj.Fatal(err.Error(), " dropping cloves")
					return
				}
				for _, dataClove := range df.Delivery.Cloves {
					_, err := connect.Write(dataClove)
					if err != nil {
						utils.LogObj.Fatal(err.Error())
					}
				}
			case df.Content != nil:
				//index file
			case df.Query != nil :
				searcher := g.getGCFileSearcher()
				go searcher.startSearch(df.Query.Keywords, &df.Query.SessionKey)
				//send results back to initiator
			case df.FileInfo != nil:
				g.FilesRouting.addOwnerPath(*df.FileInfo,paths)

			}
		} else {
			//this sequence number is corrupted, drop whole serie
			logger.Fatal(g.Name, "> ", err.Error(), " ", cc.cloves[sequenceNumber])
			for predecessor, cloves := range cc.cloves[sequenceNumber] {
				for index, clove := range cloves {
					logger.Fatal(g.Name, "> (", index, " = ", clove.Index, ") from:", predecessor, " ", clove.Data[:5], clove.Data[len(clove.Data)-5:], "=", clove.Canary, " => ", []byte(clove.Canary)[:5])
				}
			}
			delete(cc.cloves, sequenceNumber)
			forwarding = true
		}
	} else {
		forwarding = true
	}
	if forwarding { // !full
		//if successor, ok := cc.routingTable[predecessor]; ok && successor != g.addressPeer.String() {
		//	logger.Debug(g.Name, " sending ", clove.SequenceNumber, " to ", successor, " from ", predecessor)
		//	g.sendToPeer(successor, clove.Wrap())
		//} else {
		//forward to one random neighbour
		if rand.Float64() < p {
			if successor := g.getRandomPeer(predecessor); successor != nil {
				logger.Debug(g.Name, " forwarding ", clove.SequenceNumber, " to ", successor, " from ", predecessor)
				if _, ok := cc.routingTable[predecessor]; !ok {
					cc.routingTable[predecessor] = *successor
				}
				if _, ok := cc.routingTable[*successor]; !ok {
					cc.routingTable[*successor] = predecessor
				}
				g.sendToPeer(*successor, clove.Wrap())
			} else {
				logger.Warn("could not get no successor!")
			}
		}
		//}

	}
}

/*
manage is a forwarder/handler of cloves coupled with a state resetter that will delete all the cloves
every x seconds
*/
func (cc *ClovesCollector) manage(g *Gossiper) {
	if g == nil {
		return
	}
	logger := utils.LogObj.Named("man")
	cleaningTime := time.NewTicker(15 * time.Second)
	go func() {
		for {
			select {
			//case clove := <-cc.directs:
			// look up "cloves routing table" and forward
			case newClove := <-cc.handler:
				cc.cloveHandler(g, newClove.clove, newClove.predecessor)
			case <-cleaningTime.C:
				logger.Debug("clearing cloves", len(cc.cloves))
				cc.cloves = make(map[string]map[string]map[uint32]*utils.Clove)
				runtime.GC()
			}
		}
	}()
}
