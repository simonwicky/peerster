package utils

import ("crypto/cipher"
		"math/big"
		"crypto/rand"
		"crypto/aes"
		"errors"
)

const (
	AES_IV = "fixedIVtouse!!!!"
	SIZE = 16
	//used to generate a field and keep the blocks at 128 bits
	PRIME = "340282366920938463463374607431768211297"
)


//taken from https://play.golang.org/p/ssET5KZQuj
func Random() *big.Int {
	max,_ := big.NewInt(0).SetString(PRIME,10)
	max.Sub(max, big.NewInt(1))
	n, _ := rand.Int(rand.Reader, max)
	return n
}

func PadBigInt(n []byte) []byte {
	if len(n) % SIZE != 0 {
			padding := make([]byte,SIZE - (len(n) % SIZE))
			n = append(padding,n...)
	}
	return n
}


func EncryptAndSplit(data_raw []byte) ([]*big.Int, error) {
		//pad
		pad_size := SIZE - (len(data_raw) % SIZE)
		padding := make([]byte,pad_size)
		for i := 0; i < pad_size; i++ {
			padding[i] = byte(pad_size)
		}
		data := append(padding,data_raw...)

		//gen key
	    key_bytes := make([]byte, SIZE)
    	_, err := rand.Read(key_bytes)
    	if err != nil {
    		return nil, errors.New("Error generating random bytes")
    	}

    	//encrypt
    	c,_ := aes.NewCipher(key_bytes)
    	cbc := cipher.NewCBCEncrypter(c,[]byte(AES_IV))
    	cipherText := make([]byte, len(data))
    	cbc.CryptBlocks(cipherText, data)

    	prime,_ :=big.NewInt(0).SetString(PRIME,10)

    	//split to big.Int
    	cipherData := make([]*big.Int,len(data) / SIZE)
    	for i := 0; i < len(data) / SIZE ; i++ {
    		cipherData[i] = big.NewInt(0).SetBytes(cipherText[SIZE * i : SIZE * (i + 1)])
    		if cipherData[i].Cmp(prime) == 1 {
    			//data is greater than PRIME, will fail for decryption. Restart with another key. Very very very unlikely to happen (2^128-PRIME / 2^128)
    			return EncryptAndSplit(data_raw)
    		}
    	}

    	//combine data and key
    	key := big.NewInt(0).SetBytes(key_bytes)
    	cipherData = append(cipherData,key)
    	return cipherData, nil

}

func CombineAndDecrypt(cipherDataKey []*big.Int) ([]byte, error) {
	//split data and key
	key := cipherDataKey[len(cipherDataKey)-1]
	cipherData := cipherDataKey[:len(cipherDataKey)-1]

	//combine to bytes
	cipherText := make([]byte, 0)
	for i := 0; i < len(cipherData) ; i++ {
		int_bytes := cipherData[i].Bytes()
		int_bytes = PadBigInt(int_bytes)
		cipherText = append(cipherText, int_bytes...)
	}

	//decrypt
	key_bytes := key.Bytes()
	key_bytes = PadBigInt(key_bytes)
	c,_ := aes.NewCipher(key_bytes)
	cbc := cipher.NewCBCDecrypter(c,[]byte(AES_IV))
	data := make([]byte, len(cipherText))
	cbc.CryptBlocks(data, cipherText)

	//remove padding
	pad_size := int(data[0])
	if pad_size > SIZE {
		return nil, errors.New("Invalid padding, error during decryption")
	}
	data = data[pad_size:]

	return data, nil
}

//polynomial [a,b,c,d] is a + bx + cx^2 + dx^3
func EvaluatePolynomial(polynomial []*big.Int, x int) (y *big.Int) {
	prime,_ :=big.NewInt(0).SetString(PRIME,10)
	x_big := big.NewInt(int64(x))
	y = big.NewInt(0)
	for i := len(polynomial) - 1; i > 0; i-- {
		y.Add(y,polynomial[i])
		y.Mul(y,x_big)
		//modulo prime of 128 bits, to keep it 128 bits
		y.Mod(y,prime)
	}
	y.Add(y,polynomial[0])
	y.Mod(y,prime)
	return
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
		numerator := big.NewInt(1)
		denominator := big.NewInt(1)
		for j := range points {
			if i != j {
				xj := big.NewInt(int64(j))
				xi := big.NewInt(int64(i))

				//numerator * -xj
				numerator.Mul(numerator,tmp.Mul(xj,big.NewInt(int64(-1))))
				numerator.Mod(numerator,prime)

				//denominator * (xi-xj)
				denominator.Mul(denominator,tmp.Sub(xi,xj))
				denominator.Mod(denominator,prime)

			}
		}
		//fraction
		p.Mul(numerator,ModInverse(denominator))

		//yp += numerator/denominator * yi
		yp.Add(yp, p.Mul(p,points[i]))
		yp.Mod(yp,prime)
	}
	return yp
}

