package main

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
)

const validCardSource = `{
  type: "visa",
  niceType: "Visa",
  patterns: [4, [40, 49]],
  lengths: [16, 18, 19],
  code: { name: "CVV", size: 3 },
} as BuiltInCreditCardType`

func TestParseCardsRequiresValidationFields(t *testing.T) {
	tests := []struct {
		name        string
		old, new    string
		wantMessage string
	}{
		{"nice type", `niceType: "Visa",`, "", "no niceType field"},
		{"lengths", "lengths: [16, 18, 19],", "", "no lengths field"},
		{"nonempty lengths", "lengths: [16, 18, 19],", "lengths: [],", "empty lengths field"},
		{"security-code name", `name: "CVV",`, "", "no security-code name field"},
		{"security-code size", "size: 3", "removed: 3", "no security-code size field"},
		{"positive security-code size", "size: 3", "size: 0", "invalid security-code size"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			source := strings.Replace(validCardSource, tc.old, tc.new, 1)

			_, err := parseCards(source)
			if err == nil {
				t.Fatal("parseCards() = nil error, want malformed upstream data rejected")
			}

			if !strings.Contains(err.Error(), tc.wantMessage) {
				t.Errorf("parseCards() error = %q, want it to contain %q", err, tc.wantMessage)
			}
		})
	}
}

func TestParseCardsPreservesRequiredFields(t *testing.T) {
	cards, err := parseCards(validCardSource)
	if err != nil {
		t.Fatalf("parseCards() = %v, want nil", err)
	}

	if len(cards) != 1 {
		t.Fatalf("len(cards) = %d, want 1", len(cards))
	}

	card := cards[0]
	if card.typ != "visa" || card.niceType != "Visa" {
		t.Errorf("card identity = %q/%q, want visa/Visa", card.typ, card.niceType)
	}

	if len(card.lengths) != 3 || card.codeName != "CVV" || card.codeSize != 3 {
		t.Errorf("validation fields = lengths %v, code %q/%d", card.lengths, card.codeName, card.codeSize)
	}
}

func TestSortByTestOrderRejectsDuplicateDefinitions(t *testing.T) {
	_, err := sortByTestOrder([]card{{typ: "visa"}, {typ: "visa"}}, []string{"visa"})
	if err == nil {
		t.Fatal("sortByTestOrder() = nil error, want duplicate type rejected")
	}
}

func TestParsePatternListRejectsRangesThatCannotMatch(t *testing.T) {
	tests := []struct {
		name, src, wantMessage string
	}{
		{"backwards range", "[3589, 3528]", "runs backwards"},
		{"mismatched widths", "[3091, 30949999]", "mismatched widths"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parsePatternList(tc.src)
			if err == nil || !strings.Contains(err.Error(), tc.wantMessage) {
				t.Errorf("parsePatternList(%q) = %v, want an error containing %q", tc.src, err, tc.wantMessage)
			}
		})
	}

	// Upstream's data is copied as it is: no limit on how long a range is.
	got, err := parsePatternList("4, [51, 55], 30910000, 309100000")
	if err != nil || len(got) != 4 || got[1] != (pattern{"51", "55"}) {
		t.Errorf("parsePatternList() = (%v, %v), want four patterns", got, err)
	}
}

func TestParseCopyright(t *testing.T) {
	const mit = "Copyright (c) 2009-2018 Braintree, a division of PayPal, Inc.\n" +
		"Copyright (c) 2030 Someone Else\n\n" +
		"Permission is hereby granted, free of charge, to any person obtaining a copy\n"

	got, err := parseCopyright(mit)
	want := []string{"Copyright (c) 2009-2018 Braintree, a division of PayPal, Inc.", "Copyright (c) 2030 Someone Else"}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("parseCopyright() = (%q, %v), want %q", got, err, want)
	}

	if _, err := parseCopyright("Apache License\nVersion 2.0, January 2004\n"); err == nil {
		t.Error("parseCopyright() accepted a license that is not MIT")
	}

	if _, err := parseCopyright("Permission is hereby granted, free of charge, to any person\n"); err == nil {
		t.Error("parseCopyright() accepted an MIT license without a copyright line")
	}
}

func TestRenderCarriesTheNotice(t *testing.T) {
	cards := []card{{typ: "visa", niceType: "Visa", patterns: []pattern{{min: "4"}}, lengths: []int{16}, codeName: "CVV", codeSize: 3}}

	src, err := render("10.9.9", []string{"Copyright (c) 2009 A", "Copyright (c) 2030 B"}, "LICENSE-x", cards)
	if err != nil {
		t.Fatalf("render() = %v, want nil", err)
	}

	for _, want := range []string{
		"// Source: https://github.com/braintree/credit-card-type v10.9.9\n",
		"// Copyright (c) 2009 A\n// Copyright (c) 2030 B\n",
		"// MIT licensed; see LICENSE-x.\n",
		`const UpstreamVersion = "10.9.9"`,
		`Prefix("4")`,
	} {
		if !strings.Contains(string(src), want) {
			t.Errorf("render() output lacks %q:\n%s", want, src)
		}
	}
}

// The committed table must be the one a plain "go generate" would produce.
func TestDefaultVersionMatchesTheCommittedTable(t *testing.T) {
	src, err := os.ReadFile("../../upstream.go")
	if err != nil {
		t.Fatalf("read upstream.go: %v", err)
	}

	if want := fmt.Sprintf("const UpstreamVersion = %q", defaultVersion); !strings.Contains(string(src), want) {
		t.Errorf("upstream.go lacks %s: bump defaultVersion in main.go along with the table", want)
	}
}
