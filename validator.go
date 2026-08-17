package creditcard

import (
	"slices"
	"strconv"
)

// matches reports whether the leading digits of number satisfy p. A number
// shorter than the pattern still matches as long as the digits present agree,
// which is what makes type-as-you-go identification work.
//
// The algorithm follows credit-card-type's src/lib/matches.ts.
func (p pattern) matches(number string) bool {
	if number == "" {
		return false
	}

	n := len(p.Min)
	if len(number) < n {
		n = len(number)
	}

	if p.Max == "" {
		return p.Min[:n] == number[:n]
	}

	got, err := strconv.Atoi(number[:n])
	if err != nil {
		return false
	}

	lo, err := strconv.Atoi(p.Min[:n])
	if err != nil {
		return false
	}

	hi, err := strconv.Atoi(p.Max[:n])
	if err != nil {
		return false
	}

	return got >= lo && got <= hi
}

// allowsLength reports whether number's length is compatible with the brand.
// A number still being typed is compatible with anything up to the brand's
// longest form; a number past that length can no longer belong to the brand.
func (b *brand) allowsLength(length int) bool {
	if len(b.Lengths) == 0 {
		return true
	}

	return length <= slices.Max(b.Lengths)
}

// matchStrength returns the length of the longest pattern in the brand that
// number matches, and whether the brand matched at all. Strength is only
// counted once number is at least as long as the pattern, mirroring
// credit-card-type's matchStrength: until then the match is not yet conclusive.
func (b *brand) matchStrength(number string) (int, bool) {
	if !b.allowsLength(len(number)) {
		return 0, false
	}

	return b.prefixMatchStrength(number)
}

// prefixMatchStrength is matchStrength without the type-as-you-go maximum
// length cutoff. Final validation needs this form: a number that starts with a
// known brand does not become an unknown brand merely because it is too long.
func (b *brand) prefixMatchStrength(number string) (int, bool) {
	strength, matched := 0, false

	for _, p := range b.Patterns {
		if !p.matches(number) {
			continue
		}

		matched = true

		if len(number) >= len(p.Min) && len(p.Min) > strength {
			strength = len(p.Min)
		}
	}

	return strength, matched
}

// bestPrefixMatch identifies a brand from its patterns without discarding it
// when number exceeds the brand's maximum length. Validation uses this helper
// so an overlong known-brand number is checked against that brand's lengths
// instead of falling back to the generic unknown-brand rule.
func bestPrefixMatch(number string) *brand {
	var best *brand
	bestStrength := -1

	for i := range brands {
		strength, matched := brands[i].prefixMatchStrength(number)
		if !matched || strength <= bestStrength {
			continue
		}

		best = &brands[i]
		bestStrength = strength
	}

	return best
}

// bestMatch returns the brand whose longest matching pattern is the most
// specific. Ties are broken by the order of the brands table.
//
// This replaces the fixed switch-case precedence the package used before, so a
// specific six- or eight-digit BIN now always beats a broad two-digit prefix
// regardless of which brand happens to be checked first.
func bestMatch(number string) *brand {
	var best *brand
	bestStrength := -1

	for i := range brands {
		strength, matched := brands[i].matchStrength(number)
		if !matched || strength <= bestStrength {
			continue
		}

		best = &brands[i]
		bestStrength = strength
	}

	return best
}

// isDigits reports whether s is a non-empty run of ASCII digits.
func isDigits(s string) bool {
	if s == "" {
		return false
	}

	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}

	return true
}
