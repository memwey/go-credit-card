package creditcard_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	creditcard "github.com/memwey/go-credit-card"
)

func TestDefaultIdentify(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   creditcard.Brand
	}{
		{"American Express", "3782822463", creditcard.AmericanExpress},
		{"UnionPay 62", "62111111111", creditcard.UnionPay},
		{"UnionPay 81", "810111111111", creditcard.UnionPay},
		{"Discover", "60111111111", creditcard.Discover},
		{"Elo", "6362970000457013", creditcard.Elo},
		{"Hiper six-digit BIN", "6370950000000005", creditcard.Hiper},
		{"Hiper eight-digit BIN", "6373742300000001", creditcard.Hiper},
		{"Hipercard", "6062826786276634", creditcard.Hipercard},
		{"JCB", "353011133", creditcard.JCB},
		{"Maestro", "501855555", creditcard.Maestro},
		{"Mastercard 51-55", "5425233430109903", creditcard.Mastercard},
		{"Mastercard 2 series", "2223000048410010", creditcard.Mastercard},
		{"Mir", "2200770000000000", creditcard.Mir},
		{"Naranja", "5895626746595650", creditcard.Naranja},
		{"Troy four-digit BIN", "9792000000000000", creditcard.Troy},
		{"Troy eight-digit BIN", "6508370100000000", creditcard.Troy},
		{"Verve", "5060990000000000", creditcard.Verve},
		{"Visa", "4242424242", creditcard.Visa},
		{"Diners Club 300-305", "30022111111111", creditcard.DinersClub},
		{"Diners Club 36", "36001111111111", creditcard.DinersClub},
		{"Diners Club 38", "38001111111111", creditcard.DinersClub},

		// Brands credit-card-type does not model are reported the way it
		// reports their numbers. Assign a range to change that.
		{"Visa Electron's 4026 is Visa", "4026424242", creditcard.Visa},
		{"Dankort's 5019 is Maestro", "501955555", creditcard.Maestro},
		{"Cabal's 603522 is Maestro", "6035227716427021", creditcard.Maestro},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := creditcard.Default().Identify(tc.number); err != nil || got != tc.want {
				t.Errorf("Identify(%q) = (%q, %v), want %q", tc.number, got, err, tc.want)
			}
		})
	}
}

func TestIdentifyPrefersTheLongestRange(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   creditcard.Brand
	}{
		{"Elo 401178 beats Visa 4", "4011780000000000", creditcard.Elo},
		{"Hiper 637095 beats Maestro 63", "6370950000000005", creditcard.Hiper},
		{"Elo 636297 beats Maestro 63", "6362970000457013", creditcard.Elo},
		{"Discover 6011 beats Maestro 6", "6011111111111117", creditcard.Discover},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := creditcard.Default().Identify(tc.number); err != nil || got != tc.want {
				t.Errorf("Identify(%q) = (%q, %v), want %q", tc.number, got, err, tc.want)
			}
		})
	}
}

func TestIdentifyIgnoresLength(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   creditcard.Brand
	}{
		// Too long for the brand, but still that brand: the length is
		// ValidateNumber's to report, not a reason to pick another brand.
		{"17-digit American Express", "34000000000000009", creditcard.AmericanExpress},
		{"17-digit Mastercard", "51000000000000003", creditcard.Mastercard},
		{"19-digit Elo is not Discover", padded("650031", 19), creditcard.Elo},

		// Partial numbers, as while the user types.
		{"no other brand starts with 42", "42", creditcard.Visa},
		{"only JCB can still match", "35", creditcard.JCB},
		{"Discover once 6011 is complete", "6011", creditcard.Discover},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := creditcard.Default().Identify(tc.number); err != nil || got != tc.want {
				t.Errorf("Identify(%q) = (%q, %v), want %q", tc.number, got, err, tc.want)
			}
		})
	}
}

