package creditcard

import (
	"slices"
	"strconv"
	"testing"
	"time"
)

// currentMonthYear returns a month and year that have not expired yet.
func currentMonthYear() (month, year string) {
	now := time.Now().UTC()

	return strconv.Itoa(int(now.Month())), strconv.Itoa(now.Year())
}

func TestLastFour(t *testing.T) {
	tests := []struct {
		name    string
		number  string
		want    string
		wantErr bool
	}{
		{"sixteen digits", "4012888888881881", "1881", false},
		{"exactly four digits", "1881", "1881", false},
		{"fewer than four digits", "123", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			card := Card{Number: tc.number}

			got, err := card.LastFour()
			if (err != nil) != tc.wantErr {
				t.Fatalf("LastFour() error = %v, wantErr %v", err, tc.wantErr)
			}

			if got != tc.want {
				t.Errorf("LastFour() = %q, want %q", got, tc.want)
			}

			// LastFourDigits is an alias and must agree.
			alias, aliasErr := card.LastFourDigits()
			if alias != got || (aliasErr != nil) != (err != nil) {
				t.Errorf("LastFourDigits() = %q, %v; want %q, %v", alias, aliasErr, got, err)
			}
		})
	}
}

func TestWipe(t *testing.T) {
	card := Card{Number: "4012888888881881", Cvv: "111", Month: "02", Year: "2015"}
	card.Wipe()

	want := Card{Number: "0000000000000000", Cvv: "0000", Month: "01", Year: "1970"}
	if card != want {
		t.Errorf("Wipe() left %+v, want %+v", card, want)
	}
}

// Card and Company must stay comparable: callers compare them directly and use
// them as map keys, and a slice field anywhere inside would break that.
func TestCardIsComparable(t *testing.T) {
	a := Card{Number: "4242424242424242", Company: Company{Visa, "Visa"}}
	b := a

	if a != b {
		t.Error("a copy of a Card does not compare equal to its original")
	}

	seen := map[Card]bool{a: true}
	if !seen[b] {
		t.Error("Card cannot be used as a map key")
	}
}

