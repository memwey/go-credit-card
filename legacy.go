package creditcard

import (
	"strings"
)

// legacyNames maps current brand ids to the Short and Long pair this package
// reported before its table was aligned with credit-card-type. Brands missing
// from the map already carry their historical names.
var legacyNames = map[string]struct{ short, long string }{
	AmericanExpress: {LegacyAmericanExpress, "American Express"},
	UnionPay:        {LegacyUnionPay, "China UnionPay"},
	VisaElectron:    {LegacyVisaElectron, "Visa Electron"},
	// Only the display name changed for Mastercard.
	Mastercard: {Mastercard, "MasterCard"},
}

// The Diners Club sub-brands the alignment folded into a single brand.
var (
	legacyEnRoute       = Company{LegacyDinersClubEnRoute, "Diners Club enRoute"}
	legacyCarteBlanche  = Company{LegacyDinersClubCarteBlanche, "Diners Club Carte Blanche"}
	legacyInternational = Company{LegacyDinersClubInternational, "Diners Club International"}
)

// legacyAliases maps legacy brand ids that are only a different spelling of a
// brand still in the table, so SpecFor can resolve them by delegation.
var legacyAliases = map[string]string{
	LegacyAmericanExpress: AmericanExpress,
	LegacyUnionPay:        UnionPay,
	LegacyVisaElectron:    VisaElectron,
}

// legacySpecs carries the specs for the Diners Club sub-brands, which have no
// entry in the table: Carte Blanche and enRoute were folded away, and neither
// shares the merged brand's lengths.
var legacySpecs = map[string]Spec{
	LegacyDinersClubCarteBlanche:  {Lengths: []int{14}, Code: SecurityCode{"CVV", 3}},
	LegacyDinersClubEnRoute:       {Lengths: []int{15, 16}, Code: SecurityCode{"CVV", 3}},
	LegacyDinersClubInternational: {Lengths: []int{14, 16, 19}, Code: SecurityCode{"CVV", 3}},
}

// LegacyMethod is LegacyCompany's counterpart to Method: it attaches the
// legacy company to the card instead of returning it.
func (c *Card) LegacyMethod() error {
	c.Company = Company{}

	company, err := c.LegacyCompany()
	if err != nil {
		return err
	}

	c.Company = company

	return nil
}

// LegacyCompany identifies the card and reports it under the brand names this
// package used before its table was aligned with credit-card-type. It exists so
// callers written against the old names keep working; new code should use
// Method or MethodValidate, whose ids match upstream.
//
// Beyond renaming, it restores the two prefixes the alignment dropped -- Diners
// Club enRoute's 2014 and 2149, and Diners Club International's 309 -- and
// splits Diners Club back into Carte Blanche and International by number
// length, the way the old table did.
//
// It deliberately does not reproduce the old table's mistakes:
//
//   - BINs that now belong to Hiper, Troy, Verve or Mir are reported under those
//     brands. The old table lacked them, so it misfiled 637095 as InstaPayment
//     and 9792 as unknown.
//   - The old Elo ranges 650035-650033, 650901-650920 and 655021-655021 were
//     wrong. The corrected ranges apply here too.
//   - A 16 digit Diners Club number is reported as International. The old code
//     capped Diners Club at 14 digits and reported it as unknown.
func (c *Card) LegacyCompany() (Company, error) {
	if !isDigits(c.Number) {
		return Company{}, ErrUnknownMethod
	}

	// Prefixes the upstream table no longer carries. HasPrefix rather than
	// pattern matching, because the old table needed the full prefix present
	// before it would claim the number.
	if strings.HasPrefix(c.Number, "2014") || strings.HasPrefix(c.Number, "2149") {
		return legacyEnRoute, nil
	}

	if strings.HasPrefix(c.Number, "309") {
		return legacyInternational, nil
	}

	company, err := c.MethodValidate()
	if err != nil {
		return Company{}, err
	}

	if company.Short == DinersClub {
		if len(c.Number) == 14 && between("300", "305").matches(c.Number) {
			return legacyCarteBlanche, nil
		}

		return legacyInternational, nil
	}

	if name, ok := legacyNames[company.Short]; ok {
		company.Short, company.Long = name.short, name.long
	}

	return company, nil
}
