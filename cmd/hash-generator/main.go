package main

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	passwords := []string{
		"testpassword123",
		"test@#$%^&*()",
		"this-is-a-very-long-password-that-tests-edge-cases-for-bcrypt-hashing-algorithm",
		"тест123",
	}

	for _, password := range passwords {
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			fmt.Printf("Error generating hash for %s: %v\n", password, err)
			continue
		}
		fmt.Printf("Password: %s\nHash: %s\n\n", password, string(hash))
	}
}
