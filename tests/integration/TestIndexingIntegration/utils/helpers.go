package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

func GenerateID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func LogInfo(message string) {
	log.Printf("[INFO] %s", message)
}

func LogError(message string, err error) {
	log.Printf("[ERROR] %s: %v", message, err)
}

func SanitizeInput(input string) string {
	// Remove potentially dangerous characters
	sanitized := strings.ReplaceAll(input, "<", "&lt;")
	sanitized = strings.ReplaceAll(sanitized, ">", "&gt;")
	return strings.TrimSpace(sanitized)
}

func ValidateEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func FormatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%.0fms", float64(d.Nanoseconds())/1e6)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	return fmt.Sprintf("%.1fm", d.Minutes())
}
