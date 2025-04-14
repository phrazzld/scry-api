package domain

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Common validation errors
var (
	ErrEmptyUserID         = errors.New("user ID cannot be empty")
	ErrInvalidEmail        = errors.New("invalid email format")
	ErrEmptyEmail          = errors.New("email cannot be empty")
	ErrPasswordTooShort    = errors.New("password must be at least 8 characters long")
	ErrEmptyPassword       = errors.New("password cannot be empty")
	ErrEmptyHashedPassword = errors.New("hashed password cannot be empty")
)

// User represents a registered user of the Scry application.
// It contains essential user information and authentication details.
type User struct {
	ID             uuid.UUID `json:"id"`
	Email          string    `json:"email"`
	HashedPassword string    `json:"-"` // Never expose password hash in JSON
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// NewUser creates a new User with the given email and hashed password.
// It generates a new UUID for the user ID and sets the creation/update timestamps.
// Returns an error if validation fails.
func NewUser(email, hashedPassword string) (*User, error) {
	user := &User{
		ID:             uuid.New(),
		Email:          email,
		HashedPassword: hashedPassword,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := user.Validate(); err != nil {
		return nil, err
	}

	return user, nil
}

// Validate checks if the User has valid data.
// Returns an error if any field fails validation.
func (u *User) Validate() error {
	if u.ID == uuid.Nil {
		return ErrEmptyUserID
	}

	if u.Email == "" {
		return ErrEmptyEmail
	}

	// Basic email format validation
	// In a real application, consider using a more robust email validation library
	if !validateEmailFormat(u.Email) {
		return ErrInvalidEmail
	}

	if u.HashedPassword == "" {
		return ErrEmptyHashedPassword
	}

	return nil
}

// validateEmailFormat performs basic validation of email format.
// Returns true if the email appears to be in a valid format.
func validateEmailFormat(email string) bool {
	// Simple check for demonstration - should have @ and at least one . after @
	// In production, consider using a proper email validation library
	atIndex := -1
	for i, char := range email {
		if char == '@' {
			atIndex = i
			break
		}
	}

	if atIndex == -1 || atIndex == 0 || atIndex == len(email)-1 {
		return false
	}

	// Check for domain part after @
	domainPart := email[atIndex+1:]
	if len(domainPart) < 3 { // minimum would be "a.b"
		return false
	}

	// Check for dot in domain, but not immediately after @ and not at the end
	dotIndex := -1
	for i, char := range domainPart {
		if char == '.' {
			dotIndex = i
			break
		}
	}

	if dotIndex == -1 || dotIndex == 0 || dotIndex == len(domainPart)-1 {
		return false
	}

	return true
}
