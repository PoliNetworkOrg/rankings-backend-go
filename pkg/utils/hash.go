package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	saltGlobal  = "saltPoliNetwork"
	maxCharHash = 20
)

func HashWithSalt(input string) string {
	salted := input + saltGlobal
	hash := sha256.Sum256([]byte(salted))
	hexHash := hex.EncodeToString(hash[:])

	return strings.ToLower(hexHash[:maxCharHash])
}
