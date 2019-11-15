package gossiper

import ("github.com/simonwicky/Peerster/utils"
		"encoding/hex"
		"fmt"
		"os"
		"time"
)

type Match struct {
	fileData *FileData
	chunkCount uint64
	complete bool
	name string
	metafileHash []byte
}
type FileSearcher struct {
	running bool
	replies chan *utils.SearchReply
	keywords []string
	budget uint64
	g *Gossiper
	matches []*Match
}


func NewFileSearcher(g *Gossiper) *FileSearcher {
	return &FileSearcher{
		g : g,
		replies : make(chan *utils.SearchReply),
		running : false,
	}
}

func (s *FileSearcher) Start(budget uint64, keyword []string){
	s.running = true
	s.budget = budget
	s.keywords = keyword
	s.matches = make([]*Match,0)
	go s.search(budget, keyword)
	s.handleReply()
	s.running = false
}

func (s *FileSearcher) getMatches() []*Match {
	return s.matches
}

func (s *FileSearcher) search(budget uint64, keyword []string){
	for {
		searchRequest := &utils.SearchRequest{
			Origin : s.g.Name,
			Budget : s.budget,
			Keywords : s.keywords,
		}
		if !s.running {
			return
		}
		fmt.Fprintf(os.Stderr,"Sending search with budget %d\n", searchRequest.Budget)
		s.g.relayRequest(searchRequest)
		if s.budget >= 32 {
			return
		}

		s.budget *= 2
		time.Sleep(1 * time.Second)
	}
}

func(s *FileSearcher) handleReply(){
	matchCount := 0
	timeout := time.NewTimer(10 * time.Second)
	for {
		if matchCount >= 2 {
			utils.LogSearchFinished()
			return
		}
		select {
			case reply := <- s.replies:
				timeout.Reset(10 * time.Second)
				for _,result := range reply.Results {
					utils.LogFileFound(result.FileName, reply.Origin, hex.EncodeToString(result.MetafileHash), result.ChunkMap)
					is_new_match := true
					for _,match := range s.matches {
						if match.name == result.FileName && hex.EncodeToString(result.MetafileHash) == hex.EncodeToString(match.metafileHash){
							is_new_match = false
							fd := match.fileData
							if match.complete {
								matchCount += 1
							} else {
								for _,chunkId := range result.ChunkMap {
									for index,id := range fd.chunkmap {
										if chunkId == id {
											break
										}
										if chunkId < id {
											fd.chunkmap = append(fd.chunkmap[:index],append([]uint64{id},fd.chunkmap[index:]...)...)
											fd.chunkLocation[chunkId] = reply.Origin
											fmt.Println("ADDING CHUNk")
											break
										}
										if index == len(fd.chunkmap) - 1  && chunkId > id {
											fd.chunkmap = append(fd.chunkmap,id)
											fd.chunkLocation[chunkId] = reply.Origin
											fmt.Println("ADDING CHUNk")
											break
										}
									}
								}
								if uint64(len(fd.chunkmap)) == match.chunkCount {
									matchCount += 1
									match.complete = true
									s.g.fileStorage.addFile(fd)
								}
							}

						}
					}
					if is_new_match {
						//new file
						fileData := &FileData{
							name : result.FileName,
							size : int64(0),
							metafileHash : make([]byte,len(result.MetafileHash)),
							metafileLocation : reply.Origin,
							sha : hex.EncodeToString(result.MetafileHash),
							chunkmap : result.ChunkMap,
							local : false,
							chunkLocation : make(map[uint64]string,0),
						}
						copy(fileData.metafileHash,result.MetafileHash)
						for _,chunkId := range result.ChunkMap {
							fileData.chunkLocation[chunkId] = reply.Origin
						}
						new_match := &Match{
							chunkCount : result.ChunkCount,
							fileData : fileData,
							complete : false,
							name : result.FileName,
							metafileHash : result.MetafileHash,
						}
						s.matches = append(s.matches,new_match)
						if uint64(len(fileData.chunkmap)) == new_match.chunkCount {
							matchCount += 1
							new_match.complete = true
							s.g.fileStorage.addFile(fileData)
						}

					}
				}
			default:
				time.Sleep(10 * time.Millisecond)
			    select {
    				case _ = <- timeout.C:
    					fmt.Fprintln(os.Stderr,"Search timeout")
    					return
    				default:
    			}
		}
	}
}