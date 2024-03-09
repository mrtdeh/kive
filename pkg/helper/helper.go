package helper

import (
	"crypto/sha256"
	"encoding/hex"
)

func Hash(str string) string {
	id := sha256.Sum256([]byte(str))
	return hex.EncodeToString(id[:])[:6]
}

func Contains(str string, arrStr []string) bool {
	for _, v := range arrStr {
		if v == str {
			return true
		}
	}
	return false
}
