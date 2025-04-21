package store_test

import (
	"errors"
	"testing"

	"github.com/phrazzld/scry-api/internal/store"
	"github.com/stretchr/testify/assert"
)

// TestErrorDefinitions ensures that the error definitions in the store
// package are defined as expected and can be used with errors.Is.
func TestErrorDefinitions(t *testing.T) {
	t.Parallel()

	// Create functions that return the standard errors
	// This simulates how store implementations might return these errors
	userNotFoundFn := func() error {
		return store.ErrUserNotFound
	}

	emailExistsFn := func() error {
		return store.ErrEmailExists
	}

	// Test ErrUserNotFound
	t.Run("ErrUserNotFound", func(t *testing.T) {
		t.Parallel()

		// Get the error from the function
		err := userNotFoundFn()

		// Verify it can be detected with errors.Is
		assert.True(t, errors.Is(err, store.ErrUserNotFound))
		assert.False(t, errors.Is(err, store.ErrEmailExists))

		// Verify the error can be unwrapped to ErrNotFound
		assert.True(t, errors.Is(err, store.ErrNotFound))
	})

	// Test ErrEmailExists
	t.Run("ErrEmailExists", func(t *testing.T) {
		t.Parallel()

		// Get the error from the function
		err := emailExistsFn()

		// Verify it can be detected with errors.Is
		assert.True(t, errors.Is(err, store.ErrEmailExists))
		assert.False(t, errors.Is(err, store.ErrUserNotFound))

		// Verify the error can be unwrapped to ErrDuplicate
		assert.True(t, errors.Is(err, store.ErrDuplicate))
	})
}

// The following is commented out intentionally.
// This is a common pattern to check at compile-time that a type implements an interface.
// Uncomment to verify that your custom types properly implement the UserStore interface.
/*
type CompileTimeInterfaceCheck struct{}
var _ store.UserStore = (*CompileTimeInterfaceCheck)(nil) // Will fail, as CompileTimeInterfaceCheck doesn't implement UserStore
*/
