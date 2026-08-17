package creditcard

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
)

// Card holds generic information about the credit card
type Card struct {
	Number, Cvv, Month, Year string
	Company                  Company
}

// Company holds a short and long names of who has issued the credit card.
// Short is the brand id -- one of the constants in brands.go. For the number
// lengths and security code the brand uses, pass Short to SpecFor.
type Company struct {
	Short, Long string
}

// LastFour returns the last four digits of the credit card's number
func (c *Card) LastFour() (string, error) {
	if len(c.Number) < 4 {
		return "", errors.New("Credit card number is not long enough")
	}

	return c.Number[len(c.Number)-4 : len(c.Number)], nil
}

// LastFourDigits as an alias for LastFour
func (c *Card) LastFourDigits() (string, error) {
	return c.LastFour()
}

// Wipe returns the credit card with false/nullified/generic information
func (c *Card) Wipe() {
	c.Cvv, c.Number, c.Month, c.Year = "0000", "0000000000000000", "01", "1970"
}

// Validate returns nil or an error describing why the credit card didn't validate
// this method checks for expiration date, CCV/CVV and the credit card's numbers.
// For allowing test cards to go through, simply pass true (bool) as the first argument
func (c *Card) Validate(allowTestNumbers ...bool) error {
	err := c.ValidateExpiration()
	if err != nil {
		return err
	}

	err = c.ValidateCVV()
	if err != nil {
		return err
	}

	switch c.Number {
	// test cards: https://stripe.com/docs/testing
	case "4242424242424242",
		"4012888888881881",
		"4000056655665556",
		"5555555555554444",
		"5200828282828210",
		"5105105105105100",
		"378282246310005",
		"371449635398431",
		"6011111111111117",
		"6011000990139424",
		"30569309025904",
		"38520000023237",
		"3530111333300000",
		"3566002020360505",
		"4111111111111111",
		"4916909992637469",
		"4000111111111115",
		"2223000048400011",
		"6035227716427021":
		if len(allowTestNumbers) > 0 && allowTestNumbers[0] {
			return nil
		}

		return errors.New("Test numbers are not allowed")
	}

	valid := c.ValidateNumber()

	if !valid {
		return errors.New("Invalid credit card number")
	}

	return nil
}

// validates the credit card's expiration date
func (c *Card) ValidateExpiration() error {
	var year, month int
	var err error
	timeNow := timeNowCaller()

	if len(c.Year) < 3 {
		year, err = strconv.Atoi(strconv.Itoa(timeNow.UTC().Year())[:2] + c.Year)
		if err != nil {
			return errors.New("Invalid year")
		}
	} else {
		year, err = strconv.Atoi(c.Year)
		if err != nil {
			return errors.New("Invalid year")
		}
	}

	month, err = strconv.Atoi(c.Month)
	if err != nil {
		return errors.New("Invalid month")
	}

	if month < 1 || 12 < month {
		return errors.New("Invalid month")
	}

	if year < timeNowCaller().UTC().Year() {
		return errors.New("Credit card has expired")
	}

	if year == timeNowCaller().UTC().Year() && month < int(timeNowCaller().UTC().Month()) {
		return errors.New("Credit card has expired")
	}

	return nil
}

// ValidateCVV validates the card's security code. When the number is long
// enough to identify a brand, the brand's own rule applies: American Express
// requires a 4 digit CID, most other networks require exactly 3 digits. When no
// brand can be identified the generic 3-4 digit rule is used.
func (c *Card) ValidateCVV() error {
	if !isDigits(c.Cvv) {
		return errors.New("Invalid CVV")
	}

	if b := bestMatch(c.Number); b != nil && b.Code.Size > 0 {
		if len(c.Cvv) != b.Code.Size {
			return fmt.Errorf("Invalid %s: %s requires %d digits", b.Code.Name, b.Long, b.Code.Size)
		}

		return nil
	}

	if len(c.Cvv) < 3 || len(c.Cvv) > 4 {
		return errors.New("Invalid CVV")
	}

	return nil
}

// Method returns an error from MethodValidate() or returns the
// credit card with it's company / issuer attached to it
func (c *Card) Method() error {
	company, err := c.MethodValidate()

	if err != nil {
		return err
	}

	c.Company = company
	return nil
}

// MethodValidate adds/checks/verifies the credit card's company / issuer.
// When several brands claim the number, the one matching on the longest BIN
// pattern wins.
func (c *Card) MethodValidate() (Company, error) {
	if !isDigits(c.Number) {
		return Company{}, errors.New("Unknown credit card method")
	}

	b := bestMatch(c.Number)
	if b == nil {
		return Company{}, errors.New("Unknown credit card method")
	}

	return Company{Short: b.Short, Long: b.Long}, nil
}

// Luhn algorithm
// http://en.wikipedia.org/wiki/Luhn_algorithm

// ValidateNumber will check the credit card's number against the Luhn
// algorithm, and against the valid lengths of the brand it belongs to. Numbers
// that match no known brand fall back to the generic 13-19 digit rule.
func (c *Card) ValidateNumber() bool {
	if !isDigits(c.Number) {
		return false
	}

	numberLen := len(c.Number)

	if b := bestMatch(c.Number); b != nil && len(b.Lengths) > 0 {
		if !slices.Contains(b.Lengths, numberLen) {
			return false
		}
	} else if numberLen < 13 || numberLen > 19 {
		return false
	}

	var sum int
	var alternate bool

	for i := numberLen - 1; i > -1; i-- {
		mod, _ := strconv.Atoi(string(c.Number[i]))
		if alternate {
			mod *= 2
			if mod > 9 {
				mod = (mod % 10) + 1
			}
		}

		alternate = !alternate

		sum += mod
	}

	return sum%10 == 0
}
