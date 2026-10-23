package creditcard

import (
	"fmt"
	"slices"
	"time"
)

// maxBINDigits is the longest BIN or IIN ISO/IEC 7812 issues.
const maxBINDigits = 8

// BINRange is an inclusive run of leading digits, such as 3528-3589. Lo and Hi
// have the same number of digits and are compared with that many leading
// digits of a card number. A prefix is a range with Lo == Hi; Prefix builds
// one.
type BINRange struct {
	Lo, Hi string
}

// Prefix returns the BINRange of every number that starts with p.
func Prefix(p string) BINRange {
	return BINRange{Lo: p, Hi: p}
}

// Range returns the BINRange from lo to hi inclusive. Both ends must have the
// same number of digits: Range("30910000", "30949999"), not Range("3091", "30949999").
func Range(lo, hi string) BINRange {
	return BINRange{Lo: lo, Hi: hi}
}

// String renders the range the way BIN tables are usually written: "356" for a
// prefix, "3528-3589" for a range.
func (r BINRange) String() string {
	if r.Lo == r.Hi {
		return r.Lo
	}

	return r.Lo + "-" + r.Hi
}

// problem describes what is wrong with r, or returns "" for a valid range.
func (r BINRange) problem() string {
	switch {
	case !isDigits(r.Lo) || !isDigits(r.Hi):
		return "both ends must be digits"
	case len(r.Lo) > maxBINDigits || len(r.Hi) > maxBINDigits:
		return fmt.Sprintf("at most %d digits", maxBINDigits)
	case len(r.Lo) != len(r.Hi):
		return "both ends must have the same number of digits"
	case r.Lo > r.Hi:
		return "low end is above high end"
	}

	return ""
}

// matches reports whether number starts inside r. Only the digits number has
// so far are compared, so a number still being typed matches every range it
// could still land in. number must be a non-empty run of digits.
//
// Equal-length digit strings order the same way as the numbers they spell, so
// comparing the strings compares the numbers. This is credit-card-type's
// src/lib/matches.ts.
func (r BINRange) matches(number string) bool {
	n := min(len(r.Lo), len(number))

	return r.Lo[:n] <= number[:n] && number[:n] <= r.Hi[:n]
}

// overlaps reports whether r and o share a number at the same precision.
// Ranges of different lengths are nested, not conflicting: the longer one is
// the more specific and wins, as in the package table.
func (r BINRange) overlaps(o BINRange) bool {
	return len(r.Lo) == len(o.Lo) && r.Lo <= o.Hi && o.Lo <= r.Hi
}

// entry is one brand's row in a table: the ranges that identify it and, in the
// package table, the rules its cards follow. Assign makes one entry per range,
// carrying only the brand; the rules always come from the package table.
type entry struct {
	Brand   Brand
	Name    string
	Ranges  []BINRange
	Lengths []int
	Code    SecurityCode
}

// match is how an entry matches a number. As in credit-card-type's
// addMatchingCardsToResults, only the entry's first matching range counts.
type match struct {
	brand Brand
	// length is the number of digits in that range.
	length int
	// reached reports whether the number is at least that long, which is
	// what upstream records as matchStrength.
	reached bool
}

// matchOf reports how e matches number, and false when none of its ranges do.
func (e *entry) matchOf(number string) (match, bool) {
	for _, r := range e.Ranges {
		if r.matches(number) {
			return match{brand: e.Brand, length: len(r.Lo), reached: len(number) >= len(r.Lo)}, true
		}
	}

	return match{}, false
}

// errTooShort reports a number that still matches several brands without
// having reached all of their ranges.
var errTooShort = fmt.Errorf("%w: more digits needed to tell the brands apart", ErrUnknownBrand)

// decide is credit-card-type's findBestMatch over table. Once number has
// reached every entry it matches, the longest match wins, ties going to the
// earlier entry. Until then number is decided only when every entry it
// matches names the same brand; that is when upstream returns one candidate.
func decide(table []entry, number string) (Brand, error) {
	// best is a value, not a pointer to the loop's match: taking that address
	// would move every match to the heap.
	var (
		best    match
		matched bool
	)

	reached, oneBrand := true, true

	for i := range table {
		m, ok := table[i].matchOf(number)
		if !ok {
			continue
		}

		if matched && m.brand != best.brand {
			oneBrand = false
		}

		reached = reached && m.reached

		if !matched || m.length > best.length {
			best, matched = m, true
		}
	}

	switch {
	case !matched:
		return "", ErrUnknownBrand
	case reached || oneBrand:
		return best.brand, nil
	}

	return "", errTooShort
}

// Table maps BIN ranges to brands. Default returns the package table, which
// follows credit-card-type and nothing else; NewTable builds one with ranges
// of your own on top of it.
//
// A Table is immutable once built and safe for concurrent use. The zero Table
// and a nil *Table behave like Default.
type Table struct {
	// assigned holds one entry per assigned range, in the order given.
	assigned []entry
	// all is assigned followed by the package table, for numbers that have
	// not reached any assigned range yet.
	all []entry
}

// Default returns the package table: credit-card-type's brands and ranges at
// UpstreamVersion, with no assignments.
//
// Every call returns a Table of its own, so overwriting one -- even with
// *Default() = *other -- cannot change what any other code identifies. Code
// that needs other ranges builds its own Table with NewTable and uses that
// explicitly.
func Default() *Table {
	return &Table{}
}

// Assignment reports the numbers in some BIN ranges as a brand. Build one with
// Assign and pass it to NewTable.
type Assignment struct {
	brand  Brand
	ranges []BINRange
}

