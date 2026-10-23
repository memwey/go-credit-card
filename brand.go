package creditcard

import (
	"fmt"
	"slices"
)

//go:generate go run ./internal/gen

// Brand is a card network, named by credit-card-type's id for it: "visa",
// "american-express" and so on. Brands are plain strings, so they compare with
// ==, work as map keys and go in switch statements.
//
// The rules a brand's cards follow -- number lengths and security code -- are
// facts about the brand, so they are methods here and do not depend on which
// Table identified the number. A Brand that is not in Brands is unknown: its
// methods report zero values or ErrUnknownBrand.
type Brand string

// The brands the package table can report, one per credit-card-type brand.
const (
	Visa            Brand = "visa"
	Mastercard      Brand = "mastercard"
	AmericanExpress Brand = "american-express"
	DinersClub      Brand = "diners-club"
	Discover        Brand = "discover"
	Troy            Brand = "troy"
	JCB             Brand = "jcb"
	UnionPay        Brand = "unionpay"
	Naranja         Brand = "naranja"
	Verve           Brand = "verve"
	Maestro         Brand = "maestro"
	Elo             Brand = "elo"
	Mir             Brand = "mir"
	Hiper           Brand = "hiper"
	Hipercard       Brand = "hipercard"
)

// Brands returns every brand the package table can report, in the order the
// table breaks ties between equally specific ranges.
func Brands() []Brand {
	out := make([]Brand, len(upstream))

	for i := range upstream {
		out[i] = upstream[i].Brand
	}

	return out
}

// SecurityCode describes a brand's card verification code the way the network
// names it -- CVV, CVC, CID and so on. Size is the exact number of digits.
type SecurityCode struct {
	Name string
	Size int
}

// Name returns the brand's display name, such as "American Express", or ""
// for an unknown brand.
func (b Brand) Name() string {
	if e := lookup(b); e != nil {
		return e.Name
	}

	return ""
}

// Lengths returns the card-number lengths the brand issues, or nil for an
// unknown brand. The slice is the caller's to keep.
func (b Brand) Lengths() []int {
	if e := lookup(b); e != nil {
		return slices.Clone(e.Lengths)
	}

	return nil
}

// SecurityCode returns the brand's security code, or the zero SecurityCode for
// an unknown brand.
func (b Brand) SecurityCode() SecurityCode {
	if e := lookup(b); e != nil {
		return e.Code
	}

	return SecurityCode{}
}

// ValidateNumber checks number against the brand's rules: it must be all
// digits, have one of the brand's lengths and pass the Luhn check. A wrong
// length is reported as a *LengthError. Every failure matches
// ErrInvalidNumber, except an unknown brand, which reports ErrUnknownBrand.
//
// ValidateNumber does not ask whether number belongs to the brand; that is
// identification, which is Table's job.
func (b Brand) ValidateNumber(number string) error {
	e := lookup(b)
	if e == nil {
		return unknownBrand(b)
	}

	if !isDigits(number) {
		return fmt.Errorf("%w: not all digits", ErrInvalidNumber)
	}

	if !slices.Contains(e.Lengths, len(number)) {
		return &LengthError{Brand: b, Length: len(number), Lengths: slices.Clone(e.Lengths)}
	}

	if !Luhn(number) {
		return fmt.Errorf("%w: fails the Luhn check", ErrInvalidNumber)
	}

	return nil
}

// ValidateCVV checks cvv against the brand's security code: exactly as many
// digits as the code's Size. A wrong code is reported as a *SecurityCodeError,
// which matches ErrInvalidCVV; an unknown brand reports ErrUnknownBrand.
func (b Brand) ValidateCVV(cvv string) error {
	e := lookup(b)
	if e == nil {
		return unknownBrand(b)
	}

	if !isDigits(cvv) || len(cvv) != e.Code.Size {
		return &SecurityCodeError{Brand: b, Code: e.Code}
	}

	return nil
}

// lookup returns b's entry in the package table, or nil when b is unknown.
func lookup(b Brand) *entry {
	i := slices.IndexFunc(upstream, func(e entry) bool { return e.Brand == b })
	if i < 0 {
		return nil
	}

	return &upstream[i]
}

func unknownBrand(b Brand) error {
	return fmt.Errorf("%w: %s", ErrUnknownBrand, brandLabel(b))
}

// brandLabel quotes b for an error message when it looks like a brand id --
// lower-case letters and dashes, as every id is -- so a card number passed as
// a Brand by mistake never reaches the message.
func brandLabel(b Brand) string {
	for i := 0; i < len(b); i++ {
		if c := b[i]; (c < 'a' || c > 'z') && c != '-' {
			return "not a brand id"
		}
	}

	return fmt.Sprintf("%q", b)
}
