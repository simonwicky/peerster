package gossiper

import (
	"fmt"
	"github.com/simonwicky/Peerster/utils"
	"os"
	_ "crypto/aes"
)

type SecretSharer struct {
	cloves map[int] []*utils.Cloves
}


func NewSecretSharer() *SecretSharer {
	return &SecretSharer{
		cloves: make(map[int][]*utils.Cloves),
	}
}

func (s *SecretSharer) checkSecret(id int) {
	array := s.cloves[id]
	k := array[0].K
	for _, c := range array {
		if c.K != k {
			fmt.Fprintln(os.Stderr,"Different K for same Id, dropping secret")
			s.cloves[id] = nil
			return
		}
	}
	n := len(array)
	if n < k {
		fmt.Fprintln(os.Stderr,"Not enough clove to decrypt secret yet")
		return
	}
	s.recoverSecret(array)
}

func (s *SecretSharer) recoverSecret(array []*utils.Cloves) {
	//recover secret and handle it
}

//==============================
//Sending secret
//==============================
func (g *Gossiper) splitAndSendSecret(data []byte, threshold int, n int) bool {
	if threshold < 2 {
		fmt.Fprintln(os.Stderr,"Threshold must be greater than 2")
		return false
	}
	if threshold > n {
		fmt.Fprintln(os.Stderr,"N must be at least equl to threshold")
		return false
	}
	return true
}