package creditcard

import (
	"errors"
	"fmt"
)

// Errors returned by Card's validation and identification methods. Callers
// should use errors.Is rather than comparing their text.
var (
	ErrNumberTooShort       = errors.New("credit card number is not long enough")
	ErrTestNumberNotAllowed = errors.New("test numbers are not allowed")
	ErrInvalidNumber        = errors.New("invalid credit card number")
	ErrInvalidYear          = errors.New("invalid year")
	ErrInvalidMonth         = errors.New("invalid month")
	ErrExpired              = errors.New("credit card has expired")
	ErrInvalidCVV           = errors.New("invalid CVV")
	ErrUnknownMethod        = errors.New("unknown credit card method")
)

// SecurityCodeError reports the brand-specific security-code rule that was
// violated. It matches ErrInvalidCVV through errors.Is and exposes its details
// through errors.As.
type SecurityCodeError struct {
	Name     string
	Brand    string
	Required int
}

func (e *SecurityCodeError) Error() string {
	return fmt.Sprintf("invalid %s: %s requires %d digits", e.Name, e.Brand, e.Required)
}

// Unwrap classifies a SecurityCodeError as ErrInvalidCVV.
func (e *SecurityCodeError) Unwrap() error {
	return ErrInvalidCVV
}
