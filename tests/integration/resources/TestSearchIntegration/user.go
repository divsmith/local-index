//go:build testdata

package main

import (
	"fmt"
	"regexp"
	"strings"
)

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func ValidateUser(user *User) error {
	if user.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if !isValidEmail(user.Email) {
		return fmt.Errorf("invalid email format")
	}

	return nil
}

func isValidEmail(email string) bool {
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

func (u *User) GetDisplayName() string {
	return strings.Title(strings.ToLower(u.Name))
}
