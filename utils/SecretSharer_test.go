//Authored by Simon Wicky
package utils

import (
	crand "crypto/rand"
	"math/rand"
	"testing"
	"time"
)

func TestSplitRecoverSecretRandom(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	for i := 0; i < 1000; i++ {
		//data set up
		data := make([]byte, rand.Intn(5000))
		if len(data) == 0 {
			continue
		}
		_, _ = crand.Read(data)

		array := splitSecret(data, 4, 4)
		plaintext := recoverSecret(array, []int{1, 2, 3, 4})

		//check validity
		if len(plaintext) != len(data) {
			t.Errorf("Wrong output length %d, expected %d\n", len(plaintext), len(data))
		}
		for i := 0; i < len(data); i++ {
			if data[i] != plaintext[i] {
				t.Error("Error splitting and recovering data\n")
				break
			}
		}

	}
}
