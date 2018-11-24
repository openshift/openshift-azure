package randomstring

import (
	"crypto/rand"
	"math/big"
)

func RandomString(letterBytes string, length int) (string, error) {
	b := make([]byte, length)
	for i := range b {
		o, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			return "", err
		}
		b[i] = letterBytes[o.Int64()]
	}

	return string(b), nil
}