func TestIdentifyRejects(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   error
	}{
		{"empty", "", creditcard.ErrInvalidNumber},
		{"letters", "abcdefg", creditcard.ErrInvalidNumber},
		{"spaces are not stripped", "4242 4242 4242 4242", creditcard.ErrInvalidNumber},
		{"dashes are not stripped", "4242-4242-4242-4242", creditcard.ErrInvalidNumber},
		{"unknown prefix", "1112424242", creditcard.ErrUnknownBrand},
		{"too few digits to choose", "6", creditcard.ErrUnknownBrand},
		{"leading zero", "0000000000000000", creditcard.ErrUnknownBrand},
		{"Diners Club enRoute 2014 is not in upstream", "20142111111111", creditcard.ErrUnknownBrand},
		{"Diners Club 309 is not in upstream", "3090000000000000", creditcard.ErrUnknownBrand},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := creditcard.Default().Identify(tc.number)
			if !errors.Is(err, tc.want) || got != "" {
				t.Errorf("Identify(%q) = (%q, %v), want errors.Is(_, %v)", tc.number, got, err, tc.want)
			}
		})
	}
}

// Every way of asking for the package table alone identifies the same way.
func TestDefaultTable(t *testing.T) {
	tables := map[string]*creditcard.Table{
		"NewTable()": mustTable(t),
		"zero Table": {},
		"nil *Table": nil,
	}

	for _, number := range []string{"4242424242424242", "6500310000000000", "3528000000000007", "1112424242", "12ab"} {
		want, wantErr := creditcard.Default().Identify(number)

		for name, table := range tables {
			// Same code path, so the same error text.
			got, err := table.Identify(number)
			if got != want || fmt.Sprint(err) != fmt.Sprint(wantErr) {
				t.Errorf("%s: Identify(%q) = (%q, %v), want (%q, %v)", name, number, got, err, want, wantErr)
			}
		}
	}
}

func TestDefaultCannotBeReplaced(t *testing.T) {
	custom := mustTable(t, creditcard.Assign(creditcard.AmericanExpress, creditcard.Prefix("4")))

	// Overwriting the Table Default returned changes that copy only.
	*creditcard.Default() = *custom

	if got, err := creditcard.Default().Identify("4242424242424242"); err != nil || got != creditcard.Visa {
		t.Errorf("Identify() after overwriting a Default() = (%q, %v), want %q", got, err, creditcard.Visa)
	}
}

func TestIdentifyAssignedRanges(t *testing.T) {
	// The shape of a card network's own BIN table: ranges the package table
	// lacks, and ranges it files under another brand.
	table := mustTable(t,
		creditcard.Assign(creditcard.JCB, creditcard.Range("30910000", "30949999"), creditcard.Range("33390000", "33499999")),
		creditcard.Assign(creditcard.Discover, creditcard.Range("6440", "6599")),
	)

	tests := []struct {
		name     string
		number   string
		want     creditcard.Brand // "" means ErrUnknownBrand
		upstream creditcard.Brand // what Default reports
	}{
		{"8-digit JCB IIN upstream lacks", "3091000000000000", creditcard.JCB, ""},
		{"top of an 8-digit JCB IIN range", "3349999900000000", creditcard.JCB, ""},
		{"a 4-digit assignment beats Elo's 6-digit 650031", "6500310000000000", creditcard.Discover, creditcard.Elo},
		{"a 4-digit assignment beats Troy's 6-digit 650052", "6500520000000000", creditcard.Discover, creditcard.Troy},
		{"numbers outside every assignment use the package table", "4242424242424242", creditcard.Visa, creditcard.Visa},
		{"the package table's own JCB range is untouched", "3528000000000007", creditcard.JCB, creditcard.JCB},
		{"still unknown outside both", "1112424242", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, c := range []struct {
				label string
				table *creditcard.Table
				want  creditcard.Brand
			}{{"table", table, tc.want}, {"Default", creditcard.Default(), tc.upstream}} {
				got, err := c.table.Identify(tc.number)
				if c.want == "" {
					if !errors.Is(err, creditcard.ErrUnknownBrand) {
						t.Errorf("%s: Identify() = (%q, %v), want ErrUnknownBrand", c.label, got, err)
					}

					continue
				}

				if err != nil || got != c.want {
					t.Errorf("%s: Identify() = (%q, %v), want %q", c.label, got, err, c.want)
				}
			}
		})
	}
}

