package userauth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionTokenPrefix = "fly_session_"
	sessionTokenBytes  = 32
	SessionDuration    = 7 * 24 * time.Hour
)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func PasswordMatches(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func GenerateSessionToken() (string, [sha256.Size]byte, error) {
	random := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(random); err != nil {
		return "", [sha256.Size]byte{}, err
	}

	token := sessionTokenPrefix + base64.RawURLEncoding.EncodeToString(random)
	return token, sha256.Sum256([]byte(token)), nil
}

func ParseSessionToken(token string) ([sha256.Size]byte, bool) {
	if !strings.HasPrefix(token, sessionTokenPrefix) {
		return [sha256.Size]byte{}, false
	}

	random, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(token, sessionTokenPrefix))
	if err != nil || len(random) != sessionTokenBytes {
		return [sha256.Size]byte{}, false
	}
	return sha256.Sum256([]byte(token)), true
}
