package utils

import (
	"fmt"
	"testing"
)

func TestSimpleSplitRecover(t *testing.T) {
	cloves := NewProxyInit().Split(2, 3)
	for _, clove := range cloves {
		fmt.Println(clove)
	}
	df := NewDataFragment(cloves[:cloves[0].Threshold])
	if df == nil || df.Proxy == nil {
		t.Error("error")
	}
}
