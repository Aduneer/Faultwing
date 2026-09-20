package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"
)

func Event(exceptionType, stacktrace string) string {
	data := "v1:" + strconv.Itoa(len(exceptionType)) + ":" + exceptionType + stacktrace
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}
