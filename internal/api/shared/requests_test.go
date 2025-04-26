package shared

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestData defines a structure for JSON decoding tests
type TestData struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// ErrorCheck is a helper for validating errors
func ErrorCheck(t *testing.T, err error, shouldErr bool, expectedErrText string) {
	t.Helper()

	if shouldErr {
		assert.Error(t, err, "Expected an error")
		if expectedErrText != "" {
			assert.Contains(t, err.Error(), expectedErrText, "Error message should contain expected text")
		}
	} else {
		assert.NoError(t, err, "Expected no error")
	}
}

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name            string
		requestBody     string
		target          interface{}
		wantErr         bool
		expectedErrText string
		validateResult  func(t *testing.T, result interface{})
	}{
		{
			name:        "valid json",
			requestBody: `{"name": "test", "age": 30}`,
			target:      &TestData{},
			wantErr:     false,
			validateResult: func(t *testing.T, result interface{}) {
				data, ok := result.(*TestData)
				require.True(t, ok, "Result should be a *TestData")
				assert.Equal(t, "test", data.Name, "Name field mismatch")
				assert.Equal(t, 30, data.Age, "Age field mismatch")
			},
		},
		{
			name:            "invalid json",
			requestBody:     `{"name": "test", "age": 30,}`, // trailing comma
			target:          &TestData{},
			wantErr:         true,
			expectedErrText: "invalid character",
		},
		{
			name:            "empty body",
			requestBody:     "",
			target:          &TestData{},
			wantErr:         true,
			expectedErrText: "EOF",
		},
		{
			name:            "malformed json",
			requestBody:     `{"name": "te`,
			target:          &TestData{},
			wantErr:         true,
			expectedErrText: "unexpected EOF",
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

			// Check error result
			ErrorCheck(t, err, tc.wantErr, tc.expectedErrText)

			// If we expect success and have a validation function, use it
			if !tc.wantErr && tc.validateResult != nil {
				tc.validateResult(t, tc.target)
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
	ErrorCheck(t, err, true, "unexpected EOF")
}

// ValidatableStruct implements the Validate method for testing custom validation
type ValidatableStruct struct {
	Name string `validate:"required"`
	Age  int    `validate:"gte=18"`
}

// Validate checks the struct validity
func (v *ValidatableStruct) Validate() error {
	if v.Name == "invalid" {
		return errors.New("validation failed: invalid name")
	}
	return nil
}

// TestValidatorStruct uses the validator library directly
type TestValidatorStruct struct {
	Name string `validate:"required"`
	Age  int    `validate:"gte=18"`
}

func TestValidateRequest(t *testing.T) {
	tests := []struct {
		name            string
		req             interface{}
		wantErr         bool
		expectedErrText string
	}{
		{
			name: "valid request with custom validator",
			req: &ValidatableStruct{
				Name: "test",
				Age:  20,
			},
			wantErr: false,
		},
		{
			name: "invalid request with custom validator",
			req: &ValidatableStruct{
				Name: "invalid",
				Age:  20,
			},
			wantErr:         true,
			expectedErrText: "validation failed",
		},
		{
			name: "valid request with struct validator",
			req: &TestValidatorStruct{
				Name: "test",
				Age:  20,
			},
			wantErr: false,
		},
		{
			name: "invalid request with struct validator - missing required field",
			req: &TestValidatorStruct{
				Name: "", // Empty, will fail required validation
				Age:  20,
			},
			wantErr:         true,
			expectedErrText: "required",
		},
		{
			name: "invalid request with struct validator - value too low",
			req: &TestValidatorStruct{
				Name: "test",
				Age:  17, // Below 18, will fail gte validation
			},
			wantErr:         true,
			expectedErrText: "gte",
		},
		{
			name:    "request without validator",
			req:     &struct{ Name string }{"test"},
			wantErr: false,
		},
		{
			name:            "nil request",
			req:             nil,
			wantErr:         true,
			expectedErrText: "validator: (nil)",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRequest(tc.req)
			ErrorCheck(t, err, tc.wantErr, tc.expectedErrText)

			// Additional check to ensure validator actually ran for struct validator cases
			if strings.Contains(tc.name, "struct validator") && tc.wantErr {
				// Verify we're getting the expected validator error type
				var valErr validator.ValidationErrors
				assert.True(t, errors.As(err, &valErr) ||
					strings.Contains(err.Error(), "validation"),
					"Expected validator error type")
			}
		})
	}
}
