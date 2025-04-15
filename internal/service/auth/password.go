package auth

import "golang.org/x/crypto/bcrypt"

// PasswordVerifier defines the interface for comparing passwords.
type PasswordVerifier interface {
	// Compare compares a hashed password with its possible plaintext equivalent.
	// Returns nil on success, or an error on failure (e.g., mismatch).
	Compare(hashedPassword, password string) error
}

// BcryptVerifier implements PasswordVerifier using bcrypt.
type BcryptVerifier struct{}

// NewBcryptVerifier creates a new BcryptVerifier.
func NewBcryptVerifier() *BcryptVerifier {
	return &BcryptVerifier{}
}

// Compare implements the PasswordVerifier interface using bcrypt.
func (v *BcryptVerifier) Compare(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}