// An assigned range a number has not reached yet is one more candidate: the
// number is decided only once no other brand could still claim it.
func TestIdentifyWhileTyping(t *testing.T) {
	table := mustTable(t, creditcard.Assign(creditcard.JCB, creditcard.Range("30010000", "30019999")))

	tests := []struct {
		name   string
		number string
		want   creditcard.Brand // "" means more digits are needed
	}{
		{"Diners Club 300-305 and the assignment could both claim it", "3001", ""},
		{"seven digits are still short of the 8-digit assignment", "3001999", ""},
		{"once the number reaches the assignment, it wins", "30010000", creditcard.JCB},
		{"outside the assignment, Diners Club is all that is left", "3002", creditcard.DinersClub},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := table.Identify(tc.number)
			if tc.want == "" {
				if !errors.Is(err, creditcard.ErrUnknownBrand) || !strings.Contains(err.Error(), "more digits") {
					t.Errorf("Identify(%q) = (%q, %v), want more digits needed", tc.number, got, err)
				}

				return
			}

			if err != nil || got != tc.want {
				t.Errorf("Identify(%q) = (%q, %v), want %q", tc.number, got, err, tc.want)
			}
		})
	}

	// Where nothing else could match, an assignment the number has not reached
	// yet is the only candidate, and decides it.
	table = mustTable(t, creditcard.Assign(creditcard.Discover, creditcard.Range("31120000", "31209999")))

	for number, want := range map[string]creditcard.Brand{"31": creditcard.Discover, "3112": creditcard.Discover} {
		if got, err := table.Identify(number); err != nil || got != want {
			t.Errorf("Identify(%q) = (%q, %v), want %q", number, got, err, want)
		}

		if got, err := creditcard.Default().Identify(number); err != creditcard.ErrUnknownBrand {
			t.Errorf("Default: Identify(%q) = (%q, %v), want ErrUnknownBrand", number, got, err)
		}
	}

	// "3" could still be that assignment, American Express, Diners Club or JCB.
	if got, err := table.Identify("3"); !errors.Is(err, creditcard.ErrUnknownBrand) || got != "" {
		t.Errorf(`Identify("3") = (%q, %v), want ErrUnknownBrand`, got, err)
	}
}

func TestIdentifyPrefersTheLongestAssignment(t *testing.T) {
	// Nested assignments at different precisions are not a conflict: the more
	// specific one wins, as in the package table.
	table := mustTable(t,
		creditcard.Assign(creditcard.JCB, creditcard.Prefix("35")),
		creditcard.Assign(creditcard.Discover, creditcard.Prefix("3560")),
	)

	for number, want := range map[string]creditcard.Brand{
		"3560000000000000": creditcard.Discover,
		"3570000000000000": creditcard.JCB,
		"3560":             creditcard.Discover,
		// Discover's 3560 is out of the running once the third digit is 7.
		"357": creditcard.JCB,
		// Both assignments still could, and reaching JCB's 35 settles nothing
		// while Discover's 3560 is still ahead.
		"35":  "",
		"356": "",
	} {
		got, err := table.Identify(number)
		if want == "" {
			if !errors.Is(err, creditcard.ErrUnknownBrand) || !strings.Contains(err.Error(), "more digits") {
				t.Errorf("Identify(%q) = (%q, %v), want more digits needed", number, got, err)
			}

			continue
		}

		if err != nil || got != want {
			t.Errorf("Identify(%q) = (%q, %v), want %q", number, got, err, want)
		}
	}
}

