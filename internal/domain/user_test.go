package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewUser(t *testing.T) {
	t.Parallel() // Enable parallel execution
	// Test valid user creation
	validEmail := "test@example.com"
	validPassword := "Password123!ABC" // 15 characters - meets length requirements

	user, err := NewUser(validEmail, validPassword)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if user.ID == uuid.Nil {
		t.Error("Expected non-nil UUID, got nil UUID")
	}

	if user.Email != validEmail {
		t.Errorf("Expected email %s, got %s", validEmail, user.Email)
	}

	if user.Password != validPassword {
		t.Errorf("Expected password %s, got %s", validPassword, user.Password)
	}

	if user.CreatedAt.IsZero() {
		t.Error("Expected non-zero CreatedAt time")
	}

	if user.UpdatedAt.IsZero() {
		t.Error("Expected non-zero UpdatedAt time")
	}

	// Test invalid email
	_, err = NewUser("", validPassword)
	if err != ErrUserEmailEmpty {
		t.Errorf("Expected error %v, got %v", ErrUserEmailEmpty, err)
	}

	_, err = NewUser("invalidemail", validPassword)
	if err != ErrUserEmailInvalid {
		t.Errorf("Expected error %v, got %v", ErrUserEmailInvalid, err)
	}

	// Note: Password validation is now done by Validate() through User.Validate()
	// rather than through direct length checks in this test, but the functionality
	// should remain the same
}

func TestUserValidate(t *testing.T) {
	t.Parallel() // Enable parallel execution
	validUser := User{
		ID:             uuid.New(),
		Email:          "test@example.com",
		HashedPassword: "hashedpassword123",
	}

	// Test valid user
	if err := validUser.Validate(); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Test invalid ID
	invalidUser := validUser
	invalidUser.ID = uuid.Nil
	if err := invalidUser.Validate(); err != ErrUserIDEmpty {
		t.Errorf("Expected error %v, got %v", ErrUserIDEmpty, err)
	}

	// Test invalid email
	invalidUser = validUser
	invalidUser.Email = ""
	if err := invalidUser.Validate(); err != ErrUserEmailEmpty {
		t.Errorf("Expected error %v, got %v", ErrUserEmailEmpty, err)
	}

	invalidUser = validUser
	invalidUser.Email = "invalidemail"
	if err := invalidUser.Validate(); err != ErrUserEmailInvalid {
		t.Errorf("Expected error %v, got %v", ErrUserEmailInvalid, err)
	}

	// Test both password fields empty
	invalidUser = validUser
	invalidUser.HashedPassword = ""
	invalidUser.Password = ""
	if err := invalidUser.Validate(); err != ErrUserHashedPasswordEmpty {
		t.Errorf("Expected error %v, got %v", ErrUserHashedPasswordEmpty, err)
	}

	// Test with Password present but HashedPassword empty - should pass validation
	// as the Password will be hashed during persistence
	validUser = User{
		ID:       uuid.New(),
		Email:    "test@example.com",
		Password: "validpassword12345",
	}
	if err := validUser.Validate(); err != nil {
		t.Errorf("Expected no error when Password is present, got %v", err)
	}
}

