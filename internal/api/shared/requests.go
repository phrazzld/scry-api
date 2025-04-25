package shared

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// Global validator instance for reuse
var validate = validator.New()

// DecodeJSON decodes the request body into the given struct.
func DecodeJSON(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return err
	}
	return nil
}

// ValidateRequest validates the given struct using the validator package.
func ValidateRequest(v interface{}) error {
	// Check if the object implements the Validate interface
	if validator, ok := v.(interface{ Validate() error }); ok {
		return validator.Validate()
	}

	// Otherwise, use the struct validator
	return validate.Struct(v)
}
