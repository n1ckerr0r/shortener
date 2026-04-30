package generator

import (
	"crypto/rand"
	"math/big"

	"github.com/n1ckerr0r/shortener/internal/link"
)

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const codeLength = 6

type RandomGenerator struct{}

func (RandomGenerator) Generate() (link.ShortCode, error) {
	value := make([]byte, codeLength)
	for i := range value {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return link.ShortCode{}, err
		}
		value[i] = alphabet[n.Int64()]
	}

	return link.NewShortCode(string(value))
}
