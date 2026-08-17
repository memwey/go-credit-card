package creditcard

//go:generate go run ./internal/gen

// SecurityCode describes a brand's verification code the way the issuing
// network names it. Size is the exact number of digits the network requires.
type SecurityCode struct {
	Name string
	Size int
}

// pattern matches the leading digits of a card number. With Max empty it is an
// exact prefix; otherwise it is an inclusive range compared over the first
// len(Min) digits. Both forms tolerate partial numbers, so a pattern still
// matches while the user is halfway through typing.
type pattern struct {
	Min, Max string
}

func prefix(p string) pattern { return pattern{Min: p} }

func between(lo, hi string) pattern { return pattern{Min: lo, Max: hi} }

// brand is one entry in the identification table.
type brand struct {
	Short, Long string
	Patterns    []pattern
	// Lengths are the valid total card-number lengths for the brand.
	Lengths []int
	Code    SecurityCode
}

// brands is the identification table: credit-card-type's brands in its own
// tie-break order, followed by the brands this package adds on top.
//
// Order only breaks ties between patterns of equal specificity; a longer
// matching pattern always wins regardless of position.
var brands = append(append([]brand{}, upstreamBrands...), extraBrands...)
