package gossiper

import ("github.com/simonwicky/Peerster/utils"
		"crypto/sha256"
		"os"
		"fmt"
		"encoding/hex"
		"io"
		"strings"
		"sync"
)

type FileData struct {
	name string
	size int64
	metafile []string
	data []byte
	metafileHash []byte
	sha string
	chunkmap []uint64
	metafileLocation string
	chunkLocation map[uint64] string
	local bool
}

type FileStorage struct {
	data map[string] *FileData
	lock sync.RWMutex
}

var CHUNK_SIZE int64 = 8192
var SHARED_FILE_FOLDER = "_SharedFiles/"
var DOWNLOAD_FILE_FOLDER = "_Downloads/"

func NewFileStorage() *FileStorage {
	return &FileStorage{
		data: make(map[string] *FileData),
	}
}


func (fs *FileStorage) addFromSystem(g* Gossiper, name string){
	file,err := os.Open(SHARED_FILE_FOLDER + name)
	defer file.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr,"File not found, cannot index it")
		return
	}
	stat,err := file.Stat()
	if err != nil {
		fmt.Fprintln(os.Stderr,"Could not acquire stat, cannot index it")
		return
	}
	var fileMetaData FileData
	fileMetaData.name = name
	fileMetaData.size = stat.Size()

	var metafileBytes []byte

	for offset := int64(0); offset < (fileMetaData.size / CHUNK_SIZE) + 1 ; offset += 1 {
		buffer := make([]byte,CHUNK_SIZE)
		n,err := file.ReadAt(buffer,offset * CHUNK_SIZE)
		if err != nil && err != io.EOF {
			fmt.Fprintln(os.Stderr,err)
			return
		}
		fileMetaData.data = append(fileMetaData.data,buffer...)
		fileMetaData.chunkmap = append(fileMetaData.chunkmap,uint64(offset))
		currentChecksum := sha256.Sum256(buffer[:n])
		//converting [32]byte to []byte
		bytes := currentChecksum[:]
		fileMetaData.metafile = append(fileMetaData.metafile,hex.EncodeToString(bytes))
		metafileBytes = append(metafileBytes,bytes...)
	}

	metafileChecksum := sha256.Sum256(metafileBytes)
	fileMetaData.metafileHash = metafileChecksum[:]
	fileMetaData.sha = hex.EncodeToString(metafileChecksum[:])
	fileMetaData.local = true
	fs.lock.Lock()
	fs.data[fileMetaData.sha] = &fileMetaData
	fmt.Fprintln(os.Stderr,"Indexed file: " + fs.data[fileMetaData.sha].name)
	fmt.Fprintln(os.Stderr,"Id file: " + string(fileMetaData.sha))
	fs.lock.Unlock()

	//TLCMessage
	if g.hw3ex2 {
		infos := utils.FileInfo{
			Name: fileMetaData.name,
			Size: fileMetaData.size,
			MetafileHash : fileMetaData.metafileHash,
		}
		Publish(g, &infos)
	}
}



func (fs *FileStorage) createFile(filename, id string){
	fd := FileData{
		name: filename,
		size: int64(0),
		sha : id,
		metafile: make([]string,0),
		data : make([]byte,0),
		chunkmap : make([]uint64,0),
		chunkLocation : make(map[uint64] string),
		local : false,
	}
	fd.metafileHash,_ = hex.DecodeString(id)
	fs.lock.Lock()
	fs.data[fd.sha] = &fd
	fmt.Fprintln(os.Stderr,"Indexed file: " + fs.data[fd.sha].name)
	fmt.Fprintln(os.Stderr,"Id file: " + string(fd.sha))
	fs.lock.Unlock()
}

func (fs *FileStorage) addFile(data *FileData){
	if _,ok := fs.data[data.sha]; ok {
		fmt.Fprintln(os.Stderr,"File already exists")
		return
	}
	fs.lock.Lock()
	fs.data[data.sha] = data
	fmt.Fprintln(os.Stderr,"Indexed file: " + fs.data[data.sha].name)
	fmt.Fprintln(os.Stderr,"Id file: " + string(data.sha))
	fs.lock.Unlock()
}

