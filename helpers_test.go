package creditcard_test

import (
	"strconv"
	"testing"

	creditcard "github.com/memwey/go-credit-card"
)

// withCheckDigit appends the Luhn check digit to body, so tests can build valid
// numbers of any length and prefix. It is written independently of
// creditcard.Luhn, which the tests check it against.
func withCheckDigit(body string) string {
	sum := 0

	for i := len(body) - 1; i >= 0; i-- {
		d := int(body[i] - '0')
		// The check digit will sit to the right, so the rightmost body digit
		// is the first to be doubled.
		if (len(body)-1-i)%2 == 0 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}

		sum += d
	}

	return body + strconv.Itoa((10-sum%10)%10)
}

// padded returns a Luhn-valid number of the given length starting with bin.
func padded(bin string, length int) string {
	body := bin
	for len(body) < length-1 {
		body += "0"
	}

	return withCheckDigit(body)
}

// mustTable builds a Table or fails the test.
func mustTable(t *testing.T, assignments ...creditcard.Assignment) *creditcard.Table {
	t.Helper()

	table, err := creditcard.NewTable(assignments...)
	if err != nil {
		t.Fatalf("NewTable() = %v, want nil", err)
	}

	return table
}
