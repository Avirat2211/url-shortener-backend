package shortener

import (
	"crypto/sha256"
	"fmt"
	"math/big"
	"os"

	"github.com/itchyny/base58-go"
)

func sha256Hash(input string) []byte {
	algo := sha256.New()
	algo.Write([]byte(input))
	return algo.Sum(nil)
}

func base58Encoded(bytes []byte) string {
	encoding := base58.BitcoinEncoding
	encoded, err := encoding.Encode(bytes)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	return string(encoded)
}

func GenerateShortenUrl(initalUrl string, userId string) string {
	hashed := sha256Hash(initalUrl + userId)
	num := new(big.Int).SetBytes(hashed).Uint64()
	ans := base58Encoded([]byte(fmt.Sprintf("%d", num)))
	return ans[:8]
}
