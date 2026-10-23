package creditcard_test

import (
	"errors"
	"testing"
	"time"

	creditcard "github.com/memwey/go-credit-card"
)

func TestLuhn(t *testing.T) {
	tests := []struct {
		number string
		want   bool
	}{
		{"4556974850403706", true},
		{"378282246310005", true},
		{"30569309025904", true},
		{"0", true},
		{"4242424242424241", false},
		{"18", true},
		{"19", false},
		{"", false},
		{"4242 4242 4242 4242", false},
		{"42a2", false},
	}

	for _, tc := range tests {
		if got := creditcard.Luhn(tc.number); got != tc.want {
			t.Errorf("Luhn(%q) = %v, want %v", tc.number, got, tc.want)
		}
	}

	// The helper the other tests build numbers with must agree.
	for _, body := range []string{"4", "37828224631000", "65003100000000000", "123456789"} {
		if n := withCheckDigit(body); !creditcard.Luhn(n) {
			t.Errorf("Luhn(withCheckDigit(%q)) = false, want true", body)
		}
	}
}

func TestValidateExpiry(t *testing.T) {
	now := time.Date(2026, time.October, 23, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name        string
		month, year string
		want        error
	}{
		{"this month", "10", "2026", nil},
		{"next month", "11", "2026", nil},
		{"next year", "01", "2027", nil},
		{"no leading zero", "1", "2027", nil},
		{"two-digit year", "10", "26", nil},
		{"last month", "09", "2026", creditcard.ErrExpired},
		{"last year", "12", "2025", creditcard.ErrExpired},
		{"two-digit year, last month", "9", "26", creditcard.ErrExpired},

		{"month zero", "0", "2027", creditcard.ErrInvalidMonth},
		{"month 00", "00", "2027", creditcard.ErrInvalidMonth},
		{"month thirteen", "13", "2027", creditcard.ErrInvalidMonth},
		{"three-digit month", "001", "2027", creditcard.ErrInvalidMonth},
		{"signed month", "+1", "2027", creditcard.ErrInvalidMonth},
		{"month with a space", " 1", "2027", creditcard.ErrInvalidMonth},
		{"empty month", "", "2027", creditcard.ErrInvalidMonth},

		{"one-digit year", "1", "7", creditcard.ErrInvalidYear},
		{"three-digit year", "1", "202", creditcard.ErrInvalidYear},
		{"five-digit year", "1", "20270", creditcard.ErrInvalidYear},
		// Not the year 26, nor 2026 in disguise: four digits start with a 1-9.
		{"four digits with a leading zero", "12", "0026", creditcard.ErrInvalidYear},
		{"year zero", "12", "0000", creditcard.ErrInvalidYear},
		{"signed year", "1", "+27", creditcard.ErrInvalidYear},
		{"letters", "1", "ab", creditcard.ErrInvalidYear},
		{"empty year", "1", "", creditcard.ErrInvalidYear},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := creditcard.ValidateExpiry(tc.month, tc.year, now); !errors.Is(err, tc.want) {
				t.Errorf("ValidateExpiry(%q, %q) = %v, want errors.Is(_, %v)", tc.month, tc.year, err, tc.want)
			}
		})
	}
}

func TestValidateExpiryUsesNowsLocation(t *testing.T) {
	tokyo := time.FixedZone("JST", 9*60*60)

	// The first half hour of November in Tokyo is still October in UTC.
	instant := time.Date(2026, time.November, 1, 0, 30, 0, 0, tokyo)

	if err := creditcard.ValidateExpiry("10", "2026", instant); !errors.Is(err, creditcard.ErrExpired) {
		t.Errorf("in Tokyo: ValidateExpiry() = %v, want ErrExpired", err)
	}

	if err := creditcard.ValidateExpiry("10", "2026", instant.UTC()); err != nil {
		t.Errorf("in UTC: ValidateExpiry() = %v, want nil", err)
	}
}

func TestValidateExpiryReadsTwoDigitYearsInNowsCentury(t *testing.T) {
	now := time.Date(2099, time.December, 1, 0, 0, 0, 0, time.UTC)

	if err := creditcard.ValidateExpiry("12", "99", now); err != nil {
		t.Errorf(`"99" in 2099: ValidateExpiry() = %v, want nil`, err)
	}

	if err := creditcard.ValidateExpiry("01", "00", now); !errors.Is(err, creditcard.ErrExpired) {
		t.Errorf(`"00" in 2099: ValidateExpiry() = %v, want ErrExpired (2000, not 2100)`, err)
	}
}
