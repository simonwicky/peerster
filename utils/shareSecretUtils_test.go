//Authored by Simon Wicky
package utils

import ("testing"
		crand "crypto/rand"
		"math/rand"
		"math/big"
		"math"
		"fmt"
		"time")

func TestEncryptDecrypt(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	for i := 0; i < 1000; i++ {
		data := make([]byte, rand.Intn(5000))
		if len(data) == 0 {
			continue
		}
		_,_ = crand.Read(data)
		cipherText,_ := EncryptAndSplit(data)
		plaintext,_ := CombineAndDecrypt(cipherText)
		if len(plaintext) != len(data) {
			t.Errorf("Wrong output length %d, expected %d\n", len(plaintext), len(data))
		}
		for i := 0; i < len(data); i++{
			if data[i] != plaintext[i] {
				t.Error("Error encrypting and decrypting\n")
				break
			}
		}

	}
}

func TestEvaluatePolynomial(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	for iters := 0; iters < 1000 ; iters++ {
		length := rand.Intn(10)
		for length == 0 {
			length = rand.Intn(10)
		}
		polynomial_big := make([]*big.Int, length)
		polynomial := make([]int, length)

		for i := 0; i < len(polynomial); i++ {
			coeff := rand.Intn(100)
			polynomial_big[i] = big.NewInt(int64(coeff))
			polynomial[i] = coeff
		}

		x := rand.Intn(80)
		result := EvaluatePolynomial(polynomial_big, x)
		expected := 0
		for i := 0; i < len(polynomial); i++ {
			expected += polynomial[i] * int(math.Pow(float64(x),float64(i)))
		}
		if big.NewInt(int64(expected)).Cmp(result) != 0 {
			t.Errorf("Exected %s, got %s\n", big.NewInt(int64(expected)).String(),result.String())
			fmt.Println(polynomial)
			fmt.Println(x)
		}
	}
}

func TestModInverse(t *testing.T) {
	for iters := 0; iters < 1000 ; iters++ {
		prime,_ := big.NewInt(0).SetString(PRIME,10)
		n := Random()
		inv := ModInverse(n)
		n.Mul(n,inv)
		n.Mod(n,prime)
		if n.Cmp(big.NewInt(int64(1))) != 0 {
			t.Errorf("Wrong inverse")
		}
	}
}

func TestComputeLagrange(t *testing.T) {
	rand.Seed(time.Now().UTC().UnixNano())
	for iters := 0; iters < 1 ; iters++ {
		length := rand.Intn(10)
		for length == 0 {
			length = rand.Intn(10)
		}
		polynomial_big := make([]*big.Int, length)

		for i := 0; i < len(polynomial_big); i++ {
			coeff := rand.Intn(100)
			polynomial_big[i] = big.NewInt(int64(coeff))
		}
		points := make(map[int]*big.Int)
		for i := 1; i <= length; i++ {
			result := EvaluatePolynomial(polynomial_big, i)
			points[i] = result
		}
		expected := polynomial_big[0]
		result := ComputeLagrange(points)
		if expected.Cmp(result) != 0 {
			t.Errorf("Exected %s, got %s\n", expected.String(),result.String())
		}

	}
}