func TestIdentifyDoesNotAllocate(t *testing.T) {
	table := mustTable(t, creditcard.Assign(creditcard.Discover, creditcard.Range("6440", "6599")))

	for _, number := range []string{"6011111111111117", "6500310000000005", "4242424242424242", "35"} {
		for name, identify := range map[string]func(){
			"Default()": func() { _, _ = creditcard.Default().Identify(number) },
			"NewTable":  func() { _, _ = table.Identify(number) },
		} {
			if allocs := testing.AllocsPerRun(100, identify); allocs != 0 {
				t.Errorf("%s: Identify(%q) made %v allocations, want 0", name, number, allocs)
			}
		}
	}
}

func TestNewTableRejectsBadAssignments(t *testing.T) {
	tests := []struct {
		name        string
		assignments []creditcard.Assignment
		want        error
	}{
		{"unknown brand", []creditcard.Assignment{creditcard.Assign("jcb-overseas", creditcard.Prefix("356"))}, creditcard.ErrUnknownBrand},
		{"zero Assignment", []creditcard.Assignment{{}}, creditcard.ErrUnknownBrand},
		{"no ranges", []creditcard.Assignment{creditcard.Assign(creditcard.JCB)}, creditcard.ErrInvalidBINRange},
		{"empty prefix", []creditcard.Assignment{creditcard.Assign(creditcard.JCB, creditcard.Prefix(""))}, creditcard.ErrInvalidBINRange},
		{"non-digit", []creditcard.Assignment{creditcard.Assign(creditcard.JCB, creditcard.Range("35x8", "3589"))}, creditcard.ErrInvalidBINRange},
		{"different lengths", []creditcard.Assignment{creditcard.Assign(creditcard.JCB, creditcard.Range("3091", "30949999"))}, creditcard.ErrInvalidBINRange},
		{"too long", []creditcard.Assignment{creditcard.Assign(creditcard.JCB, creditcard.Prefix("309100000"))}, creditcard.ErrInvalidBINRange},
		{"low above high", []creditcard.Assignment{creditcard.Assign(creditcard.JCB, creditcard.Range("3589", "3528"))}, creditcard.ErrInvalidBINRange},
		{
			"same-precision overlap with another brand",
			[]creditcard.Assignment{
				creditcard.Assign(creditcard.JCB, creditcard.Range("6440", "6599")),
				creditcard.Assign(creditcard.Discover, creditcard.Range("6500", "6510")),
			},
			creditcard.ErrConflictingAssignment,
		},
		{
			"a later bad range fails the whole assignment",
			[]creditcard.Assignment{creditcard.Assign(creditcard.JCB, creditcard.Prefix("3091"), creditcard.Range("3099", "3091"))},
			creditcard.ErrInvalidBINRange,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			table, err := creditcard.NewTable(tc.assignments...)
			if !errors.Is(err, tc.want) {
				t.Fatalf("NewTable() = (%v, %v), want errors.Is(_, %v)", table, err, tc.want)
			}

			if table != nil {
				t.Errorf("NewTable() returned %v alongside an error, want nil", table)
			}
		})
	}
}

func TestNewTableAllowsOverlapsWithinOneBrand(t *testing.T) {
	// Overlapping or repeated ranges for the same brand are redundant, not
	// ambiguous, and nested ranges for different brands are resolved by length.
	mustTable(t,
		creditcard.Assign(creditcard.JCB, creditcard.Range("3528", "3589"), creditcard.Prefix("356")),
		creditcard.Assign(creditcard.JCB, creditcard.Range("3550", "3560")),
		creditcard.Assign(creditcard.Discover, creditcard.Prefix("35600")),
	)
}

func TestAssignCopiesItsRanges(t *testing.T) {
	ranges := []creditcard.BINRange{creditcard.Prefix("3091")}
	a := creditcard.Assign(creditcard.JCB, ranges...)

	// Changing the caller's slice after Assign must not change the assignment.
	ranges[0] = creditcard.Prefix("9999")

	if got, err := mustTable(t, a).Identify("3091000000000000"); err != nil || got != creditcard.JCB {
		t.Errorf("Identify() = (%q, %v), want %q", got, err, creditcard.JCB)
	}
}

