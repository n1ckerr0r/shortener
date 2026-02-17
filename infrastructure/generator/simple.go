package generator

import (
	"crypto/rand"
	"math/big"

	"github.com/n1ckerr0r/shortener/core/domain/link"
)

type SimpleGenerator struct{}

const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func (SimpleGenerator) Generate() (link.ShortCode, error) {
	b := make([]byte, 6)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		b[i] = alphabet[n.Int64()]
	}
	return link.NewShortCode(string(b))
}
