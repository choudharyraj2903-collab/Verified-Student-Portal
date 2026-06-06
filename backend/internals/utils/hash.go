package utils 

import (
	"crypto/sha256"
	"encoding/hex"
	"crypto/subtle"
	"crypto/rand"
)

func Hash256String(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}


func Hash256Bytes(password []byte) string {
	hash := sha256.Sum256(password)
	return hex.EncodeToString(hash[:])
}

func ConstantTimeCompare(a,b string) bool {
	if subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 {
		return true
	}
	return false
}

func GenerateRandomString(size int) string {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

func StringSliceEqual(a, b []string) bool {
    if len(a) != len(b) {
        return false
    }
    for i := range a {
        if a[i] != b[i] {
            return false
        }
    }
    return true
}