package utils

import (
	"fmt"
	"math/big"
	"os"
)



func recoverSecret(data [][]byte, pointX []int) ([]byte, error) {
	/*if len(data) != len(pointX) {
		return nil, errors.New("data and pointX do not match")
	}*/
	var result_data []*big.Int
	points := make(map[int]*big.Int)

	for i := 0; i < len(data[0])/SIZE; i++ {
		for j := 0; j < len(pointX); j++ {
			/*if len(data[j]) != len(data[0]) {
				return nil, errors.New("corrupted data")
			}*/
			points[pointX[j]] = big.NewInt(0).SetBytes(data[j][SIZE*i : SIZE*(i+1)])
		}
		//recover part i of secret using lagrange polynomial
		result_data = append(result_data, ComputeLagrange(points))
	}
	return CombineAndDecrypt(result_data)

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
