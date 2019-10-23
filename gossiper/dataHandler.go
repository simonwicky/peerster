package gossiper

import ("github.com/simonwicky/Peerster/utils"
		"encoding/hex"
		"fmt"
		"os"
		"time"
)


type Datadownloader struct {
	waitingFor []byte
	timeout *time.Timer
	fileName string
	data []byte
	metafile []byte
	id string
	destination string
	replies chan *utils.DataReply
	g *Gossiper
	trial_counter int
}

var MAX_TRIAL int = 10

func (g *Gossiper) NewDatadownloader(request *utils.DataRequest, fileName string){
	dd := &Datadownloader{}
	dd.fileName = fileName
	dd.destination = request.Destination
	dd.replies = make(chan *utils.DataReply, 20)
	dd.waitingFor = request.HashValue
	dd.id = hex.EncodeToString(request.HashValue)
	dd.g = g
	dd.g.addDownloader(dd)
	dd.trial_counter = MAX_TRIAL
}

func (dd *Datadownloader) Start(){
	fmt.Fprintln(os.Stderr,"Starting downloader file id " + dd.id)
	utils.LogMetafile(dd.fileName, dd.destination)
	metafile := dd.requestAndReceiveData()
	if metafile == nil {
		return
	}
	dd.metafile = make([]byte,len(metafile.Data))
	copy(dd.metafile, metafile.Data)
	//loop through metafile, requestAndReceiveData the chunks
	for offset := 0; offset < len(dd.metafile) / 32 ; offset += 1 {
		utils.LogChunk(dd.fileName,dd.destination, offset)
		chunkID_bytes := dd.metafile[offset * 32 : (offset + 1) * 32]
		dd.waitingFor = chunkID_bytes
		chunk_reply := dd.requestAndReceiveData()
		if chunk_reply == nil {
			return
		}
		dd.data = append(dd.data, chunk_reply.Data...)
	}
	//save and index the file
	dd.g.fileStorage.addFromDatadownloader(dd)
	utils.LogReconstruct(dd.fileName)
}

func (dd *Datadownloader) requestAndReceiveData() *utils.DataReply{
	dr := &utils.DataRequest{
		Origin : dd.g.Name,
		Destination : dd.destination,
		HopLimit : 10,
		HashValue : dd.waitingFor,
	}
	dd.timeout = time.NewTimer(5 * time.Second)
	dd.g.sendPointToPoint(&utils.GossipPacket{DataRequest: dr}, dr.Destination)
	for {
		select {
			case reply := <- dd.replies :
				dd.trial_counter = MAX_TRIAL
				return reply
			default:
				select {
					case _ = <- dd.timeout.C :
						dd.timeout.Stop()
						dr.HopLimit = 10
						dd.trial_counter -= 1
						if dd.trial_counter <= 0 {
							fmt.Fprintln(os.Stderr,"Too much trial, aborting")
							return nil
						}
						dd.g.sendPointToPoint(&utils.GossipPacket{DataRequest: dr}, dr.Destination)
						dd.timeout = time.NewTimer(5 * time.Second)
					default:  
						time.Sleep(250 * time.Millisecond)
				}	
				
		}
	}
}



//================================
//Handling data request
//================================
func (g *Gossiper) replyData(request *utils.DataRequest){
	fileID := hex.EncodeToString(request.HashValue)
	g.fileStorage.lock.RLock()
	defer g.fileStorage.lock.RUnlock()
	//first look for metafile
	for _,fileData := range g.fileStorage.data{
		if fileID == fileData.sha {
			fmt.Fprintln(os.Stderr,"Found metafile")
			//send the metafile
			metafilebytes := assembleMetaFile(fileData.metafile)
			if metafilebytes == nil {
				fmt.Fprintln(os.Stderr,"Metafile malformed, could not send")
				return
			}
			g.sendData(metafilebytes, request)

			return
		}
	}
	//if not found in metafile, look for chunk
	for _,fileData := range g.fileStorage.data{
		for index, chunkID := range fileData.metafile {
			if chunkID == fileID {
				fmt.Fprintf(os.Stderr,"Found chunk n : %d\n",index)
				bytes := g.fileStorage.getFileChunk(fileData,index)
				if bytes == nil {
					fmt.Fprintln(os.Stderr,"Chunk not found, can not send")
				}
				g.sendData(bytes, request)
				return
			}
		}
	}
	fmt.Fprintln(os.Stderr,"Chunk not found, can not send")

}

func (g *Gossiper) sendData(bytes []byte, request *utils.DataRequest){
		reply := &utils.DataReply{
			Origin: g.Name,
			Destination: request.Origin,
			HopLimit: 10,
			HashValue: request.HashValue,
			Data: bytes,
		}
		g.sendPointToPoint(&utils.GossipPacket{DataReply : reply}, reply.Destination)
}

