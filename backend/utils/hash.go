package utils

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
)

func Hash256(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func Hash256Bytes(password []byte) string {
	hash := sha256.Sum256(password)
	return hex.EncodeToString(hash[:])
}

func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
