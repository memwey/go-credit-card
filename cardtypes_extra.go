package creditcard

// Brands credit-card-type does not model. These are maintained by hand and are
// never touched by "go run ./internal/gen".
//
// Short follows upstream's kebab-case brand id convention so the two halves of
// the table read the same way.
//
// They are appended after the upstream brands, so when a pattern here is
// exactly as specific as an upstream one the upstream brand wins. Being listed
// later costs nothing when the pattern is longer: Cabal's six digit 603522
// still beats Maestro's one digit 6.
var extraBrands = []brand{
	{
		Short:    Bankcard,
		Long:     "Bankcard",
		Patterns: []pattern{prefix("5610"), between("560221", "560225")},
		Lengths:  []int{16},
		Code:     SecurityCode{"CVV", 3},
	},
	{
		Short: Cabal,
		Long:  "Cabal",
		Patterns: []pattern{
			prefix("604400"),
			prefix("627170"),
			prefix("603522"),
			prefix("589657"),
			between("604201", "604219"),
			between("604300", "604399"),
		},
		Lengths: []int{16},
		Code:    SecurityCode{"CVV", 3},
	},
	{
		Short: Dankort,
		Long:  "Dankort",
		// Written as a six digit range rather than the 5019 prefix so it
		// outranks Maestro's six digit ranges instead of losing to them. The
		// matching hole in Maestro is opened by a split in internal/gen.
		Patterns: []pattern{between("501900", "501999")},
		Lengths:  []int{16},
		Code:     SecurityCode{"CVV", 3},
	},
	{
		Short:    InterPayment,
		Long:     "InterPayment",
		Patterns: []pattern{prefix("636")},
		Lengths:  []int{16, 17, 18, 19},
		Code:     SecurityCode{"CVV", 3},
	},
	{
		Short:    InstaPayment,
		Long:     "InstaPayment",
		Patterns: []pattern{between("637", "639")},
		Lengths:  []int{16},
		Code:     SecurityCode{"CVV", 3},
	},
	{
		Short: VisaElectron,
		Long:  "Visa Electron",
		Patterns: []pattern{
			prefix("4026"),
			prefix("4405"),
			prefix("4508"),
			prefix("4844"),
			prefix("4913"),
			prefix("4917"),
			prefix("417500"),
		},
		Lengths: []int{16},
		Code:    SecurityCode{"CVV", 3},
	},
	{
		Short: Aura,
		Long:  "Aura",
		// NOTE: this two digit prefix loses to Maestro's six digit ranges,
		// which cover almost all of 50xxxx. Aura is effectively unreachable
		// until a specific Aura BIN range is supplied here.
		Patterns: []pattern{prefix("50")},
		Lengths:  []int{16},
		Code:     SecurityCode{"CVV", 3},
	},
}
