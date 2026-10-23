package creditcard_test

import (
	"errors"
	"slices"
	"testing"

	creditcard "github.com/memwey/go-credit-card"
)

func TestBrandRules(t *testing.T) {
	tests := []struct {
		brand   creditcard.Brand
		name    string
		lengths []int
		code    creditcard.SecurityCode
	}{
		{creditcard.AmericanExpress, "American Express", []int{15}, creditcard.SecurityCode{Name: "CID", Size: 4}},
		{creditcard.Visa, "Visa", []int{16, 18, 19}, creditcard.SecurityCode{Name: "CVV", Size: 3}},
		{creditcard.Mastercard, "Mastercard", []int{16}, creditcard.SecurityCode{Name: "CVC", Size: 3}},
		{creditcard.JCB, "JCB", []int{16, 17, 18, 19}, creditcard.SecurityCode{Name: "CVV", Size: 3}},
		{creditcard.Elo, "Elo", []int{16}, creditcard.SecurityCode{Name: "CVE", Size: 3}},
		{creditcard.Mir, "Mir", []int{16, 17, 18, 19}, creditcard.SecurityCode{Name: "CVP2", Size: 3}},
		{creditcard.DinersClub, "Diners Club", []int{14, 16, 19}, creditcard.SecurityCode{Name: "CVV", Size: 3}},
	}

	for _, tc := range tests {
		t.Run(string(tc.brand), func(t *testing.T) {
			if got := tc.brand.Name(); got != tc.name {
				t.Errorf("Name() = %q, want %q", got, tc.name)
			}

			if got := tc.brand.Lengths(); !slices.Equal(got, tc.lengths) {
				t.Errorf("Lengths() = %v, want %v", got, tc.lengths)
			}

			if got := tc.brand.SecurityCode(); got != tc.code {
				t.Errorf("SecurityCode() = %+v, want %+v", got, tc.code)
			}
		})
	}
}

func TestUnknownBrand(t *testing.T) {
	// Dankort was a hand-maintained brand before v1.2.0; credit-card-type does
	// not model it, so it is unknown now.
	for _, b := range []creditcard.Brand{"", "dankort", "Visa"} {
		t.Run(string(b), func(t *testing.T) {
			if got := b.Name(); got != "" {
				t.Errorf("Name() = %q, want empty", got)
			}

			if got := b.Lengths(); got != nil {
				t.Errorf("Lengths() = %v, want nil", got)
			}

			if got := b.SecurityCode(); got != (creditcard.SecurityCode{}) {
				t.Errorf("SecurityCode() = %+v, want zero", got)
			}

			if err := b.ValidateNumber("4242424242424242"); !errors.Is(err, creditcard.ErrUnknownBrand) {
				t.Errorf("ValidateNumber() = %v, want ErrUnknownBrand", err)
			}

			if err := b.ValidateCVV("123"); !errors.Is(err, creditcard.ErrUnknownBrand) {
				t.Errorf("ValidateCVV() = %v, want ErrUnknownBrand", err)
			}
		})
	}
}

func TestLengthsIsACopy(t *testing.T) {
	creditcard.AmericanExpress.Lengths()[0] = 99

	if got := creditcard.AmericanExpress.Lengths(); got[0] != 15 {
		t.Errorf("Lengths()[0] = %d after a caller changed an earlier result, want 15", got[0])
	}
}

func TestBrandValidateNumber(t *testing.T) {
	tests := []struct {
		name       string
		brand      creditcard.Brand
		number     string
		want       error
		wantLength bool // the error is a *LengthError
	}{
		{"Visa, 16 digits", creditcard.Visa, "4556974850403706", nil, false},
		{"Visa, 19 digits", creditcard.Visa, padded("4", 19), nil, false},
		{"American Express, 15 digits", creditcard.AmericanExpress, "378282246310005", nil, false},
		{"Visa, 17 digits", creditcard.Visa, padded("4", 17), creditcard.ErrInvalidNumber, true},
		{"American Express, 16 digits", creditcard.AmericanExpress, padded("37", 16), creditcard.ErrInvalidNumber, true},
		{"fails Luhn", creditcard.Visa, "4242424242424241", creditcard.ErrInvalidNumber, false},
		{"spaces", creditcard.Visa, "4242 4242 4242 4242", creditcard.ErrInvalidNumber, false},
		{"empty", creditcard.Visa, "", creditcard.ErrInvalidNumber, false},
		// The brand's rules apply whatever the number's prefix; whether it
		// belongs to the brand is Table's question, not this one's.
		{"another brand's prefix", creditcard.Visa, "5555555555554444", nil, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.brand.ValidateNumber(tc.number)
			if !errors.Is(err, tc.want) {
				t.Fatalf("ValidateNumber() = %v, want errors.Is(_, %v)", err, tc.want)
			}

			var lengthErr *creditcard.LengthError
			if got := errors.As(err, &lengthErr); got != tc.wantLength {
				t.Fatalf("ValidateNumber() = %v; is a *LengthError: %v, want %v", err, got, tc.wantLength)
			}

			if tc.wantLength {
				want := creditcard.LengthError{Brand: tc.brand, Length: len(tc.number), Lengths: tc.brand.Lengths()}
				if lengthErr.Brand != want.Brand || lengthErr.Length != want.Length || !slices.Equal(lengthErr.Lengths, want.Lengths) {
					t.Errorf("LengthError = %+v, want %+v", *lengthErr, want)
				}
			}
		})
	}
}

