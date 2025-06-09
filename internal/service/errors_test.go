package service

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSentinelErrors(t *testing.T) {
	t.Run("ErrNotOwned", func(t *testing.T) {
		assert.Equal(t, "resource is owned by another user", ErrNotOwned.Error())
		assert.True(t, errors.Is(ErrNotOwned, ErrNotOwned))
	})

	t.Run("ErrStatsNotFound", func(t *testing.T) {
		assert.Equal(t, "user card statistics not found", ErrStatsNotFound.Error())
		assert.True(t, errors.Is(ErrStatsNotFound, ErrStatsNotFound))
	})

	t.Run("sentinel errors are different", func(t *testing.T) {
		assert.False(t, errors.Is(ErrNotOwned, ErrStatsNotFound))
		assert.False(t, errors.Is(ErrStatsNotFound, ErrNotOwned))
	})
}

func TestServiceError_Error(t *testing.T) {
	tests := []struct {
		name     string
		service  string
		op       string
		err      error
		expected string
	}{
		{
			name:     "with underlying error",
			service:  "user",
			op:       "create",
			err:      errors.New("database connection failed"),
			expected: "user service create operation failed: database connection failed",
		},
		{
			name:     "without underlying error",
			service:  "card",
			op:       "delete",
			err:      nil,
			expected: "card service delete operation failed",
		},
		{
			name:     "with sentinel error",
			service:  "card",
			op:       "get",
			err:      ErrNotOwned,
			expected: "card service get operation failed: resource is owned by another user",
		},
		{
			name:     "empty service name",
			service:  "",
			op:       "update",
			err:      errors.New("validation failed"),
			expected: " service update operation failed: validation failed",
		},
		{
			name:     "empty operation name",
			service:  "memo",
			op:       "",
			err:      errors.New("invalid input"),
			expected: "memo service  operation failed: invalid input",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceErr := &ServiceError{
				Service: tt.service,
				Op:      tt.op,
				Err:     tt.err,
			}

			assert.Equal(t, tt.expected, serviceErr.Error())
		})
	}
}

func TestServiceError_Unwrap(t *testing.T) {
	tests := []struct {
		name              string
		underlyingError   error
		expectedUnwrapped error
	}{
		{
			name:              "with underlying error",
			underlyingError:   errors.New("database error"),
			expectedUnwrapped: errors.New("database error"),
		},
		{
			name:              "with sentinel error",
			underlyingError:   ErrNotOwned,
			expectedUnwrapped: ErrNotOwned,
		},
		{
			name:              "with nil error",
			underlyingError:   nil,
			expectedUnwrapped: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serviceErr := &ServiceError{
				Service: "test",
				Op:      "test",
				Err:     tt.underlyingError,
			}

			unwrapped := serviceErr.Unwrap()
			if tt.expectedUnwrapped == nil {
				assert.Nil(t, unwrapped)
			} else {
				assert.Equal(t, tt.expectedUnwrapped.Error(), unwrapped.Error())
			}
		})
	}
}

func TestServiceError_ErrorsIs(t *testing.T) {
	underlyingErr := errors.New("database connection failed")
	serviceErr := &ServiceError{
		Service: "user",
		Op:      "create",
		Err:     underlyingErr,
	}

	t.Run("errors.Is works with wrapped error", func(t *testing.T) {
		assert.True(t, errors.Is(serviceErr, underlyingErr))
	})

	t.Run("errors.Is works with sentinel errors", func(t *testing.T) {
		sentinelServiceErr := &ServiceError{
			Service: "card",
			Op:      "get",
			Err:     ErrNotOwned,
		}
		assert.True(t, errors.Is(sentinelServiceErr, ErrNotOwned))
	})

	t.Run("errors.Is returns false for different errors", func(t *testing.T) {
		differentErr := errors.New("different error")
		assert.False(t, errors.Is(serviceErr, differentErr))
	})
}

func TestServiceError_ErrorsAs(t *testing.T) {
	originalErr := &ServiceError{
		Service: "original",
		Op:      "test",
		Err:     errors.New("inner error"),
	}

	wrappedErr := &ServiceError{
		Service: "wrapper",
		Op:      "wrap",
		Err:     originalErr,
	}

	t.Run("errors.As works with ServiceError", func(t *testing.T) {
		var serviceErr *ServiceError
		assert.True(t, errors.As(wrappedErr, &serviceErr))
		assert.Equal(t, "wrapper", serviceErr.Service)
		assert.Equal(t, "wrap", serviceErr.Op)
	})

	t.Run("errors.As finds nested ServiceError", func(t *testing.T) {
		var serviceErr *ServiceError
		found := errors.As(wrappedErr.Err, &serviceErr)
		assert.True(t, found)
		assert.Equal(t, "original", serviceErr.Service)
		assert.Equal(t, "test", serviceErr.Op)
	})
}

func TestNewServiceError(t *testing.T) {
	tests := []struct {
		name    string
		service string
		op      string
		err     error
	}{
		{
			name:    "with underlying error",
			service: "user",
			op:      "create",
			err:     errors.New("database error"),
		},
		{
			name:    "with sentinel error",
			service: "card",
			op:      "get",
			err:     ErrNotOwned,
		},
		{
			name:    "with nil error",
			service: "memo",
			op:      "update",
			err:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewServiceError(tt.service, tt.op, tt.err)

			// Verify it returns a ServiceError
			var serviceErr *ServiceError
			assert.True(t, errors.As(err, &serviceErr))

			// Verify fields are set correctly
			assert.Equal(t, tt.service, serviceErr.Service)
			assert.Equal(t, tt.op, serviceErr.Op)
			assert.Equal(t, tt.err, serviceErr.Err)

			// Verify error message format
			expectedMsg := tt.service + " service " + tt.op + " operation failed"
			if tt.err != nil {
				expectedMsg += ": " + tt.err.Error()
			}
			assert.Equal(t, expectedMsg, err.Error())

			// Verify unwrapping works
			assert.Equal(t, tt.err, errors.Unwrap(err))

			// Verify errors.Is works if underlying error is provided
			if tt.err != nil {
				assert.True(t, errors.Is(err, tt.err))
			}
		})
	}
}

func TestServiceError_ChainedErrors(t *testing.T) {
	// Test error chaining scenarios
	baseErr := errors.New("database connection lost")
	serviceErr1 := NewServiceError("store", "query", baseErr)
	serviceErr2 := NewServiceError("service", "get", serviceErr1)

	t.Run("chained errors maintain unwrapping", func(t *testing.T) {
		// Should be able to find the base error through the chain
		assert.True(t, errors.Is(serviceErr2, baseErr))
		assert.True(t, errors.Is(serviceErr2, serviceErr1))
	})

	t.Run("error message includes full context", func(t *testing.T) {
		expected := "service service get operation failed: store service query operation failed: database connection lost"
		assert.Equal(t, expected, serviceErr2.Error())
	})

	t.Run("errors.As finds ServiceError at any level", func(t *testing.T) {
		var serviceErr *ServiceError

		// Should find the outermost ServiceError first
		assert.True(t, errors.As(serviceErr2, &serviceErr))
		assert.Equal(t, "service", serviceErr.Service)
		assert.Equal(t, "get", serviceErr.Op)
	})
}
