package gossiper

import (
	"fmt"
	"github.com/simonwicky/Peerster/utils"
	"os"
	"math/big"
)


func recoverSecret(data [][]byte, pointX []int) []byte{
	var result_data []*big.Int
	points := make(map[int]*big.Int)

	for i := 0; i < len(data[0]) / utils.SIZE; i++ {
		for j := 0; j < len(pointX); j++{
			points[pointX[j]] = big.NewInt(0).SetBytes(data[j][utils.SIZE * i : utils.SIZE * (i + 1)])
		}
		//recover part i of secret using lagrange polynomial
		result_data = append(result_data, utils.ComputeLagrange(points))
	}
	plain_data := utils.CombineAndDecrypt(result_data)
	return plain_data
}

//==============================
//Split secret
//==============================
func splitSecret(data []byte, threshold int, n int) [][]byte {
	if threshold < 2 {
		fmt.Fprintln(os.Stderr,"Threshold must be greater than 2")
		return nil
	}
	if threshold > n {
		fmt.Fprintln(os.Stderr,"N must be at least equal to threshold")
		return nil
	}

	data_split := utils.EncryptAndSplit(data)
	nb_blocks := len(data_split)
	//generating polynomials

	//array of polynomial coefficient, one for each secret parts
	polynomials := make([][]*big.Int, nb_blocks)
	for i := 0; i < nb_blocks; i++ {
		polynomials[i] = make([]*big.Int, threshold)
		//setting the 0 value to the secret
		polynomials[i][0] = data_split[i]

		for j := 1; j < threshold; j++ {
			polynomials[i][j] = utils.Random()
		}
	}

	var result [][]byte
	//computing secret shares

	for i := 1; i <= n ; i++ {
		tmp := make([]byte,0)
		for j := 0; j < len(data_split) ; j++ {
			poly := polynomials[j]
			y_bytes := utils.EvaluatePolynomial(poly,i).Bytes()

			y_bytes = utils.PadBigInt(y_bytes)

			tmp = append(tmp,y_bytes...)
		}
		result = append(result,tmp)
	}


	return result
}