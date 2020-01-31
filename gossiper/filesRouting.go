//author: Boubacar Camara
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
	ProxyOwnerPaths []string
	ClovesPath []string
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

func (filesRouting *FilesRouting) addOwnerPath(fileInfo utils.FileInfo,  paths[]string){

	if fileRoutes, found := filesRouting.filesRoutes[ fileInfo.Name]; found {
		if len(fileRoutes.ProxyOwnerPaths)==0{
			fileRoutes.ProxyOwnerPaths = make([]string, len(paths))	
			copy(fileRoutes.ProxyOwnerPaths, paths)
		}
	}else {
		froute := &FileRoutes{
			FileInfo:fileInfo, 
			ProxyOwnerPaths: make([]string, len(paths)),
		}
		copy(froute.ProxyOwnerPaths, paths)	
	}
}


func (filesRouting *FilesRouting) GetRoutes(fname string) []string{
	filesRouting.Lock()
	defer filesRouting.Unlock()
	var routes []string
	if fileRoutes, found := filesRouting.filesRoutes[fname]; found{
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
	copy(filescpy, filesRoutes)

	sort.Slice(filescpy, func(i, j int) bool {
		return len(longestMatch(filescpy[i], keywords)) > len(longestMatch(filescpy[j], keywords))
	})
	return filescpy
}

//Updates the routing table according to the search reply
func (filesRouting *FilesRouting) UpdateRouting(reply utils.GCSearchReply){
	//fmt.Println("print before Update files routing table")
	utils.LogGCSearchReply(&reply)

	for _, result := range reply.AccessibleFiles {
		//if len(result.ChunkMap) == int(result.ChunkCount){
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
	longestkw := "" 
	for _, kw := range keywords {
		if strings.Contains(fRoute.FileInfo.Name, kw) && len(kw) > len(longestkw){
			longestkw = kw
		}
	}
	return longestkw
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

	if fileRoutes, found := filesRouting.filesRoutes[ fileInfo.Name]; found {
		for _, r := range fileRoutes.Routes {
			if r == route {
				return 
			}
		}
		fileRoutes.Routes = append(fileRoutes.Routes, route)
	}else {
		filesRouting.filesRoutes[ fileInfo.Name] = createFileRoute(fileInfo, []string{route})
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
	for _,v := range filesRouting.filesRoutes {
		i += 1
		fmt.Printf("%d) %s %v %s\n",i, v.FileInfo.Name, v.Routes, hex.EncodeToString(v.FileInfo.MetafileHash)[:5])

	}
	return
}