func TestValidateExpiration(t *testing.T) {
	month, year := currentMonthYear()

	now := time.Now().UTC()
	nextYear := strconv.Itoa(now.Year() + 1)
	prevYear, prevMonth, _ := now.AddDate(0, -1, 0).Date()

	tests := []struct {
		name        string
		month, year string
		wantErr     bool
	}{
		{"year in the past", "02", "2001", true},
		{"month zero", "0", nextYear, true},
		{"month thirteen", "13", nextYear, true},
		{"last month", strconv.Itoa(int(prevMonth)), strconv.Itoa(prevYear), true},
		{"year is two letters", "1", "ab", true},
		{"year is three letters", "1", "abc", true},
		{"month is not a number", "abc", year, true},
		{"two digit year", month, year[2:], false},
		{"four digit year", month, year, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			card := Card{Number: "4012888888881881", Cvv: "111", Month: tc.month, Year: tc.year}

			if err := card.ValidateExpiration(); (err != nil) != tc.wantErr {
				t.Errorf("ValidateExpiration() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateExpirationMockedClock(t *testing.T) {
	defer resetMocks()

	timeNowCaller = func() time.Time {
		return time.Date(2001, 3, 1, 1, 1, 1, 1, time.UTC)
	}

	card := Card{Number: "4012888888881881", Cvv: "111", Month: "02", Year: "2001"}

	err := card.ValidateExpiration()
	if err == nil {
		t.Fatal("ValidateExpiration() = nil, want an error for a card that expired last month")
	}

	if err.Error() != "Credit card has expired" {
		t.Errorf("ValidateExpiration() = %q, want %q", err.Error(), "Credit card has expired")
	}
}

func TestValidateCVV(t *testing.T) {
	tests := []struct {
		name    string
		number  string
		cvv     string
		wantErr bool
	}{
		{"three digits on Visa", "4012888888881881", "111", false},
		{"four digits on Visa", "4012888888881881", "1111", true},
		{"four digits on American Express", "378282246310005", "1111", false},
		{"three digits on American Express", "378282246310005", "111", true},
		{"too many digits", "4012888888881881", "11111", true},
		{"too few digits", "4012888888881881", "11", true},
		{"not digits at all", "4012888888881881", "abc", true},
		// No brand match, so the generic three-or-four rule applies.
		{"unknown brand accepts three", "1234", "111", false},
		{"unknown brand accepts four", "1234", "1111", false},
		{"unknown brand rejects five", "1234", "11111", true},
		{"no number yet", "", "111", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			card := Card{Number: tc.number, Cvv: tc.cvv}

			if err := card.ValidateCVV(); (err != nil) != tc.wantErr {
				t.Errorf("ValidateCVV() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidateNumber(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   bool
	}{
		{"Visa sixteen digits", "4012888888881881", true},
		{"Visa seventeen digits", "40128888888818818", false},
		{"real Visa", "4556974850403706", true},
		{"fails Luhn", "4242424242424241", false},
		{"too short for any brand", "424832", false},
		{"not digits", "abcdefg", false},
		// No brand claims a leading zero, so the generic 13-19 rule applies.
		{"unknown brand within the generic length", "0000000000000000", true},
		{"unknown brand below the generic length", "000000000000", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			card := Card{Number: tc.number}

			if got := card.ValidateNumber(); got != tc.want {
				t.Errorf("ValidateNumber() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	month, year := currentMonthYear()

	t.Run("accepts a real card", func(t *testing.T) {
		card := Card{Number: "4556974850403706", Cvv: "111", Month: month, Year: year}

		if err := card.Validate(); err != nil {
			t.Errorf("Validate() = %v, want nil", err)
		}
	})

	t.Run("rejects an expired card before looking at anything else", func(t *testing.T) {
		card := Card{Number: "4556974850403706", Cvv: "111", Month: "02", Year: "2001"}

		if err := card.Validate(); err == nil {
			t.Error("Validate() = nil, want an error")
		}
	})

	t.Run("rejects a CVV the brand does not use", func(t *testing.T) {
		card := Card{Number: "4556974850403706", Cvv: "1111", Month: month, Year: year}

		if err := card.Validate(); err == nil {
			t.Error("Validate() = nil, want an error for a 4 digit CVV on Visa")
		}
	})

	t.Run("rejects a number that is too short", func(t *testing.T) {
		card := Card{Number: "1234", Cvv: "1111", Month: month, Year: year}

		if err := card.Validate(true); err == nil {
			t.Error("Validate(true) = nil, want an error")
		}
	})

	t.Run("rejects a length the brand does not issue", func(t *testing.T) {
		// 17 digits, but Visa only issues 16, 18 or 19.
		card := Card{Number: "40128888888818818", Cvv: "111", Month: month, Year: year}

		if err := card.Validate(true); err == nil {
			t.Error("Validate(true) = nil, want an error")
		}
	})
}

func TestValidateTestNumbers(t *testing.T) {
	month, year := currentMonthYear()

	// American Express uses a 4 digit CID, every other brand here a 3 digit code.
	cards := []struct{ number, cvv string }{
		{"4242424242424242", "111"},
		{"4012888888881881", "111"},
		{"4000056655665556", "111"},
		{"5555555555554444", "111"},
		{"5200828282828210", "111"},
		{"5105105105105100", "111"},
		{"378282246310005", "1111"},
		{"371449635398431", "1111"},
		{"6011111111111117", "111"},
		{"6011000990139424", "111"},
		{"30569309025904", "111"},
		{"38520000023237", "111"},
		{"3530111333300000", "111"},
		{"3566002020360505", "111"},
	}

	for _, tc := range cards {
		t.Run(tc.number, func(t *testing.T) {
			card := Card{Number: tc.number, Cvv: tc.cvv, Month: month, Year: year}

			if err := card.Validate(true); err != nil {
				t.Errorf("Validate(true) = %v, want nil", err)
			}

			if err := card.Validate(); err == nil {
				t.Error("Validate() = nil, want test numbers to be rejected")
			}
		})
	}
}

func TestMethod(t *testing.T) {
	tests := []struct {
		name        string
		number      string
		short, long string
	}{
		{"American Express", "3782822463", AmericanExpress, "American Express"},
		{"Bankcard 5610", "56101111111", Bankcard, "Bankcard"},
		{"Bankcard 560221-560225", "56022111111", Bankcard, "Bankcard"},
		{"Cabal", "6035227716427021", Cabal, "Cabal"},
		{"China UnionPay 62", "62111111111", UnionPay, "UnionPay"},
		{"China UnionPay 81", "810111111111", UnionPay, "UnionPay"},
		{"Dankort", "501955555", Dankort, "Dankort"},
		{"Discover", "60111111111", Discover, "Discover"},
		{"Elo", "6362970000457013", Elo, "Elo"},
		{"Hiper six digit BIN", "6370950000000005", Hiper, "Hiper"},
		// Eight digit BIN, which the old [6]int prefix buffer could not hold.
		{"Hiper eight digit BIN", "6373742300000001", Hiper, "Hiper"},
		{"Hipercard", "6062826786276634", Hipercard, "Hipercard"},
		{"InstaPayment", "6370111331111111", InstaPayment, "InstaPayment"},
		{"InterPayment", "6360111331111111", InterPayment, "InterPayment"},
		{"JCB", "353011133", JCB, "JCB"},
		{"Maestro", "501855555", Maestro, "Maestro"},
		{"Mastercard 51-55", "5425233430109903", Mastercard, "Mastercard"},
		{"Mastercard 2 series", "2223000048410010", Mastercard, "Mastercard"},
		{"Mir", "2200770000000000", Mir, "Mir"},
		{"Naranja", "5895626746595650", Naranja, "Naranja"},
		{"Troy four digit BIN", "9792000000000000", Troy, "Troy"},
		{"Troy eight digit BIN", "6508370100000000", Troy, "Troy"},
		{"Verve", "5060990000000000", Verve, "Verve"},
		{"Visa", "4242424242", Visa, "Visa"},
		{"Visa Electron", "4026424242", VisaElectron, "Visa Electron"},

		// Upstream models a single Diners Club brand, so the Carte Blanche and
		// enRoute sub-brands this package used to report are gone.
		{"Diners Club 14 digits", "30022111111111", DinersClub, "Diners Club"},
		{"Diners Club 13 digits", "3002111111111", DinersClub, "Diners Club"},
		{"Diners Club 36", "36001111111111", DinersClub, "Diners Club"},
		{"Diners Club 38", "38001111111111", DinersClub, "Diners Club"},

		// Aura is registered under the bare "50" prefix, which credit-card-type
		// covers with Maestro's six digit ranges. Maestro matches on more digits
		// and therefore wins. Aura stays in the table but is only reachable once
		// a specific Aura BIN range is added to cardtypes_extra.go.
		{"50xxxx is Maestro, not Aura", "5078608877345328", Maestro, "Maestro"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			card := Card{Number: tc.number}

			if err := card.Method(); err != nil {
				t.Fatalf("Method() = %v, want nil", err)
			}

			if card.Company.Short != tc.short {
				t.Errorf("Company.Short = %q, want %q", card.Company.Short, tc.short)
			}

			if card.Company.Long != tc.long {
				t.Errorf("Company.Long = %q, want %q", card.Company.Long, tc.long)
			}
		})
	}
}

func TestMethodPrefersTheLongestBIN(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   string
	}{
		{"Elo 401178 beats the bare Visa 4 prefix", "4011780000000000", Elo},
		{"Hiper 637095 beats the InstaPayment 637-639 range", "6370950000000005", Hiper},
		{"Elo 636297 beats the InterPayment 636 prefix", "6362970000457013", Elo},
		{"Cabal 603522 beats the Maestro 6 prefix", "6035227716427021", Cabal},
		{"Bankcard 5610 beats the Maestro 56-59 range", "56101111111", Bankcard},
		{"Dankort 5019xx beats the Maestro 500000-504174 range", "501955555", Dankort},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			card := Card{Number: tc.number}

			company, err := card.MethodValidate()
			if err != nil {
				t.Fatalf("MethodValidate() = %v, want nil", err)
			}

			if company.Short != tc.want {
				t.Errorf("Company.Short = %q, want %q", company.Short, tc.want)
			}
		})
	}
}

func TestMethodUnknown(t *testing.T) {
	tests := []struct {
		name   string
		number string
	}{
		{"unknown prefix", "1112424242"},
		{"not digits", "abcdefg"},
		{"empty", ""},
		// Dropped when the table was aligned with credit-card-type.
		{"Diners Club enRoute 2014", "20142111111111"},
		{"Diners Club International 309", "3090000000000000"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			card := Card{Number: tc.number}

			if err := card.Method(); err == nil {
				t.Fatal("Method() = nil, want an error")
			}

			if card.Company.Short != "" || card.Company.Long != "" {
				t.Errorf("Company = %+v, want it left empty", card.Company)
			}
		})
	}
}

func TestSpecFor(t *testing.T) {
	tests := []struct {
		name    string
		brand   string
		lengths []int
		code    SecurityCode
	}{
		{"American Express", AmericanExpress, []int{15}, SecurityCode{"CID", 4}},
		{"Visa", Visa, []int{16, 18, 19}, SecurityCode{"CVV", 3}},
		{"Mastercard", Mastercard, []int{16}, SecurityCode{"CVC", 3}},
		{"Elo", Elo, []int{16}, SecurityCode{"CVE", 3}},
		{"Mir", Mir, []int{16, 17, 18, 19}, SecurityCode{"CVP2", 3}},
		{"Diners Club", DinersClub, []int{14, 16, 19}, SecurityCode{"CVV", 3}},
		{"a brand outside upstream", Dankort, []int{16}, SecurityCode{"CVV", 3}},

		// Legacy ids resolve too, so callers on the old naming can still ask.
		{"legacy Amex delegates to the table", LegacyAmericanExpress, []int{15}, SecurityCode{"CID", 4}},
		{"legacy UnionPay delegates to the table", LegacyUnionPay, []int{14, 15, 16, 17, 18, 19}, SecurityCode{"CVN", 3}},
		{"legacy Visa Electron delegates to the table", LegacyVisaElectron, []int{16}, SecurityCode{"CVV", 3}},

		// Diners Club sub-brands have no table entry and carry their own spec.
		{"legacy Carte Blanche", LegacyDinersClubCarteBlanche, []int{14}, SecurityCode{"CVV", 3}},
		{"legacy enRoute", LegacyDinersClubEnRoute, []int{15, 16}, SecurityCode{"CVV", 3}},
		{"legacy International", LegacyDinersClubInternational, []int{14, 16, 19}, SecurityCode{"CVV", 3}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			spec, ok := SpecFor(tc.brand)
			if !ok {
				t.Fatalf("SpecFor(%q) reported the brand as unknown", tc.brand)
			}

			if !slices.Equal(spec.Lengths, tc.lengths) {
				t.Errorf("Lengths = %v, want %v", spec.Lengths, tc.lengths)
			}

			if spec.Code != tc.code {
				t.Errorf("Code = %+v, want %+v", spec.Code, tc.code)
			}
		})
	}

	t.Run("every brand in the table has a spec", func(t *testing.T) {
		for _, id := range Brands() {
			if _, ok := SpecFor(id); !ok {
				t.Errorf("SpecFor(%q) reported a table brand as unknown", id)
			}
		}
	})

	t.Run("unknown brand", func(t *testing.T) {
		if _, ok := SpecFor("not-a-brand"); ok {
			t.Error(`SpecFor("not-a-brand") = ok, want it reported as unknown`)
		}
	})

	t.Run("Lengths is a copy of the table's slice", func(t *testing.T) {
		spec, _ := SpecFor(AmericanExpress)
		spec.Lengths[0] = 99

		again, _ := SpecFor(AmericanExpress)
		if again.Lengths[0] != 15 {
			t.Errorf("Lengths[0] = %d after a caller mutated an earlier copy, want 15", again.Lengths[0])
		}
	})
}

// matches is the core of identification, so its edges are tested directly
// rather than only through the brand table.
func TestPatternMatches(t *testing.T) {
	tests := []struct {
		name    string
		pattern pattern
		number  string
		want    bool
	}{
		{"prefix, number still being typed", prefix("6011"), "60", true},
		{"prefix, number longer than the pattern", prefix("6011"), "6011111111111117", true},
		{"prefix, number diverges", prefix("6011"), "6012", false},
		{"range, number the same width", between("644", "649"), "645", true},
		{"range, number below the range", between("644", "649"), "643", false},
		{"range, number above the range", between("644", "649"), "650", false},
		// Both bounds are truncated to the digits present, so a partial number
		// still matches while it could still land inside the range.
		{"range, number still being typed", between("622010", "622999"), "622", true},
		{"range, partial number already outside", between("622010", "622999"), "623", false},
		{"range, number longer than the range", between("644", "649"), "6491111111111111", true},
		{"empty number", prefix("4"), "", false},
		{"empty number against a range", between("644", "649"), "", false},
		{"non-digits against a range", between("300", "305"), "30a", false},
		// Defensive: the table only ever holds digits, but a malformed pattern
		// must not be able to claim a number.
		{"malformed lower bound", pattern{Min: "3a0", Max: "305"}, "301", false},
		{"malformed upper bound", pattern{Min: "300", Max: "3b5"}, "301", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.pattern.matches(tc.number); got != tc.want {
				t.Errorf("%+v.matches(%q) = %v, want %v", tc.pattern, tc.number, got, tc.want)
			}
		})
	}
}

func TestAllowsLength(t *testing.T) {
	tests := []struct {
		name    string
		lengths []int
		length  int
		want    bool
	}{
		{"no lengths declared accepts anything", nil, 99, true},
		{"still being typed", []int{16, 19}, 4, true},
		{"at the longest form", []int{16, 19}, 19, true},
		{"between two valid lengths", []int{16, 19}, 17, true},
		{"past the longest form", []int{16, 19}, 20, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := brand{Lengths: tc.lengths}

			if got := b.allowsLength(tc.length); got != tc.want {
				t.Errorf("allowsLength(%d) with lengths %v = %v, want %v", tc.length, tc.lengths, got, tc.want)
			}
		})
	}
}

func TestUpstreamVersion(t *testing.T) {
	if UpstreamVersion == "" {
		t.Error("UpstreamVersion is empty; the generated table should record the release it came from")
	}
}

// brandConstants mirrors the exported constants in brands.go. Keeping it in
// sync is the point of the test below: a brand added upstream shows up in
// Brands() but not here, and the test fails until a constant is declared.
var brandConstants = map[string]string{
	"Visa":            Visa,
	"Mastercard":      Mastercard,
	"AmericanExpress": AmericanExpress,
	"DinersClub":      DinersClub,
	"Discover":        Discover,
	"Troy":            Troy,
	"JCB":             JCB,
	"UnionPay":        UnionPay,
	"Naranja":         Naranja,
	"Verve":           Verve,
	"Maestro":         Maestro,
	"Elo":             Elo,
	"Mir":             Mir,
	"Hiper":           Hiper,
	"Hipercard":       Hipercard,
	"Bankcard":        Bankcard,
	"Cabal":           Cabal,
	"Dankort":         Dankort,
	"InterPayment":    InterPayment,
	"InstaPayment":    InstaPayment,
	"VisaElectron":    VisaElectron,
	"Aura":            Aura,
}

func TestBrandConstants(t *testing.T) {
	inTable := make(map[string]bool, len(brands))
	for _, id := range Brands() {
		inTable[id] = true
	}

	declared := make(map[string]bool, len(brandConstants))
	for _, id := range brandConstants {
		declared[id] = true
	}

	t.Run("every constant names a brand in the table", func(t *testing.T) {
		for name, id := range brandConstants {
			if !inTable[id] {
				t.Errorf("constant %s = %q, which is not in the brand table", name, id)
			}
		}
	})

	t.Run("every brand in the table has a constant", func(t *testing.T) {
		for _, id := range Brands() {
			if !declared[id] {
				t.Errorf("brand %q has no exported constant; declare one in brands.go", id)
			}
		}
	})
}

func TestBrandsReturnsACopy(t *testing.T) {
	first := Brands()
	if len(first) != len(brands) {
		t.Fatalf("Brands() returned %d ids, want %d", len(first), len(brands))
	}

	first[0] = "clobbered"

	if got := Brands()[0]; got != Visa {
		t.Errorf("Brands()[0] = %q after a caller mutated an earlier result, want %q", got, Visa)
	}
}

// The legacy constants are a compatibility contract with callers written
// against the old table, so their values are pinned here rather than only
// compared against themselves.
func TestLegacyConstants(t *testing.T) {
	tests := []struct{ name, got, want string }{
		{"LegacyAmericanExpress", LegacyAmericanExpress, "amex"},
		{"LegacyUnionPay", LegacyUnionPay, "china unionpay"},
		{"LegacyVisaElectron", LegacyVisaElectron, "visa electron"},
		{"LegacyDinersClubCarteBlanche", LegacyDinersClubCarteBlanche, "diners club carte blanche"},
		{"LegacyDinersClubEnRoute", LegacyDinersClubEnRoute, "diners club enroute"},
		{"LegacyDinersClubInternational", LegacyDinersClubInternational, "diners club international"},
	}

	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

func TestLegacyCompany(t *testing.T) {
	tests := []struct {
		name        string
		number      string
		short, long string
	}{
		// Renamed brands come back under their old names.
		{"American Express", "3782822463", LegacyAmericanExpress, "American Express"},
		{"China UnionPay", "62111111111", LegacyUnionPay, "China UnionPay"},
		{"Mastercard keeps its id but not its display name", "5425233430109903", Mastercard, "MasterCard"},
		{"Visa Electron", "4026424242", LegacyVisaElectron, "Visa Electron"},

		// Diners Club splits back into its sub-brands.
		{"Carte Blanche needs 300-305 and exactly 14 digits", "30022111111111", LegacyDinersClubCarteBlanche, "Diners Club Carte Blanche"},
		{"13 digits is International", "3002111111111", LegacyDinersClubInternational, "Diners Club International"},
		{"36 is International", "36001111111111", LegacyDinersClubInternational, "Diners Club International"},
		{"11 digits is International", "30569309025", LegacyDinersClubInternational, "Diners Club International"},

		// Prefixes the alignment dropped come back.
		{"enRoute 2014", "20142111111111", LegacyDinersClubEnRoute, "Diners Club enRoute"},
		{"enRoute 2149", "21492111111111", LegacyDinersClubEnRoute, "Diners Club enRoute"},
		{"International 309", "3090000000000000", LegacyDinersClubInternational, "Diners Club International"},

		// Brands that were never renamed pass through untouched.
		{"Visa", "4242424242", Visa, "Visa"},
		{"Discover", "60111111111", Discover, "Discover"},
		{"JCB", "353011133", JCB, "JCB"},
		{"Cabal", "6035227716427021", Cabal, "Cabal"},
		{"Troy", "9792000000000000", Troy, "Troy"},
		{"Mir", "2200770000000000", Mir, "Mir"},

		// The old table's mistakes do not come back.
		{"637095 is Hiper, not InstaPayment", "6370950000000005", Hiper, "Hiper"},
		{"650035-650051 is Elo, which the old empty range never matched", "6500400000000000", Elo, "Elo"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			card := Card{Number: tc.number}

			company, err := card.LegacyCompany()
			if err != nil {
				t.Fatalf("LegacyCompany() = %v, want nil", err)
			}

			if company.Short != tc.short {
				t.Errorf("Company.Short = %q, want %q", company.Short, tc.short)
			}

			if company.Long != tc.long {
				t.Errorf("Company.Long = %q, want %q", company.Long, tc.long)
			}
		})
	}
}

func TestLegacyCompanyUnknown(t *testing.T) {
	tests := []struct {
		name   string
		number string
	}{
		{"unknown prefix", "1112424242"},
		{"not digits", "abcdefg"},
		// The old table needed the full prefix before it would claim enRoute.
		{"partial enRoute prefix", "20"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			card := Card{Number: tc.number}

			if _, err := card.LegacyCompany(); err == nil {
				t.Error("LegacyCompany() = nil, want an error")
			}
		})
	}
}

func TestLegacyMethod(t *testing.T) {
	card := Card{Number: "3782822463"}

	if err := card.LegacyMethod(); err != nil {
		t.Fatalf("LegacyMethod() = %v, want nil", err)
	}

	if card.Company.Short != LegacyAmericanExpress {
		t.Errorf("Company.Short = %q, want %q", card.Company.Short, LegacyAmericanExpress)
	}

	// The legacy id still resolves through SpecFor.
	spec, ok := SpecFor(card.Company.Short)
	if !ok {
		t.Fatalf("SpecFor(%q) reported the brand as unknown", card.Company.Short)
	}

	if spec.Code.Size != 4 {
		t.Errorf("SpecFor(%q).Code.Size = %d, want 4", card.Company.Short, spec.Code.Size)
	}

	t.Run("propagates errors", func(t *testing.T) {
		card := Card{Number: "1112424242"}

		if err := card.LegacyMethod(); err == nil {
			t.Error("LegacyMethod() = nil, want an error")
		}
	})
}
