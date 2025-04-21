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
	// Validate password format first, before creating the user
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}

	user := &User{
		ID:        uuid.New(),
		Email:     email,
		Password:  password, // Plaintext password - must be hashed before storage
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	// Check other validation rules (ID, email, etc.)
	if err := user.Validate(); err != nil {
		return nil, err
	}

	return user, nil
}

// Validate checks if the User has valid data for persistence.
// This method only validates fields relevant for database persistence,
// not input validation concerns like password requirements.
// Returns an error if any field fails validation.
func (u *User) Validate() error {
	if u.ID == uuid.Nil {
		return ErrEmptyUserID
	}

	if u.Email == "" {
		return ErrEmptyEmail
	}

	// Basic email format validation
	if !validateEmailFormat(u.Email) {
		return ErrInvalidEmail
	}

	// For persistence, we need either a plaintext password (which will be hashed)
	// or a hashed password to be present
	if u.Password == "" && u.HashedPassword == "" {
		return ErrEmptyHashedPassword
	}

	return nil
}

// TODO(email-validation): Replace this basic email validation with a more robust solution:
//  1. Evaluate and select one of these approaches:
//     a. Use the mail.ParseAddress function from the net/mail standard library
//     b. Implement advanced regex validation (see RFC 5322 for guidelines)
//     c. Add a third-party validation library (e.g., github.com/go-playground/validator)
//  2. Create a validation package in internal/validation/ for all validation functions
//  3. Implement the new email validation in that package with comprehensive tests
//  4. Replace the current validateEmailFormat with the new implementation
//  5. Update any existing tests that might be affected by stricter validation
//
// Technical debt: This implementation is intentionally simple and has several limitations:
// 1. It only checks for @ and . characters in the right positions
// 2. It doesn't validate against RFC 5322 standards for email addresses
// 3. It doesn't handle international domains or special character requirements
// 4. It can't detect many invalid email patterns that would be rejected by mail servers
//
// Related resources:
// - RFC 5322 for email format: https://tools.ietf.org/html/rfc5322
// - RFC 6531 for international email: https://tools.ietf.org/html/rfc6531
// - net/mail package: https://pkg.go.dev/net/mail
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

// ValidatePassword validates the plaintext password format
// This should be called before setting the password field or creating a user
// Returns specific errors based on validation failures.
func ValidatePassword(password string) error {
	passLen := len(password)
	if passLen < 12 {
		return ErrPasswordTooShort
	}
	if passLen > 72 {
		return ErrPasswordTooLong
	}
	return nil
}

// validatePasswordComplexity checks if a password meets our security requirements
// based on length rather than character class composition.
//
// Password requirements:
// - Minimum length: 12 characters
// - Maximum length: 72 characters (bcrypt's practical limit)
// - No character class requirements (uppercase, lowercase, digits, symbols)
//
// Rationale for length-based approach:
//
//  1. Security research shows password length is more important than complexity
//     rules for resistance against brute force attacks. Each additional character
//     exponentially increases the password's entropy.
//
//  2. Complex character class requirements (uppercase, digits, symbols) often lead
//     to predictable patterns that weaken passwords (e.g., "Password1!") or increase
//     user frustration, leading to password reuse across services.
//
//  3. The 72-character maximum is a technical limitation of bcrypt, which truncates
//     passwords longer than 72 bytes. This prevents unnecessary password data from
//     being ignored during the hashing process.
//
//  4. This approach aligns with NIST SP 800-63B guidelines, which recommend
//     allowing longer passwords without arbitrary complexity requirements.
//
// Returns true if the password meets the length requirements, false otherwise.
func validatePasswordComplexity(password string) bool {
	// Check if password is between 12 and 72 characters
	passLen := len(password)
	return passLen >= 12 && passLen <= 72
}
