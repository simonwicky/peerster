package utils

import (
	"errors"
	"fmt"
	"math/big"
	"os"
)

// type SecretSharer struct {
// 	cloves map[*big.Int] []*utils.Cloves
// }

// func NewSecretSharer() *SecretSharer {
// 	return &SecretSharer{
// 		cloves: make(map[*big.Int][]*utils.Cloves),
// 	}
// }

// func (s *SecretSharer) checkSecret(id *big.Int) {
// 	array := s.cloves[id]
// 	k := array[0].K
// 	data_len := len(array[0].Data)
// 	sequence_numbers := make(map[int]struct{})
// 	for _, c := range array {
// 		//check k
// 		if c.K != k {
// 			fmt.Fprintln(os.Stderr,"Different K for same Id, dropping secret")
// 			s.cloves[id] = nil
// 			return
// 		}
// 		//check data_len
// 		if len(c.Data) != data_len {
// 			fmt.Fprintln(os.Stderr,"Different len for same Id, dropping secret")
// 			s.cloves[id] = nil
// 			return
// 		}

// 		sequence_numbers[c.Sequence_number] = struct{}{}
// 	}

// 	n := len(array)
// 	if n < k {
// 		fmt.Fprintln(os.Stderr,"Not enough clove to decrypt secret yet")
// 		return
// 	}

// 	//sequence_numbers check
// 	var sequence_numbers_array []int
// 	for k := range sequence_numbers {
// 		sequence_numbers_array = append(sequence_numbers_array, k)
// 	}
// 	if len(sequence_numbers_array) < k {
// 		fmt.Fprintln(os.Stderr,"Not enough sequence number to decrypt secret yet")
// 		return
// 	}
// 	s.recoverSecret(array)
// }

func recoverSecret(data [][]byte, pointX []int) ([]byte, error) {
	if len(data) != len(pointX) {
		return nil, errors.New("data and pointX do not match")
	}
	var result_data []*big.Int
	points := make(map[int]*big.Int)

	for i := 0; i < len(data[0])/SIZE; i++ {
		for j := 0; j < len(pointX); j++ {
			if len(data[j]) != len(data[0]) {
				return nil, errors.New("corrupted data")
			}
			points[pointX[j]] = big.NewInt(0).SetBytes(data[j][SIZE*i : SIZE*(i+1)])
		}
		//recover part i of secret using lagrange polynomial
		result_data = append(result_data, ComputeLagrange(points))
	}
	return CombineAndDecrypt(result_data)

	//direct data to proxy/content/whatever
}

//==============================
//Split secret
//==============================
func splitSecret(data []byte, threshold int, n int) ([][]byte, error) {
	if threshold < 2 {
		fmt.Fprintln(os.Stderr, "Threshold must be greater than 2")
		return nil, nil
	}
	if threshold > n {
		fmt.Fprintln(os.Stderr, "N must be at least equal to threshold")
		return nil, nil
	}

	data_split, err := EncryptAndSplit(data)
	if err != nil {
		return [][]byte{}, err
	}
	nb_blocks := len(data_split)
	//generating polynomials

	//array of polynomial coefficient, one for each secret parts
	polynomials := make([][]*big.Int, nb_blocks)
	for i := 0; i < nb_blocks; i++ {
		polynomials[i] = make([]*big.Int, threshold)
		//setting the 0 value to the secret
		polynomials[i][0] = data_split[i]

		for j := 1; j < threshold; j++ {
			polynomials[i][j] = Random()
		}
	}

	var result [][]byte
	//computing secret shares

	for i := 1; i <= n; i++ {
		tmp := make([]byte, 0)
		for j := 0; j < len(data_split); j++ {
			poly := polynomials[j]
			y_bytes := EvaluatePolynomial(poly, i).Bytes()

			y_bytes = PadBigInt(y_bytes)

			tmp = append(tmp, y_bytes...)
		}
		result = append(result, tmp)
	}

	return result, nil
}