func (fs *FileStorage) addChunk(chunk []byte, id string, chunkNumber int) {
	fs.lock.Lock()
	defer fs.lock.Unlock()
	fd := fs.data[id]
	fd.data = append(fd.data,chunk...)
	fd.size += int64(len(chunk))
	fd.metafile = append(fd.metafile, hex.EncodeToString(chunk))
}

func (fs *FileStorage) saveToDisk(id, filename string){
	fs.lock.RLock()
	defer fs.lock.RUnlock()
	fd := fs.data[id]
	//add to file system
	file,err := os.Create(DOWNLOAD_FILE_FOLDER + filename)
	defer file.Close()
	if err != nil{
		fmt.Fprintln(os.Stderr,"File creation error")
		fmt.Fprintln(os.Stderr,err)
		return
	}
	_,err = file.Write(fd.data)
	if err != nil {
		fmt.Fprintln(os.Stderr,"Error writing file")
		fmt.Fprintln(os.Stderr,err)
		return
	}
	fmt.Fprintln(os.Stderr,"File saved successfully")
	fd.local = true
}

func (fs *FileStorage) checkFile(id string) bool{
	fs.lock.RLock()
	defer fs.lock.RUnlock()
	fd := fs.data[id]

		var metafileBytes []byte

	for offset := int64(0); offset < (fd.size / CHUNK_SIZE) + 1 ; offset += 1 {
		upperbound := (offset + 1) * CHUNK_SIZE
		if upperbound > fd.size{
			upperbound = fd.size
		}
		currentChecksum := sha256.Sum256(fd.data[offset * CHUNK_SIZE : upperbound])
		//converting [32]byte to []byte
		bytes := currentChecksum[:]
		metafileBytes = append(metafileBytes,bytes...)
	}

	metafileChecksum := sha256.Sum256(metafileBytes)
	sha := hex.EncodeToString(metafileChecksum[:])
	return sha == id
}

func (fs *FileStorage) deleteFile(id string) {
	fs.lock.Lock()
	delete(fs.data,id)
	fs.lock.Unlock()
}

func assembleMetaFile(metafile []string) []byte{
	hexdump := strings.Join(metafile,"")
	bytes,err := hex.DecodeString(hexdump)
	if err != nil {
		fmt.Fprintln(os.Stderr,"Malformed meta file, exiting")
		fmt.Fprintln(os.Stderr,err)
		return nil
	}
	return bytes
}


func (fs *FileStorage) getFileChunk(fileData *FileData, chunk int) []byte{
	lowerbound := int64(chunk) * CHUNK_SIZE
	upperbound := (int64(chunk) + 1) * CHUNK_SIZE
	if lowerbound > fileData.size {
		return []byte{}
	}
	if upperbound > fileData.size {
		upperbound = fileData.size
	}
	buffer := make([]byte,upperbound-lowerbound)
	copy(buffer, fileData.data[lowerbound:upperbound])
	return buffer
}

func (fs *FileStorage) lookupFile(keyword string) []*FileData {
	results := make([]*FileData,0)
	fs.lock.RLock()
	defer fs.lock.RUnlock()
	for _, fd := range fs.data {
		if strings.Contains(fd.name, keyword) {
			results = append(results,fd)
		}
	}
	return results
}



func (fs *FileStorage) asSearchResults()[]*utils.SearchResult{
	var results []*utils.SearchResult
	fs.lock.RLock()
	defer fs.lock.RUnlock()
	for _, fd := range fs.data {
		result := &utils.SearchResult{
			FileName: fd.name,
			MetafileHash: make([]byte, len(fd.metafileHash)),
			ChunkCount: uint64(fd.size / int64(2 << 12)),
		}
		copy(result.MetafileHash, fd.metafileHash)
		results = append(results,result)
	}
	return results
}