// Assign returns an Assignment of ranges to b, which must be one of Brands.
// Numbers in the ranges follow b's rules like any other b number.
//
// Once a number is at least as long as an assigned range, the assignments win
// over the package table no matter how specific the table's own match is:
// assigning Range("6440", "6599") to Discover takes 650031 away from Elo even
// though Elo lists that BIN with six digits. The assignments then decide the
// number among themselves by the rule Identify applies to the package table:
// with Prefix("35") assigned to JCB and Prefix("3560") to Discover, "357" is
// JCB and "3560" Discover, but "356" waits for another digit.
//
// A number that has not reached any assigned range yet is decided as Identify
// describes, with the assigned ranges it could still grow into counting as
// candidates: "3001" stays undecided while both an assigned 30010000-30019999
// and the package table's Diners Club 300-305 could claim it.
func Assign(b Brand, ranges ...BINRange) Assignment {
	return Assignment{brand: b, ranges: slices.Clone(ranges)}
}

// NewTable returns the package table with the given assignments on top. Every
// assignment is checked up front: a malformed range reports
// ErrInvalidBINRange, an unknown brand ErrUnknownBrand, and a range that
// overlaps one assigned to a different brand at the same precision
// ErrConflictingAssignment. The error names the assignment by position and
// brand, and the range by position. A range that passed validation is shown
// too; a malformed one never is, since it may be a card number pasted in by
// mistake.
//
// Nested ranges of different lengths are allowed, whichever brands they name;
// the longer one is the more specific and wins.
func NewTable(assignments ...Assignment) (*Table, error) {
	t := &Table{}

	for i, a := range assignments {
		if err := t.assign(i+1, a); err != nil {
			return nil, err
		}
	}

	t.all = slices.Concat(t.assigned, upstream)

	return t, nil
}

// assign adds the n-th assignment. A malformed range could be a card number
// pasted in by mistake, so errors name it by position and never echo it;
// ranges that passed validation are at most eight digits and are named.
func (t *Table) assign(n int, a Assignment) error {
	if lookup(a.brand) == nil {
		return fmt.Errorf("%w: assignment %d: %s", ErrUnknownBrand, n, brandLabel(a.brand))
	}

	if len(a.ranges) == 0 {
		return fmt.Errorf("%w: assignment %d (%s): no ranges", ErrInvalidBINRange, n, a.brand)
	}

	for i, r := range a.ranges {
		if problem := r.problem(); problem != "" {
			return fmt.Errorf("%w: assignment %d (%s), range %d: %s", ErrInvalidBINRange, n, a.brand, i+1, problem)
		}

		for _, prev := range t.assigned {
			if p := prev.Ranges[0]; prev.Brand != a.brand && p.overlaps(r) {
				return fmt.Errorf("%w: assignment %d (%s), range %d: %s overlaps %s for %s",
					ErrConflictingAssignment, n, a.brand, i+1, r, p, prev.Brand)
			}
		}

		t.assigned = append(t.assigned, entry{Brand: a.brand, Ranges: []BINRange{r}})
	}

	return nil
}

// Identify reports which brand number belongs to. number must be digits only;
// strip spaces and dashes first.
//
// It follows credit-card-type exactly. Each brand is matched by the first of
// its ranges that number matches, and number "reaches" that range once it is
// at least as long. When number has reached every brand it matches, the
// longest range wins, ties going to the brand listed first. Assignments come
// before that, as Assign describes. Lengths play no part: a number too long
// for its brand is still that brand, and Brand.ValidateNumber reports the
// length.
//
// A full card number reaches every range it matches, so it is always decided.
// A partial one, as while the user types, may not be: "6" could still become
// Discover, Maestro, UnionPay and more. It is decided only when every brand it
// matches is the same one, as "35" is JCB; otherwise Identify reports
// ErrUnknownBrand, noting that more digits are needed.
//
// Identify returns ErrInvalidNumber when number is not all digits, and
// ErrUnknownBrand when no brand matches or more digits are needed.
func (t *Table) Identify(number string) (Brand, error) {
	if !isDigits(number) {
		return "", fmt.Errorf("%w: not all digits", ErrInvalidNumber)
	}

	var assigned, all []entry
	if t != nil {
		assigned, all = t.assigned, t.all
	}

	if all == nil { // the nil and the zero Table
		all = upstream
	}

	// Once number reaches an assigned range, the assignments alone decide it.
	for i := range assigned {
		if m, ok := assigned[i].matchOf(number); ok && m.reached {
			return decide(assigned, number)
		}
	}

	return decide(all, number)
}

// Card is a card's details as the cardholder entered them. Table.Validate is
// the only thing that reads them; nothing here changes them.
type Card struct {
	Number string
	CVV    string
	// ExpiryMonth is 1-12, with or without a leading zero.
	ExpiryMonth string
	// ExpiryYear has two or four digits.
	ExpiryYear string
}

// Validate checks c in one pass: it identifies c.Number once, checks the
// number and security code by that brand's rules, then checks the expiry date
// against now as ValidateExpiry does.
//
// The brand is returned whenever identification succeeded, even alongside a
// validation error, so a caller can say which brand's rule the card broke.
// Errors are those of Identify, Brand.ValidateNumber, Brand.ValidateCVV and
// ValidateExpiry, in that order.
func (t *Table) Validate(c Card, now time.Time) (Brand, error) {
	b, err := t.Identify(c.Number)
	if err != nil {
		return "", err
	}

	if err := b.ValidateNumber(c.Number); err != nil {
		return b, err
	}

	if err := b.ValidateCVV(c.CVV); err != nil {
		return b, err
	}

	return b, ValidateExpiry(c.ExpiryMonth, c.ExpiryYear, now)
}
