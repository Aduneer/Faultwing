package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

const (
	keyPrefix        = "faultwing_"
	displayPrefixLen = len(keyPrefix) + 8
)

var ErrInvalid = errors.New("invalid API key")

func Generate() (string, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}

	return keyPrefix + base64.RawURLEncoding.EncodeToString(random), nil
}

func Hash(key string) [sha256.Size]byte {
	return sha256.Sum256([]byte(key))
}

func DisplayPrefix(key string) string {
	if len(key) <= displayPrefixLen {
		return key
	}
	return key[:displayPrefixLen]
}
