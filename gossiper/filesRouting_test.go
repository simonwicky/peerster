/*
Author: Boubacar Camara

Run with cmd: go test gossiper/filesRouting.go gossiper/filesRouting_test.go
*/
package gossiper

import (
	"testing"
	"github.com/simonwicky/Peerster/utils"
	"time"
	"fmt"
	"math/rand"
	"crypto/sha256"

)

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		return
	}

	t.Fatal(fmt.Sprintf("%v != %v", a, b))
}




func makeRange(max uint64) []uint64 {
    a := make([]uint64, max)
    for i := range a {
        a[i] = uint64(i)
    }
    return a
}

func createGCSearchReply(origin string, fnames []string)  *utils.GCSearchReply{
	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	reply := &utils.GCSearchReply{Origin:origin}
	for _, fname := range fnames{
		nbChunks := uint64(r1.Intn(10))
		nameBytes := []byte(fname)
		hash := sha256.Sum256(nameBytes)
		reply.AccessibleFiles = append(reply.AccessibleFiles, &utils.SearchResult{
			FileName: fname, 
			MetafileHash:hash[:],
			ChunkMap: makeRange(nbChunks),
			ChunkCount: nbChunks,
		})
	}
	return reply
}

func checkArrays(t *testing.T, a1, b1 []string){
	if len(a1) != len(b1) {
		t.Errorf("Wrong arrays dimensions %d != %d for arrays %v and %v", len(a1), len(b1), a1, b1 )
	}
	for i, a := range a1 {
		if a != b1[i]{
			t.Errorf("%s != %s at index %d in arrays %v %v", a, b1[i], i, a1, b1)
		}
	}
}
func TestBasicTableUpdates(t *testing.T){
	table := NewFilesRouting()
	table.UpdateRouting(*createGCSearchReply("A", []string{"f1", "f2"}))
	table.UpdateRouting(*createGCSearchReply("B", []string{"f32", "f2"}))
	table.UpdateRouting(*createGCSearchReply("A", []string{"f3"}))
	//Add again to make sure there is no duplicated file
	table.UpdateRouting(*createGCSearchReply("A", []string{"f3"}))

	//t.Log(table.GetAllRoutes())

	t.Log(table.RoutesSorted([]string{"f1"})[0].Routes)
	assertEqual(t, len(table.GetAllRoutes()), 4)
	checkArrays(t, table.RoutesSorted([]string{"f1"})[0].Routes,  []string{"A"})
	checkArrays(t, table.RoutesSorted([]string{"f2"})[0].Routes,  []string{"A", "B"})
	checkArrays(t, table.RoutesSorted([]string{"f32"})[0].Routes, []string{"B"})
	checkArrays(t, table.RoutesSorted([]string{"f3", "f32"})[0].Routes, []string{"B"})
	checkArrays(t, table.RoutesSorted([]string{"f3"})[1].Routes, []string{"A"})


}