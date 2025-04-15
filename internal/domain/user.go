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
	ErrPasswordTooShort    = errors.New("password must be at least 12 characters long")
	ErrPasswordTooLong     = errors.New("password must be at most 72 characters long")
	ErrEmptyPassword       = errors.New("password cannot be empty")
	ErrEmptyHashedPassword = errors.New("hashed password cannot be empty")
)

// User represents a registered user of the Scry application.
// It contains essential user information and authentication details.
type User struct {
	ID             uuid.UUID `json:"id"`
	Email          string    `json:"email"`
	Password       string    `json:"-"` // Plaintext password, used temporarily during registration/updates
	HashedPassword string    `json:"-"` // Never expose password hash in JSON
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// NewUser creates a new User with the given email and password.
// It generates a new UUID for the user ID and sets the creation/update timestamps.
// Returns an error if validation fails.
//
// NOTE: This function only sets up the user structure with the plaintext password.
// The caller is responsible for hashing the password before storing the user.
func NewUser(email, password string) (*User, error) {
	user := &User{
		ID:        uuid.New(),
		Email:     email,
		Password:  password, // Plaintext password - must be hashed before storage
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
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

	// Password validation
	// During user creation/update we need to validate the provided password
	if u.Password != "" {
		// When plaintext password is provided, validate its length
		if !validatePasswordComplexity(u.Password) {
			if len(u.Password) < 12 {
				return ErrPasswordTooShort
			}
			return ErrPasswordTooLong
		}
	} else {
		// When no plaintext password is provided, the user must have a hashed password
		// (this would be the case for existing users in the database)
		if u.HashedPassword == "" {
			return ErrEmptyPassword
		}
	}

	return nil
}

// TODO: Replace this basic email validation with a more robust library.
// This implementation is intentionally simple and has limitations.
// Consider using a dedicated email validation library that follows
// RFC 5322 standards and handles edge cases properly.
//
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

// validatePasswordComplexity checks if a password meets length requirements:
// - Minimum length: 12 characters
// - Maximum length: 72 characters (bcrypt's practical limit)
//
// This simplified approach focuses on length rather than character complexity
// because longer passwords provide better security than shorter ones with
// special character requirements, which can be harder for users to remember.
func validatePasswordComplexity(password string) bool {
	// Check if password is between 12 and 72 characters
	passLen := len(password)
	return passLen >= 12 && passLen <= 72
}