func TestBINRangeString(t *testing.T) {
	tests := []struct {
		r    creditcard.BINRange
		want string
	}{
		{creditcard.Prefix("356"), "356"},
		{creditcard.Range("3528", "3589"), "3528-3589"},
		{creditcard.Range("6011", "6011"), "6011"},
	}

	for _, tc := range tests {
		if got := tc.r.String(); got != tc.want {
			t.Errorf("%#v.String() = %q, want %q", tc.r, got, tc.want)
		}
	}
}

func TestNewTableErrorsSayWhichRange(t *testing.T) {
	tests := []struct {
		name        string
		assignments []creditcard.Assignment
		want        string
	}{
		{
			// A malformed range is named by position: it might be a card
			// number pasted in by mistake.
			"malformed range",
			[]creditcard.Assignment{
				creditcard.Assign(creditcard.Visa, creditcard.Prefix("4")),
				creditcard.Assign(creditcard.JCB, creditcard.Prefix("3091"), creditcard.Range("3589", "3528")),
			},
			"invalid BIN range: assignment 2 (jcb), range 2: low end is above high end",
		},
		{
			"no ranges",
			[]creditcard.Assignment{creditcard.Assign(creditcard.JCB)},
			"invalid BIN range: assignment 1 (jcb): no ranges",
		},
		{
			"unknown brand",
			[]creditcard.Assignment{creditcard.Assign("jcb-overseas", creditcard.Prefix("356"))},
			`unknown card brand: assignment 1: "jcb-overseas"`,
		},
		{
			// Ranges that passed validation are at most eight digits, so
			// they are named the way the caller wrote them.
			"overlap",
			[]creditcard.Assignment{
				creditcard.Assign(creditcard.JCB, creditcard.Range("6440", "6599")),
				creditcard.Assign(creditcard.Discover, creditcard.Range("6500", "6510")),
			},
			"BIN range assigned to two brands: assignment 2 (discover), range 1: 6500-6510 overlaps 6440-6599 for jcb",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := creditcard.NewTable(tc.assignments...); err == nil || err.Error() != tc.want {
				t.Errorf("NewTable() error = %v, want %q", err, tc.want)
			}
		})
	}
}

// now is the clock the validation tests run against.
var now = time.Date(2026, time.October, 23, 12, 0, 0, 0, time.UTC)

func TestValidate(t *testing.T) {
	valid := creditcard.Card{Number: "4556974850403706", CVV: "123", ExpiryMonth: "12", ExpiryYear: "2030"}

	with := func(change func(*creditcard.Card)) creditcard.Card {
		c := valid
		change(&c)

		return c
	}

	tests := []struct {
		name      string
		card      creditcard.Card
		wantBrand creditcard.Brand
		want      error
	}{
		{"valid", valid, creditcard.Visa, nil},
		{"two-digit year", with(func(c *creditcard.Card) { c.ExpiryYear = "30" }), creditcard.Visa, nil},
		{"not digits", with(func(c *creditcard.Card) { c.Number = "4556 9748 5040 3706" }), "", creditcard.ErrInvalidNumber},
		{"unknown brand", with(func(c *creditcard.Card) { c.Number = withCheckDigit("111111111111111") }), "", creditcard.ErrUnknownBrand},
		{"fails Luhn", with(func(c *creditcard.Card) { c.Number = "4556974850403707" }), creditcard.Visa, creditcard.ErrInvalidNumber},
		{"wrong length", with(func(c *creditcard.Card) { c.Number = padded("4", 17) }), creditcard.Visa, creditcard.ErrInvalidNumber},
		{"wrong CVV", with(func(c *creditcard.Card) { c.CVV = "1234" }), creditcard.Visa, creditcard.ErrInvalidCVV},
		{"expired", with(func(c *creditcard.Card) { c.ExpiryMonth, c.ExpiryYear = "09", "2026" }), creditcard.Visa, creditcard.ErrExpired},
		{"bad month", with(func(c *creditcard.Card) { c.ExpiryMonth = "13" }), creditcard.Visa, creditcard.ErrInvalidMonth},
		{"bad year", with(func(c *creditcard.Card) { c.ExpiryYear = "203" }), creditcard.Visa, creditcard.ErrInvalidYear},
		{
			// 17 digits where American Express issues 15, and a 3-digit CID:
			// the number is checked first.
			"number before security code",
			creditcard.Card{Number: "34000000000000009", CVV: "123", ExpiryMonth: "12", ExpiryYear: "2030"},
			creditcard.AmericanExpress, creditcard.ErrInvalidNumber,
		},
		{"security code before expiry", with(func(c *creditcard.Card) { c.CVV, c.ExpiryYear = "1", "2001" }), creditcard.Visa, creditcard.ErrInvalidCVV},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := creditcard.Default().Validate(tc.card, now)
			if got != tc.wantBrand || !errors.Is(err, tc.want) {
				t.Errorf("Validate() = (%q, %v), want (%q, errors.Is(_, %v))", got, err, tc.wantBrand, tc.want)
			}
		})
	}

	t.Run("nil table", func(t *testing.T) {
		var table *creditcard.Table

		if got, err := table.Validate(valid, now); got != creditcard.Visa || err != nil {
			t.Errorf("Validate() = (%q, %v), want (%q, nil)", got, err, creditcard.Visa)
		}
	})
}

