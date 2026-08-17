package main

import (
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
