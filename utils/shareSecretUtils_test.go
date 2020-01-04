package utils

import ("testing"
		"crypto/rand"
		"fmt")

func TestEncryptDecrypt(t *testing.T) {
	for i := 0; i < 1000; i++ {
		data := make([]byte, 1280)
		_,_ = rand.Read(data)
		plaintext := CombineAndDecrypt(EncryptAndSplit(data))
		if len(plaintext) != len(data) {
			t.Errorf("Wrong output length %d, expected %d\n", len(plaintext), len(data))
		}
		fmt.Println(len(data))
		fmt.Println(len(plaintext))
		for i := 0; i < len(data); i++{
			if data[i] != plaintext[i] {
				t.Error("Error encrypting and decrypting\n")
				break
			}
		}

	}
}