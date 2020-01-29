package gossiper

import (
	"sync"
	"github.com/simonwicky/Peerster/utils"
	"encoding/hex"
	"sort"
	"strings"
	"fmt"
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

func NewFilesRouting() *FilesRouting {
	return &FilesRouting{filesRoutes: make(map[string]*FileRoutes)}
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
	fmt.Println("Before sort")
	filesRouting.dump()
	filesRoutes := filesRouting.GetAllRoutes()
	filescpy := make([]FileRoutes, len(filesRoutes))
	copy(filescpy, filesRoutes)

	sort.Slice(filescpy, func(i, j int) bool {
		return len(longestMatch(filescpy[i], keywords)) > len(longestMatch(filescpy[j], keywords))
	})
	filesRouting.dump()
	return filescpy
}

//Updates the routing table according to the search reply
func (filesRouting *FilesRouting) UpdateRouting(reply utils.GCSearchReply){
	fmt.Println("print before Update files routing table")
	utils.LogGCSearchReply(&reply)

	for _, result := range reply.AccessibleFiles {
		//if len(result.ChunkMap) == int(result.ChunkCount){
			fmt.Println( hex.EncodeToString(result.MetafileHash))
			fileInfo := utils.FileInfo {
				Name: result.FileName, 
				MetafileHash: make([]byte, len(result.MetafileHash)),
			}
			copy(fileInfo.MetafileHash, result.MetafileHash)
			filesRouting.addRoute(fileInfo, reply.Origin)
		//}
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
	fmt.Println( hex.EncodeToString(fileInfo.MetafileHash))

	if fileRoutes, found := filesRouting.filesRoutes[ hex.EncodeToString(fileInfo.MetafileHash)]; found {
		for _, r := range fileRoutes.Routes {
			if r == route {
				return 
			}
		}
		fileRoutes.Routes = append(fileRoutes.Routes, route)
	}else {
		fmt.Println("wrong",  hex.EncodeToString(fileInfo.MetafileHash))

		filesRouting.filesRoutes[ hex.EncodeToString(fileInfo.MetafileHash)] = createFileRoute(fileInfo, []string{route})
	}
	//fmt.Printf("Added entry %v with route %s", fileInfo, route )

}

func (filesRouting *FilesRouting) GetAllRoutes() (routes []FileRoutes){
	filesRouting.Lock()
	defer filesRouting.Unlock()
	for _,v := range filesRouting.filesRoutes {
		routes = append(routes, *v)
	}
	return
}


func (filesRouting *FilesRouting) asSearchResults() []*utils.SearchResult{
	filesRouting.Lock()
	defer filesRouting.Unlock()
	var results []*utils.SearchResult
	for _,fileRoute := range filesRouting.filesRoutes{
		result := &utils.SearchResult{
			FileName: fileRoute.FileInfo.Name,
			MetafileHash: make([]byte, len(fileRoute.FileInfo.MetafileHash)),
			ChunkCount: uint64(fileRoute.FileInfo.Size / int64(2 << 12)),
		}
		copy(result.MetafileHash, fileRoute.FileInfo.MetafileHash)


		results = append(results,result)
	}
	return results
}

func (filesRouting *FilesRouting) dump(){
	filesRouting.Lock()
	defer filesRouting.Unlock()
	fmt.Println("Dump files routing table:")
	i := 0
	for k,v := range filesRouting.filesRoutes {
		i += 1
		fmt.Printf("%d) %s %v %s %s\n",i, v.FileInfo.Name, v.Routes,  k, hex.EncodeToString(v.FileInfo.MetafileHash))
	}
	return
}