package creditcard

import "slices"

// Brand ids reported in Company.Short by Method and MethodValidate. The first
// group is credit-card-type's own ids, reported verbatim; the second is the
// brands this package adds on top of upstream.
const (
	Visa            = "visa"
	Mastercard      = "mastercard"
	AmericanExpress = "american-express"
	DinersClub      = "diners-club"
	Discover        = "discover"
	Troy            = "troy"
	JCB             = "jcb"
	UnionPay        = "unionpay"
	Naranja         = "naranja"
	Verve           = "verve"
	Maestro         = "maestro"
	Elo             = "elo"
	Mir             = "mir"
	Hiper           = "hiper"
	Hipercard       = "hipercard"

	Bankcard     = "bankcard"
	Cabal        = "cabal"
	Dankort      = "dankort"
	InterPayment = "interpayment"
	InstaPayment = "instapayment"
	VisaElectron = "visa-electron"
	Aura         = "aura"
)

// Brand ids reported by LegacyCompany and LegacyMethod, for the brands whose
// names changed when the table was aligned with credit-card-type. Brands absent
// from this group keep the same id either way, so Visa and Maestro above serve
// both.
const (
	LegacyAmericanExpress         = "amex"
	LegacyUnionPay                = "china unionpay"
	LegacyVisaElectron            = "visa electron"
	LegacyDinersClubCarteBlanche  = "diners club carte blanche"
	LegacyDinersClubEnRoute       = "diners club enroute"
	LegacyDinersClubInternational = "diners club international"
)

// Brands returns every brand id Method and MethodValidate can report, in the
// order the table tests them. LegacyCompany's ids are not included; they are
// listed in the Legacy constants above.
func Brands() []string {
	out := make([]string, 0, len(brands))

	for _, b := range brands {
		out = append(out, b.Short)
	}

	return out
}

// Spec describes how a brand's card numbers are shaped: the total lengths the
// issuing network uses, and the security code it prints on the card.
type Spec struct {
	Lengths []int
	Code    SecurityCode
}

// SpecFor returns the spec for a brand id. It accepts the ids Method and
// MethodValidate report as well as the legacy ids from LegacyCompany, so
// callers on either naming can look a brand up. The second result reports
// whether the brand is known.
//
// The returned Lengths is a fresh slice; mutating it does not affect the table.
func SpecFor(brand string) (Spec, bool) {
	for i := range brands {
		if brands[i].Short == brand {
			return Spec{
				Lengths: slices.Clone(brands[i].Lengths),
				Code:    brands[i].Code,
			}, true
		}
	}

	if current, ok := legacyAliases[brand]; ok {
		return SpecFor(current)
	}

	if spec, ok := legacySpecs[brand]; ok {
		return Spec{
			Lengths: slices.Clone(spec.Lengths),
			Code:    spec.Code,
		}, true
	}

	return Spec{}, false
}
