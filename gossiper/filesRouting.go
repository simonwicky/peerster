package gossiper

import (
	"sync"
	"github.com/simonwicky/Peerster/utils"
	"sort"
	"strings"
)

//Files routing table to keep track of the possible paths to each file
type FilesRouting struct {
	sync.Mutex
	filesRoutes map[string] *FileRoutes
}

type FileRoutes struct {
	FileInfo utils.FileInfo
	Routes []string
}

func createFileRoute(fileInfo utils.FileInfo, routes []string) *FileRoutes {
	return  &FileRoutes{
		FileInfo: fileInfo, 
		Routes: routes,
	}
}

func NewFilesRouting() FilesRouting {
	return FilesRouting{filesRoutes: make(map[string]*FileRoutes)}
}

func (filesRouting *FilesRouting) GetRoutes(metaFileHash string) []string{
	filesRouting.Lock()
	defer filesRouting.Unlock()
	var routes []string
	if fileRoutes, found := filesRouting.filesRoutes[metaFileHash]; found{
		routes = fileRoutes.Routes
	}
	return routes
}


/*
	Returns the routes where to send the search request in an optimal order.
	Routes are sorted on a longest prefix match basis.
*/
func (filesRouting *FilesRouting) RoutesSorted(keywords []string) []FileRoutes {
	filesRoutes := filesRouting.GetAllRoutes()
	filescpy := make([]FileRoutes, len(filesRoutes))
	copy(filesRoutes, filescpy)
	sort.Slice(filescpy, func(i, j int) bool {
		return len(longestMatch(filescpy[i], keywords)) < len(longestMatch(filescpy[j], keywords))
	})
	return filescpy
}

//Updates the routing table according to the search reply
func (filesRouting *FilesRouting) UpdateRouting(reply *utils.GCSearchReply){
	for _, result := range reply.AccessibleFiles {
		if len(result.ChunkMap) == int(result.ChunkCount){
			fileInfo := utils.FileInfo {
				Name: result.FileName, 
				MetafileHash: result.MetafileHash,
			}
			filesRouting.addRoute(fileInfo, reply.Origin)
		}
	}
}

/*
	Returns the longest match 
*/
func longestMatch(fRoute FileRoutes ,keywords []string) string {
	prefix := "" 
	for _, kw := range keywords {
		if strings.HasPrefix(fRoute.FileInfo.Name, kw){
			prefix = kw
		}
	}
	return prefix
}

/*
	Returns the longest common subsequence between file names and keywords
*/ 
func lcms(fRoute FileRoutes ,keywords []string) string {
	return ""
}
func (filesRouting *FilesRouting) addRoute(fileInfo utils.FileInfo, route string){
	filesRouting.Lock()
	defer filesRouting.Unlock()

	if fileRoutes, found := filesRouting.filesRoutes[utils.HexToString(fileInfo.MetafileHash)]; found {
		fileRoutes.Routes = append(fileRoutes.Routes, route)
	}else {
		filesRouting.filesRoutes[utils.HexToString(fileInfo.MetafileHash)] = createFileRoute(fileInfo, []string{route})
	}
}

func (filesRouting *FilesRouting) GetAllRoutes() (routes []FileRoutes){
	filesRouting.Lock()
	defer filesRouting.Unlock()
	for _,v := range filesRouting.filesRoutes {
		routes = append(routes, *v)
	}
	return
}
