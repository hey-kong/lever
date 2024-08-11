package util

import (
	"crypto/rand"
	"fmt"
)

func GetKey(n int) []byte {
	return []byte("key_" + fmt.Sprintf("%07d", n))
}

// GetFixedLengthKey returns a 64-byte length key.
func GetFixedLengthKey(n int) string {
	key := fmt.Sprintf("key_%07d", n)
	if len(key) > 64 {
		key = key[:64]
	} else {
		key += string(make([]byte, 64-len(key)))
	}
	return key
}

// GetFixedLengthValue returns a 64-byte length value.
func GetFixedLengthValue(n int) []byte {
	key := fmt.Sprintf("key_%07d", n)
	if len(key) > 64 {
		key = key[:64]
	} else {
		key += string(make([]byte, 64-len(key)))
	}
	return []byte(key)
}

func GetValue(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}
	return b, nil
}
