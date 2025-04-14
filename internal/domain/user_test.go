package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewUser(t *testing.T) {
	// Test valid user creation
	validEmail := "test@example.com"
	validPassword := "hashedpassword123"

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

	if user.HashedPassword != validPassword {
		t.Errorf("Expected hashed password %s, got %s", validPassword, user.HashedPassword)
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
	if err != ErrEmptyHashedPassword {
		t.Errorf("Expected error %v, got %v", ErrEmptyHashedPassword, err)
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

	// Test invalid password
	invalidUser = validUser
	invalidUser.HashedPassword = ""
	if err := invalidUser.Validate(); err != ErrEmptyHashedPassword {
		t.Errorf("Expected error %v, got %v", ErrEmptyHashedPassword, err)
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
