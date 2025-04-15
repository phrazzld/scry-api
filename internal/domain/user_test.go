package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewUser(t *testing.T) {
	// Test valid user creation
	validEmail := "test@example.com"
	validPassword := "Password123!ABC" // 15 characters

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
	if err != ErrEmptyEmail {
		t.Errorf("Expected error %v, got %v", ErrEmptyEmail, err)
	}

	_, err = NewUser("invalidemail", validPassword)
	if err != ErrInvalidEmail {
		t.Errorf("Expected error %v, got %v", ErrInvalidEmail, err)
	}

	// Test invalid password
	_, err = NewUser(validEmail, "")
	if err != ErrEmptyPassword {
		t.Errorf("Expected error %v, got %v", ErrEmptyPassword, err)
	}

	// Test password too short
	_, err = NewUser(validEmail, "Pass1!")
	if err != ErrPasswordTooShort {
		t.Errorf("Expected error %v, got %v", ErrPasswordTooShort, err)
	}

	// Test password too long
	veryLongPassword := "AbcDefGhiJklMnoPqrStuVwxYz0123456789!@#$%^&*()_+=[]{}|;:,.<>?/~`" +
		"AbcDefGhiJklMnoPqrStuVwxYz0123456789"
	_, err = NewUser(validEmail, veryLongPassword)
	if err != ErrPasswordTooLong {
		t.Errorf("Expected error %v, got %v", ErrPasswordTooLong, err)
	}
}

func TestUserValidate(t *testing.T) {
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
	if err := invalidUser.Validate(); err != ErrEmptyUserID {
		t.Errorf("Expected error %v, got %v", ErrEmptyUserID, err)
	}

	// Test invalid email
	invalidUser = validUser
	invalidUser.Email = ""
	if err := invalidUser.Validate(); err != ErrEmptyEmail {
		t.Errorf("Expected error %v, got %v", ErrEmptyEmail, err)
	}

	invalidUser = validUser
	invalidUser.Email = "invalidemail"
	if err := invalidUser.Validate(); err != ErrInvalidEmail {
		t.Errorf("Expected error %v, got %v", ErrInvalidEmail, err)
	}

	// Test both password fields empty
	invalidUser = validUser
	invalidUser.HashedPassword = ""
	if err := invalidUser.Validate(); err != ErrEmptyPassword {
		t.Errorf("Expected error %v, got %v", ErrEmptyPassword, err)
	}

	// When Password is provided, check that password validation is done
	// and HashedPassword validation is skipped
	invalidUser = validUser
	invalidUser.Password = "abc"    // Too short
	invalidUser.HashedPassword = "" // Would normally cause ErrEmptyHashedPassword
	if err := invalidUser.Validate(); err != ErrPasswordTooShort {
		t.Errorf("Expected error %v, got %v", ErrPasswordTooShort, err)
	}
}

func TestValidateEmailFormat(t *testing.T) {
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

func TestUserValidate_PasswordComplexity(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  error
	}{
		{
			name:     "valid password with minimum length",
			password: "password12345",
			wantErr:  nil,
		},
		{
			name:     "valid password with maximum length",
			password: "12345678901234567890123456789012345678901234567890123456789012345678901", // exactly 72 characters
			wantErr:  nil,
		},
		{
			name:     "password too short",
			password: "Pass1!",
			wantErr:  ErrPasswordTooShort,
		},
		{
			name:     "password at exact minimum length",
			password: "123456789012", // 12 characters
			wantErr:  nil,
		},
		{
			name:     "password too long",
			password: "1234567890123456789012345678901234567890123456789012345678901234567890123", // 73 characters
			wantErr:  ErrPasswordTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				ID:             uuid.New(),
				Email:          "test@example.com",
				Password:       tt.password,
				HashedPassword: "some-hashed-password", // Not validated when Password is present
			}

			err := user.Validate()

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("Expected error %v, got %v", tt.wantErr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestValidatePasswordComplexity(t *testing.T) {
	tests := []struct {
		name     string
		password string
		want     bool
	}{
		{
			name:     "password with minimum length",
			password: "123456789012", // 12 characters
			want:     true,
		},
		{
			name:     "password too short",
			password: "12345678901", // 11 characters
			want:     false,
		},
		{
			name:     "password with maximum length",
			password: "12345678901234567890123456789012345678901234567890123456789012345678901", // exactly 72 characters
			want:     true,
		},
		{
			name:     "password too long",
			password: "1234567890123456789012345678901234567890123456789012345678901234567890123", // 73 characters
			want:     false,
		},
		{
			name:     "password with mix of characters within length",
			password: "Password123!@#",
			want:     true,
		},
		{
			name:     "password with only letters within length",
			password: "abcdefghijklmnopqrstuvwx",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validatePasswordComplexity(tt.password)
			if got != tt.want {
				t.Errorf("validatePasswordComplexity() = %v, want %v", got, tt.want)
			}
		})
	}
}