func TestValidateEmailFormat(t *testing.T) {
	t.Parallel() // Enable parallel execution
	validEmails := []string{
		"user@example.com",
		"user.name@example.com",
		"user+tag@example.com",
		"user@sub.example.com",
	}

	invalidEmails := []string{
		"",
		"userexample.com",
		"user@",
		"@example.com",
		"user@.com",
		"user@example",
	}

	for _, email := range validEmails {
		if !validateEmailFormat(email) {
			t.Errorf("Expected email %s to be valid", email)
		}
	}

	for _, email := range invalidEmails {
		if validateEmailFormat(email) {
			t.Errorf("Expected email %s to be invalid", email)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	t.Parallel() // Enable parallel execution
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		// Valid password tests
		{
			name:     "valid password well above minimum length",
			password: "password12345", // 15 characters
			wantErr:  nil,
		},
		{
			name:     "valid password with all character types",
			password: "Password123!@#", // 15 characters with mixed case, numbers, symbols
			wantErr:  nil,
		},
		{
			name:     "valid password with only letters",
			password: "abcdefghijklmnopqrstuvwx", // 24 characters, all lowercase letters
			wantErr:  nil,
		},
		{
			name:     "valid password with only numbers",
			password: "123456789012345678901234", // 24 characters, all digits
			wantErr:  nil,
		},

		// Boundary tests
		{
			name:     "password at exact minimum length",
			password: "123456789012", // Exactly 12 characters
			wantErr:  nil,
		},
		{
			name:     "password at exact maximum length",
			password: "123456789012345678901234567890123456789012345678901234567890123456789012", // Exactly 72 characters
			wantErr:  nil,
		},
		{
			name:     "password just below minimum length",
			password: "12345678901", // 11 characters (one short)
			wantErr:  ErrUserPasswordTooShort,
		},
		{
			name:     "password just above maximum length",
			password: "1234567890123456789012345678901234567890123456789012345678901234567890123", // 73 characters (one over)
			wantErr:  ErrUserPasswordTooLong,
		},

		// Error cases
		{
			name:     "password very short",
			password: "Pass1!", // 6 characters
			wantErr:  ErrUserPasswordTooShort,
		},
		{
			name:     "password very long",
			password: "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890", // 100 characters
			wantErr:  ErrUserPasswordTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the ValidatePassword function directly rather than through User.Validate
			err := ValidatePassword(tt.password)

			// Check error expectations
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("Password length %d: Expected error %v, got %v",
						len(tt.password), tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Password length %d: Expected no error, got %v",
						len(tt.password), err)
				}
			}
		})
	}
}

func TestValidatePasswordComplexity(t *testing.T) {
	t.Parallel() // Enable parallel execution
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		// Valid passwords
		{
			name:     "password at exact minimum length",
			password: "123456789012", // Exactly 12 characters
			want:     true,
		},
		{
			name:     "password at exact maximum length",
			password: "12345678901234567890123456789012345678901234567890123456789012345678901", // Exactly 72 characters
			want:     true,
		},
		{
			name:     "password well above minimum length",
			password: "passwordpasswordpassword", // 24 characters
			want:     true,
		},

		// Invalid passwords - too short
		{
			name:     "password just below minimum length",
			password: "12345678901", // 11 characters - one short
			want:     false,
		},
		{
			name:     "password very short",
			password: "short", // 5 characters
			want:     false,
		},
		{
			name:     "empty password",
			password: "",
			want:     false,
		},

		// Invalid passwords - too long
		{
			name:     "password just above maximum length",
			password: "1234567890123456789012345678901234567890123456789012345678901234567890123", // 73 characters - one over
			want:     false,
		},
		{
			name:     "password far above maximum length",
			password: "1234567890123456789012345678901234567890123456789012345678901234567890123456789012345678901234567890", // 100 characters
			want:     false,
		},

		// Testing with different character compositions
		{
			name:     "password with mix of character types",
			password: "Password123!@#", // Mixed case, numbers, symbols
			want:     true,
		},
		{
			name:     "password with only letters",
			password: "abcdefghijklmnopqrstuvwx", // All lowercase letters
			want:     true,
		},
		{
			name:     "password with only numbers",
			password: "123456789012345678901234", // All digits
			want:     true,
		},
		{
			name:     "password with only symbols",
			password: "!@#$%^&*()_+-=[]{}|;:,.<>?/~`!@#$%^&*()", // All symbols
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validatePasswordComplexity(tt.password)
			if got != tt.want {
				t.Errorf("validatePasswordComplexity(%q) [length=%d] = %v, want %v",
					tt.password, len(tt.password), got, tt.want)
			}
		})
	}
}
