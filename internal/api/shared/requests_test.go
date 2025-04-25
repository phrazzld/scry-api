package shared

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name        string
		requestBody string
		target      interface{}
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid json",
			requestBody: `{"name": "test", "age": 30}`,
			target: &struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{},
			wantErr: false,
		},
		{
			name:        "invalid json",
			requestBody: `{"name": "test", "age": 30,}`, // trailing comma
			target:      &struct{}{},
			wantErr:     true,
			errContains: "invalid character",
		},
		{
			name:        "empty body",
			requestBody: "",
			target:      &struct{}{},
			wantErr:     true,
			errContains: "EOF",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create request with body
			req := httptest.NewRequest(
				http.MethodPost,
				"/test",
				bytes.NewBufferString(tc.requestBody),
			)

			// Call function
			err := DecodeJSON(req, tc.target)

			// Check result
			if tc.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
			} else {
				assert.NoError(t, err)

				// For valid JSON case, check that the target was populated correctly
				if tc.name == "valid json" {
					data := tc.target.(*struct {
						Name string `json:"name"`
						Age  int    `json:"age"`
					})
					assert.Equal(t, "test", data.Name)
					assert.Equal(t, 30, data.Age)
				}
			}
		})
	}
}

// Mock for http.Request that will return a read error
type errorReader struct{}

func (er errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestDecodeJSONWithReadError(t *testing.T) {
	// Create request with a body that will error on read
	req := httptest.NewRequest(http.MethodPost, "/test", errorReader{})

	// Call function
	var target struct{}
	err := DecodeJSON(req, &target)

	// Check result
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected EOF")
}

// Mock validator interface
type ValidatableStruct struct {
	Name string `validate:"required"`
	Age  int    `validate:"gte=18"`
}

func (v *ValidatableStruct) Validate() error {
	if v.Name == "invalid" {
		// Return a mock validator error
		return &validator.ValidationErrors{}
	}
	return nil
}

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name    string
		req     interface{}
		wantErr bool
	}{
		{
			name: "valid request with validator",
			req: &ValidatableStruct{
				Name: "test",
				Age:  20,
			},
			wantErr: false,
		},
		{
			name: "invalid request with validator",
			req: &ValidatableStruct{
				Name: "invalid",
				Age:  20,
			},
			wantErr: true,
		},
		{
			name:    "request without validator",
			req:     &struct{ Name string }{"test"},
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRequest(tc.req)

			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
