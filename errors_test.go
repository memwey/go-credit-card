package creditcard_test

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"testing"

	creditcard "github.com/memwey/go-credit-card"
)

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"one length", creditcard.Mastercard.ValidateNumber(padded("51", 17)),
			"invalid card number: Mastercard numbers have 16 digits, not 17"},
		{"two lengths", creditcard.Discover.ValidateNumber(padded("6011", 18)),
			"invalid card number: Discover numbers have 16 or 19 digits, not 18"},
		{"three lengths", creditcard.Visa.ValidateNumber(padded("4", 17)),
			"invalid card number: Visa numbers have 16, 18 or 19 digits, not 17"},
		{"not digits", creditcard.Visa.ValidateNumber("4242-4242"),
			"invalid card number: not all digits"},
		{"Luhn", creditcard.Visa.ValidateNumber("4242424242424241"),
			"invalid card number: fails the Luhn check"},
		{"security code", creditcard.AmericanExpress.ValidateCVV("123"),
			"invalid security code: American Express uses a 4 digit CID"},
		{"unknown brand", creditcard.Brand("dankort").ValidateCVV("123"),
			`unknown card brand: "dankort"`},
		{"something that is not a brand id", creditcard.Brand("Visa").ValidateCVV("123"),
			"unknown card brand: not a brand id"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil || tc.err.Error() != tc.want {
				t.Errorf("error = %v, want %q", tc.err, tc.want)
			}
		})
	}
}

func TestTypedErrorsUnwrapToTheirSentinel(t *testing.T) {
	var lengthErr error = &creditcard.LengthError{Brand: creditcard.Visa, Length: 17, Lengths: []int{16}}
	if !errors.Is(lengthErr, creditcard.ErrInvalidNumber) {
		t.Errorf("LengthError does not match ErrInvalidNumber")
	}

	var codeErr error = &creditcard.SecurityCodeError{Brand: creditcard.Visa, Code: creditcard.Visa.SecurityCode()}
	if !errors.Is(codeErr, creditcard.ErrInvalidCVV) {
		t.Errorf("SecurityCodeError does not match ErrInvalidCVV")
	}
}

// Card data must never end up in an error message, where it would be logged.
// The only numbers identification and validation messages carry are lengths
// and code sizes, so for inputs under a hundred digits none has a run of three
// or more digits, which any security code or card number would need.
// NewTable's may name a validated BIN range, but nothing longer than one.
func TestErrorsDoNotContainCardData(t *testing.T) {
	const pan = "4556974850403706"

	validationErrors := map[string]error{}
	add := func(name string, err error) {
		if err == nil {
			t.Fatalf("%s returned no error", name)
		}

		validationErrors[name] = err
	}

	for i, number := range []string{pan[:15] + "7", padded("4", 17), "4556 9748 5040 3706", withCheckDigit("111111111111111"), "6"} {
		_, err := creditcard.Default().Identify(number)
		if err == nil {
			_, err = creditcard.Default().Validate(creditcard.Card{Number: number, CVV: "123", ExpiryMonth: "12", ExpiryYear: "2030"}, now)
		}

		add("Identify or Validate, number "+strconv.Itoa(i+1), err)
	}

	add("ValidateNumber, Luhn", creditcard.Visa.ValidateNumber(pan[:15]+"7"))
	add("ValidateNumber, length", creditcard.Visa.ValidateNumber(padded("4", 17)))
	add("ValidateNumber, not digits", creditcard.Visa.ValidateNumber("4556-9748-5040-3706"))
	add("ValidateCVV, too long", creditcard.Visa.ValidateCVV("98765"))
	add("ValidateCVV, not digits", creditcard.Visa.ValidateCVV("98a"))
	add("Validate, CVV", second(creditcard.Default().Validate(creditcard.Card{Number: pan, CVV: "9876", ExpiryMonth: "12", ExpiryYear: "2030"}, now)))
	add("Validate, expiry", second(creditcard.Default().Validate(creditcard.Card{Number: pan, CVV: "987", ExpiryMonth: "1234", ExpiryYear: "2030"}, now)))
	add("ValidateExpiry", creditcard.ValidateExpiry("12", "202612", now))
	add("a card number as a Brand", creditcard.Brand(pan).ValidateNumber(pan))

	threeDigits := regexp.MustCompile(`[0-9]{3,}`)
	for name, err := range validationErrors {
		if run := threeDigits.FindString(err.Error()); run != "" {
			t.Errorf("%s: error %q contains the digits %q", name, err, run)
		}
	}

	tableErrors := map[string][]creditcard.Assignment{
		"a card number as a prefix":      {creditcard.Assign(creditcard.Visa, creditcard.Prefix(pan))},
		"a card number as one end":       {creditcard.Assign(creditcard.Visa, creditcard.Range("4556", pan))},
		"a card number with spaces":      {creditcard.Assign(creditcard.Visa, creditcard.Prefix("4556 9748 5040 3706"))},
		"a card number as a later range": {creditcard.Assign(creditcard.Visa, creditcard.Prefix("4556"), creditcard.Range(pan, pan))},
		"a card number as the brand":     {creditcard.Assign(creditcard.Brand(pan), creditcard.Prefix("4556"))},
	}

	longerThanABIN := regexp.MustCompile(`[0-9]{9,}`)
	for name, assignments := range tableErrors {
		_, err := creditcard.NewTable(assignments...)
		if err == nil {
			t.Fatalf("%s: NewTable() returned no error", name)
		}

		if run := longerThanABIN.FindString(err.Error()); run != "" || strings.Contains(err.Error(), "4556 9748") {
			t.Errorf("%s: error %q contains card data", name, err)
		}
	}
}

// second returns the error of a (Brand, error) pair.
func second(_ creditcard.Brand, err error) error {
	return err
}
