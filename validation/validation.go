// SPDX-License-Identifier: 	AGPL-3.0-or-later

package validation

import (
	"errors"

	"github.com/go-playground/validator/v10"
)

var Validate = validator.New(validator.WithRequiredStructEnabled())

var ErrNilObject = errors.New("object is nil")

// Error is a map of fields and their validation errors.
type Error struct {
	Details string
	fields  map[string]string
}

func (e *Error) Error() string {
	return "invalid data: " + e.Details
}

// IsValid will check if an object is valid according to the validation tags on
// the object. It does not return any details of validation issues, only a
// boolean for valid (true) or invalid (false).
func IsValid[T any](obj T) bool {
	err := Validate.Struct(obj)
	return err != nil
}
