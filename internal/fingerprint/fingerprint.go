package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
)

func Event(message, stacktrace string) string {
	data := message + ":" + stacktrace
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
