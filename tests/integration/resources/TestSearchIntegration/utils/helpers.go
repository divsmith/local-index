package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random string: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

func LogError(message string, err error) {
	fmt.Printf("ERROR: %s: %v\n", message, err)
}
