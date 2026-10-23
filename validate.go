package creditcard

import "time"

// Luhn reports whether number passes the Luhn check-digit algorithm
// (https://en.wikipedia.org/wiki/Luhn_algorithm). It is false for anything but
// a non-empty run of digits.
func Luhn(number string) bool {
	if !isDigits(number) {
		return false
	}

	sum, double := 0, false

	for i := len(number) - 1; i >= 0; i-- {
		d := int(number[i] - '0')

		if double {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}

		sum += d
		double = !double
	}

	return sum%10 == 0
}

// ValidateExpiry checks an expiry date as printed on a card. month is 1-12,
// with or without a leading zero. year has two digits, read in now's century,
// or four digits without a leading zero.
//
// A card is good through the last day of its expiry month as seen in now's
// location, so pass now in the time zone that should decide. It returns
// ErrInvalidMonth or ErrInvalidYear for a malformed date and ErrExpired for a
// date already past.
func ValidateExpiry(month, year string, now time.Time) error {
	if !isDigits(month) || len(month) > 2 {
		return ErrInvalidMonth
	}

	m := time.Month(atoi(month))
	if m < time.January || m > time.December {
		return ErrInvalidMonth
	}

	if !isDigits(year) || (len(year) != 2 && len(year) != 4) || (len(year) == 4 && year[0] == '0') {
		return ErrInvalidYear
	}

	y := atoi(year)
	if len(year) == 2 {
		y += now.Year() / 100 * 100
	}

	if y < now.Year() || (y == now.Year() && m < now.Month()) {
		return ErrExpired
	}

	return nil
}

// isDigits reports whether s is a non-empty run of ASCII digits.
func isDigits(s string) bool {
	if s == "" {
		return false
	}

	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}

	return true
}

// atoi parses a short run of digits the caller has already checked.
func atoi(s string) int {
	n := 0

	for i := 0; i < len(s); i++ {
		n = n*10 + int(s[i]-'0')
	}

	return n
}
