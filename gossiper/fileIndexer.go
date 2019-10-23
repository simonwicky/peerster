package gossiper 

import ("crypto/sha256"
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
	sha string
}

type FileStorage struct {
	data map[string] *FileData
	lock sync.RWMutex
}

var CHUNK_SIZE int64 = 8000
var SHARED_FILE_FOLDER = "_SharedFiles/"
var DOWNLOAD_FILE_FOLDER = "_Downloads/"

func NewFileStorage() *FileStorage {
	return &FileStorage{
		data: make(map[string] *FileData),
	}
}


func (fs *FileStorage) addFromSystem(name string){
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
		currentChecksum := sha256.Sum256(buffer[:n])
		//converting [32]byte to []byte
		bytes := currentChecksum[:]
		fileMetaData.metafile = append(fileMetaData.metafile,hex.EncodeToString(bytes))
		metafileBytes = append(metafileBytes,bytes...)
	}

	metafileChecksum := sha256.Sum256(metafileBytes)
	fileMetaData.sha = hex.EncodeToString(metafileChecksum[:])

	fs.data[fileMetaData.sha] = &fileMetaData
	fmt.Fprintln(os.Stderr,"Indexed file: " + fs.data[fileMetaData.sha].name)
	fmt.Fprintln(os.Stderr,"Id file: " + string(fileMetaData.sha))
}


func (fs *FileStorage) addFromDatadownloader(dd *Datadownloader){
	fd := FileData{
		name: dd.fileName,
		size: int64(len(dd.data)),
		sha : dd.id,
		metafile: make([]string,0),
	}
	for offset := 0; offset < len(dd.metafile) / 32 ; offset += 1 {
		chunkID_bytes := dd.metafile[offset * 32 : (offset + 1) * 32]
		fd.metafile = append(fd.metafile, hex.EncodeToString(chunkID_bytes))
	}
	fs.lock.Lock()
	fs.data[fd.sha] = &fd
	fs.lock.Unlock()
	fmt.Fprintln(os.Stderr,"Indexed file: " + fs.data[fd.sha].name)
	fmt.Fprintln(os.Stderr,"Id file: " + string(fd.sha))

	//add to file system
	file,err := os.Create(DOWNLOAD_FILE_FOLDER + dd.fileName)
	defer file.Close()
	if err != nil{
		fmt.Fprintln(os.Stderr,"File creation error")
		fmt.Fprintln(os.Stderr,err)
		return
	}
	_,err = file.Write(dd.data)
	if err != nil {
		fmt.Fprintln(os.Stderr,"Error writing file")
		fmt.Fprintln(os.Stderr,err)
		return
	}
	fmt.Fprintln(os.Stderr,"File saved successfully")
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

func (fs *FileStorage) getFile(fileData *FileData) *os.File{
	file,err := os.Open(SHARED_FILE_FOLDER + fileData.name)
	if err != nil {
		fmt.Fprintln(os.Stderr,"File not found")
		return nil
	}
	return file
}

func (fs *FileStorage) getFileChunk(fileData *FileData, chunk int) []byte{
	file,err := os.Open(SHARED_FILE_FOLDER + fileData.name)
	if err != nil {
		fmt.Fprintln(os.Stderr,"File not found")
		return nil
	}
	buffer := make([]byte,CHUNK_SIZE)
	n,err := file.ReadAt(buffer,int64(chunk) * CHUNK_SIZE)
	if err != nil && err != io.EOF {
		fmt.Fprintln(os.Stderr,err)
		return nil
	}
	return buffer[:n]

}
