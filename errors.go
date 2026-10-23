package creditcard

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Errors reported by identification and validation. Match them with errors.Is:
// most are returned wrapped, with detail added to the message.
var (
	ErrInvalidNumber = errors.New("invalid card number")
	ErrUnknownBrand  = errors.New("unknown card brand")
	ErrInvalidCVV    = errors.New("invalid security code")
	ErrInvalidMonth  = errors.New("invalid expiry month")
	ErrInvalidYear   = errors.New("invalid expiry year")
	ErrExpired       = errors.New("card has expired")
)

// Errors reported by NewTable for a rejected assignment, besides
// ErrUnknownBrand for a brand that is not in Brands.
var (
	ErrInvalidBINRange       = errors.New("invalid BIN range")
	ErrConflictingAssignment = errors.New("BIN range assigned to two brands")
)

// LengthError reports a card number whose length its brand does not issue. It
// matches ErrInvalidNumber through errors.Is.
type LengthError struct {
	Brand Brand
	// Length is the length of the number that was checked.
	Length int
	// Lengths are the lengths the brand issues.
	Lengths []int
}

func (e *LengthError) Error() string {
	return fmt.Sprintf("%v: %s numbers have %s digits, not %d",
		ErrInvalidNumber, e.Brand.Name(), orList(e.Lengths), e.Length)
}

// Unwrap classifies a LengthError as ErrInvalidNumber.
func (e *LengthError) Unwrap() error {
	return ErrInvalidNumber
}

// SecurityCodeError reports a security code that does not fit its brand's. It
// matches ErrInvalidCVV through errors.Is.
type SecurityCodeError struct {
	Brand Brand
	// Code is the security code the brand requires.
	Code SecurityCode
}

func (e *SecurityCodeError) Error() string {
	return fmt.Sprintf("%v: %s uses a %d digit %s",
		ErrInvalidCVV, e.Brand.Name(), e.Code.Size, e.Code.Name)
}

// Unwrap classifies a SecurityCodeError as ErrInvalidCVV.
func (e *SecurityCodeError) Unwrap() error {
	return ErrInvalidCVV
}

// orList renders lengths the way a sentence lists them: "16", "16 or 19",
// "16, 18 or 19".
func orList(lengths []int) string {
	s := make([]string, len(lengths))
	for i, n := range lengths {
		s[i] = strconv.Itoa(n)
	}

	if len(s) < 2 {
		return strings.Join(s, "")
	}

	return strings.Join(s[:len(s)-1], ", ") + " or " + s[len(s)-1]
}
