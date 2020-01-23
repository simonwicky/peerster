package gossiper

import (
	"sync"
	"github.com/simonwicky/Peerster/utils"
)

//Files routing table to keep track of the possible paths to each file
type FilesRouting struct {
	sync.Mutex
	filesRoutes map[string] *FileRoutes
}

type FileRoutes struct {
	fileInfo utils.FileInfo
	routes []string
}

func createFileRoute(fileInfo utils.FileInfo, route string) *FileRoutes {
	return  &FileRoutes{
		fileInfo: fileInfo, 
	}
}

func NewFilesRouting() FilesRouting {
	return FilesRouting{filesRoutes: make(map[string]*FileRoutes)}
}



func (filesRouting *FilesRouting) addRoute(fileInfo utils.FileInfo, route string){
	filesRouting.Lock()
	defer filesRouting.Unlock()

	if fileRoutes, found := filesRouting.filesRoutes[utils.HexToString(fileInfo.MetafileHash)]; found {
		fileRoutes.routes = append(fileRoutes.routes, route)
	}else {
		filesRouting.filesRoutes[utils.HexToString(fileInfo.MetafileHash)] = createFileRoute(fileInfo, route)
	}
}

func (filesRouting *FilesRouting) GetRoutes(metaFileHash string) []string{
	filesRouting.Lock()
	defer filesRouting.Unlock()
	var routes []string
	if fileRoutes, found := filesRouting.filesRoutes[metaFileHash]; found{
		routes = fileRoutes.routes
	}
	return routes
}

/*
	Returns the routes where to send the search request in an optimal order.
	Routes are sorted on a longest prefix match basis.
*/
func (filesRouting *FilesRouting) RoutesSorted(keywords []string) (peers []string ){
	return nil
}

func (filesRouting *FilesRouting) UpdateRouting(reply *utils.GCSearchReply){
	for _, result := range reply.Results {
		if len(result.ChunkMap) == int(result.ChunkCount){
			fileInfo := utils.FileInfo {
				Name: result.FileName, 
				MetafileHash: result.MetafileHash,
			}
			filesRouting.addRoute(fileInfo, reply.Origin)
		}
	}
}