// The reason Validate lives on Table: the brand that identifies a number is the
// brand whose rules check it.
func TestValidateAppliesTheIdentifiedBrandsRules(t *testing.T) {
	table := mustTable(t,
		creditcard.Assign(creditcard.JCB, creditcard.Range("30910000", "30949999")),
		creditcard.Assign(creditcard.Discover, creditcard.Range("6440", "6599")),
		// American Express issues only 15 digits; an assignment does not change that.
		creditcard.Assign(creditcard.AmericanExpress, creditcard.Prefix("4999")),
	)

	tests := []struct {
		name      string
		card      creditcard.Card
		wantBrand creditcard.Brand
		want      error
		// What Default makes of the same card.
		upstreamBrand creditcard.Brand
		upstream      error
	}{
		{
			"JCB rules for an 8-digit JCB IIN: a 4-digit code is wrong",
			creditcard.Card{Number: padded("30910000", 16), CVV: "1234", ExpiryMonth: "12", ExpiryYear: "2030"},
			creditcard.JCB, creditcard.ErrInvalidCVV,
			"", creditcard.ErrUnknownBrand,
		},
		{
			"JCB rules for an 8-digit JCB IIN: 13 digits are too few",
			creditcard.Card{Number: padded("30910000", 13), CVV: "123", ExpiryMonth: "12", ExpiryYear: "2030"},
			creditcard.JCB, creditcard.ErrInvalidNumber,
			"", creditcard.ErrUnknownBrand,
		},
		{
			"Discover rules for 650031: 19 digits are fine",
			creditcard.Card{Number: padded("650031", 19), CVV: "123", ExpiryMonth: "12", ExpiryYear: "2030"},
			creditcard.Discover, nil,
			creditcard.Elo, creditcard.ErrInvalidNumber,
		},
		{
			"American Express rules for an assigned 4999: 16 digits are too many",
			creditcard.Card{Number: padded("4999", 16), CVV: "1234", ExpiryMonth: "12", ExpiryYear: "2030"},
			creditcard.AmericanExpress, creditcard.ErrInvalidNumber,
			creditcard.Visa, creditcard.ErrInvalidCVV,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := table.Validate(tc.card, now)
			if got != tc.wantBrand || !errors.Is(err, tc.want) {
				t.Errorf("table: Validate() = (%q, %v), want (%q, errors.Is(_, %v))", got, err, tc.wantBrand, tc.want)
			}

			got, err = creditcard.Default().Validate(tc.card, now)
			if got != tc.upstreamBrand || !errors.Is(err, tc.upstream) {
				t.Errorf("Default: Validate() = (%q, %v), want (%q, errors.Is(_, %v))", got, err, tc.upstreamBrand, tc.upstream)
			}
		})
	}
}
