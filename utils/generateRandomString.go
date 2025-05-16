package utils

import (
	"crypto/rand"
	"math/big"
)

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func GenerateRandomString(length int) string {
	result := make([]byte, length)
	charsetLength := big.NewInt(int64(len(charset)))

	for i := range result {
		num, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			panic("failed to generate secure random number")
		}
		result[i] = charset[num.Int64()]
	}

	return string(result)
}
