package util

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateSecureRandomString(length int) (string, error) {
	// Calculate the number of bytes needed for the desired string length
	// Base64 encoding expands the data, so we need fewer raw bytes.
	// A common ratio is 3 bytes to 4 base64 characters.
	// We'll use a slightly more generous calculation to ensure enough data.
	numBytes := (length * 3) / 4

	b := make([]byte, numBytes)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:length], nil
}
