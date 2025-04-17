package shared

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// DecodeJSON decodes the request body into the given struct.
func DecodeJSON(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return err
	}
	return nil
}

// ValidateRequest validates the given struct using the validator package.
func ValidateRequest(v interface{}) error {
	validate := validator.New()
	return validate.Struct(v)
}
