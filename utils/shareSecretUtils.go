package utils

import ("crypto/cipher"
		"math/big"
		"crypto/rand"
		"crypto/aes"
		_ "fmt"
)

const (
	AES_IV = "fixedIVtouse!!!!"
	SIZE = 16
	//generated by rand.Prime(rand.Reader,128), used to generate a field and keep the blocks at 128 bits
	PRIME = "323278307053475795581534268335355197223"
)


//taken from https://play.golang.org/p/ssET5KZQuj
func Random() *big.Int {
	max,_ := big.NewInt(0).SetString(PRIME,10)
	max.Sub(max, big.NewInt(1))
	n, _ := rand.Int(rand.Reader, max)
	return n
}


func EncryptAndSplit(data []byte) (cipherData []*big.Int) {
		//pad
		pad_size := SIZE - (len(data) % SIZE)
		padding := make([]byte,pad_size)
		for i := 0; i < pad_size; i++ {
			padding[i] = byte(pad_size)
		}
		data = append(padding,data...)

		//gen key
	    key_bytes := make([]byte, SIZE)
    	_, err := rand.Read(key_bytes)
    	if err != nil {
    		panic("Error generating random bytes")
    	}

    	//encrypt
    	c,_ := aes.NewCipher(key_bytes)
    	cbc := cipher.NewCBCEncrypter(c,[]byte(AES_IV))
    	cipherText := make([]byte, len(data))
    	cbc.CryptBlocks(cipherText, data)

    	//split to big.Int
    	cipherData = make([]*big.Int,len(data) / SIZE)
    	for i := 0; i < len(data) / SIZE ; i++ {
    		cipherData[i] = big.NewInt(0).SetBytes(cipherText[SIZE * i : SIZE * (i + 1)])
    	}

    	//combine data and key
    	key := big.NewInt(0).SetBytes(key_bytes)
    	cipherData = append(cipherData,key)
    	return

}

func CombineAndDecrypt(cipherData []*big.Int) (data []byte) {
	//split data and key
	key := cipherData[len(cipherData)-1]
	cipherData = cipherData[:len(cipherData)-1]

	//combine to bytes
	cipherText := make([]byte, 0)
	for i := 0; i < len(cipherData) ; i++ {
		int_bytes := cipherData[i].Bytes()
		if len(int_bytes) % SIZE != 0 {
			padding := make([]byte,SIZE - (len(int_bytes) % SIZE))
			int_bytes = append(padding,int_bytes...)
		}
		cipherText = append(cipherText, int_bytes...)
	}

	//decrypt
	key_bytes := key.Bytes()
	if len(key_bytes) != SIZE {
			padding := make([]byte,SIZE - (len(key_bytes) % SIZE))
			key_bytes = append(padding,key_bytes...)
	}
	c,_ := aes.NewCipher(key_bytes)
	cbc := cipher.NewCBCDecrypter(c,[]byte(AES_IV))
	data = make([]byte, len(cipherText))
	cbc.CryptBlocks(data, cipherText)

	//remove padding
	pad_size := int(data[0])
	data = data[pad_size:]

	return
}

//polynomial [a,b,c,d] is a + bx + cx^2 + dx^3
func EvaluatePolynomial(polynomial []*big.Int, x int) (*big.Int) {
	prime,_ :=big.NewInt(0).SetString(PRIME,10)
	x_big := big.NewInt(int64(x))
	y := big.NewInt(0)
	for i := len(polynomial) - 1; i > 0; i-- {
		y = y.Add(y,polynomial[i])
		y = y.Mul(y,x_big)
		//modulo prime of 128 bits, to keep it 128 bits
		y = y.Mod(y,prime)
	}
	y = y.Add(y,polynomial[0])
	y = y.Mod(y,prime)
	return y
}

func ModInverse(n *big.Int) *big.Int {
	prime,_ := big.NewInt(0).SetString(PRIME,10)
	ncopy := big.NewInt(0).Set(n)
	ncopy = ncopy.Mod(ncopy,prime)
	x := big.NewInt(0)
	y := big.NewInt(0)
	ncopy.GCD(x,y,prime,ncopy)
	y = y.Mod(y,prime)
	return y
}

func ComputeLagrange(points map[int]*big.Int) *big.Int {
	prime,_ := big.NewInt(0).SetString(PRIME,10)
	yp := big.NewInt(0)
	tmp := big.NewInt(0)
	for i := range points {
		p := big.NewInt(1)
		for j := range points {
			if i != j {
				xj := big.NewInt(int64(j))
				xi := big.NewInt(int64(i))
				numerator := tmp.Mul(xj,big.NewInt(int64(-1)))
				numerator = numerator.Mod(numerator,prime)
				denominator := tmp.Sub(xi,xj)
				denominator = denominator.Mod(denominator,prime)
				res := tmp.Mul(numerator,ModInverse(denominator))
				p = p.Mul(p,res)
				p = p.Mod(p,prime)
			}
		}
		yp = yp.Add(yp, p.Mul(p,points[i]))
		yp = yp.Mod(yp,prime)
	}
	return yp
}

