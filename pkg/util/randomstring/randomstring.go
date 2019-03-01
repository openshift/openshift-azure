package randomstring

import (
	"crypto/rand"
	"math/big"
)

// RandomString returns a randomized string from a set of characters of the given length
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

// RandomStorageAccountName returns a valid randomized storage account name
func RandomStorageAccountName() (string, error) {
	return RandomString("abcdefghijklmnopqrstuvwxyz0123456789", 24)
}

// RandomASCIIString returns a random string of the given length from the basic ASCII char map
func RandomASCIIString(length int) (string, error) {
	return RandomString("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", length)
}
