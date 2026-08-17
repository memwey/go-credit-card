package creditcard

import (
	"errors"
	"testing"
	"time"
)

func TestSentinelErrors(t *testing.T) {
	now := time.Date(2030, time.June, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		target error
		call   func() error
	}{
		{"number too short", ErrNumberTooShort, func() error {
			_, err := (&Card{Number: "123"}).LastFour()
			return err
		}},
		{"test number", ErrTestNumberNotAllowed, func() error {
			return (&Card{Number: "4242424242424242", Cvv: "111", Month: "12", Year: "2099"}).Validate()
		}},
		{"invalid number", ErrInvalidNumber, func() error {
			return (&Card{Number: "4242424242424241", Cvv: "111", Month: "12", Year: "2099"}).Validate()
		}},
		{"invalid year", ErrInvalidYear, func() error {
			return (&Card{Month: "12", Year: "ab"}).validateExpiration(now)
		}},
		{"invalid month", ErrInvalidMonth, func() error {
			return (&Card{Month: "13", Year: "2030"}).validateExpiration(now)
		}},
		{"expired", ErrExpired, func() error {
			return (&Card{Month: "05", Year: "2030"}).validateExpiration(now)
		}},
		{"invalid CVV", ErrInvalidCVV, func() error {
			return (&Card{Cvv: "12"}).ValidateCVV()
		}},
		{"unknown method", ErrUnknownMethod, func() error {
			_, err := (&Card{Number: "not-digits"}).MethodValidate()
			return err
		}},
		{"legacy unknown method", ErrUnknownMethod, func() error {
			_, err := (&Card{Number: "not-digits"}).LegacyCompany()
			return err
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.call()
			if !errors.Is(err, tc.target) {
				t.Fatalf("error = %v, want errors.Is(_, %v)", err, tc.target)
			}

		})
	}
}

func TestSecurityCodeErrorSupportsIsAndAs(t *testing.T) {
	err := (&Card{Number: "378282246310005", Cvv: "111"}).ValidateCVV()
	if !errors.Is(err, ErrInvalidCVV) {
		t.Fatalf("error = %v, want errors.Is(_, ErrInvalidCVV)", err)
	}

	var codeErr *SecurityCodeError
	if !errors.As(err, &codeErr) {
		t.Fatalf("error type = %T, want *SecurityCodeError", err)
	}

	if codeErr.Name != "CID" || codeErr.Brand != "American Express" || codeErr.Required != 4 {
		t.Errorf("SecurityCodeError = %+v, want CID/American Express/4", codeErr)
	}

	const want = "invalid CID: American Express requires 4 digits"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}