func TestBrandValidateCVV(t *testing.T) {
	tests := []struct {
		name  string
		brand creditcard.Brand
		cvv   string
		ok    bool
	}{
		{"three digits on Visa", creditcard.Visa, "123", true},
		{"four digits on American Express", creditcard.AmericanExpress, "1234", true},
		{"four digits on Visa", creditcard.Visa, "1234", false},
		{"three digits on American Express", creditcard.AmericanExpress, "123", false},
		{"two digits", creditcard.Visa, "12", false},
		{"not digits", creditcard.Visa, "12a", false},
		{"empty", creditcard.Visa, "", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.brand.ValidateCVV(tc.cvv)
			if tc.ok {
				if err != nil {
					t.Errorf("ValidateCVV() = %v, want nil", err)
				}

				return
			}

			var codeErr *creditcard.SecurityCodeError
			if !errors.Is(err, creditcard.ErrInvalidCVV) || !errors.As(err, &codeErr) {
				t.Fatalf("ValidateCVV() = %v, want a *SecurityCodeError", err)
			}

			if want := (creditcard.SecurityCodeError{Brand: tc.brand, Code: tc.brand.SecurityCode()}); *codeErr != want {
				t.Errorf("SecurityCodeError = %+v, want %+v", *codeErr, want)
			}
		})
	}
}

// brandConstants mirrors the exported constants. Keeping it in sync is the
// point of the test below: a brand added upstream shows up in Brands() but not
// here, and the test fails until a constant is declared.
var brandConstants = map[string]creditcard.Brand{
	"Visa":            creditcard.Visa,
	"Mastercard":      creditcard.Mastercard,
	"AmericanExpress": creditcard.AmericanExpress,
	"DinersClub":      creditcard.DinersClub,
	"Discover":        creditcard.Discover,
	"Troy":            creditcard.Troy,
	"JCB":             creditcard.JCB,
	"UnionPay":        creditcard.UnionPay,
	"Naranja":         creditcard.Naranja,
	"Verve":           creditcard.Verve,
	"Maestro":         creditcard.Maestro,
	"Elo":             creditcard.Elo,
	"Mir":             creditcard.Mir,
	"Hiper":           creditcard.Hiper,
	"Hipercard":       creditcard.Hipercard,
}

func TestBrandConstants(t *testing.T) {
	brands := creditcard.Brands()
	declared := make(map[creditcard.Brand]bool, len(brandConstants))

	for name, b := range brandConstants {
		declared[b] = true

		if !slices.Contains(brands, b) {
			t.Errorf("constant %s = %q, which is not in the package table", name, b)
		}
	}

	for _, b := range brands {
		if !declared[b] {
			t.Errorf("brand %q has no exported constant; declare one in brand.go", b)
		}

		if b.Name() == "" {
			t.Errorf("brand %q has no name", b)
		}
	}

	if len(brands) != len(brandConstants) {
		t.Errorf("Brands() returned %d brands, want %d", len(brands), len(brandConstants))
	}
}

func TestBrandsReturnsACopy(t *testing.T) {
	creditcard.Brands()[0] = "clobbered"

	if got := creditcard.Brands()[0]; got != creditcard.Visa {
		t.Errorf("Brands()[0] = %q after a caller changed an earlier result, want %q", got, creditcard.Visa)
	}
}

func TestUpstreamVersion(t *testing.T) {
	if creditcard.UpstreamVersion == "" {
		t.Error("UpstreamVersion is empty; the generated table should record the release it came from")
	}
}
