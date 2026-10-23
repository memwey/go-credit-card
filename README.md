# go-credit-card

[![Go Reference](https://pkg.go.dev/badge/github.com/memwey/go-credit-card.svg)](https://pkg.go.dev/github.com/memwey/go-credit-card)

Payment card brand identification and validation for Go.

- Brands are identified from BIN data generated from Braintree's
  [credit-card-type](https://github.com/braintree/credit-card-type), and the
  package table follows it exactly.
- Where your own BIN data disagrees, you layer ranges on top of that table
  instead of forking it.
- Numbers, security codes and expiry dates are checked by the identified
  brand's own rules, together with the
  [Luhn algorithm](https://en.wikipedia.org/wiki/Luhn_algorithm).

No dependencies: `go.mod` declares none, and the tests use the standard
library only.

> **v1.2.0 is a rewrite and breaks the v1.1 API.** See
> [Upgrading from v1.1](#upgrading-from-v11).

## Installation

```bash
go get github.com/memwey/go-credit-card
```

## Quick start

```go
card := creditcard.Card{Number: "4242424242424242", CVV: "123", ExpiryMonth: "02", ExpiryYear: "30"}

// Identify the brand and check the card by that brand's rules, in one pass.
brand, err := creditcard.Default().Validate(card, time.Now())
// brand == creditcard.Visa, err == nil

// Or one step at a time.
brand, err = creditcard.Default().Identify(card.Number) // creditcard.Visa
err = brand.ValidateNumber(card.Number)                 // 16, 18 or 19 digits, Luhn
err = brand.ValidateCVV(card.CVV)                       // exactly 3 digits
err = creditcard.ValidateExpiry(card.ExpiryMonth, card.ExpiryYear, time.Now())

// What do this brand's cards look like?
brand.Name()         // "Visa"
brand.Lengths()      // []int{16, 18, 19}
brand.SecurityCode() // SecurityCode{Name: "CVV", Size: 3}
```

Card numbers are taken exactly as given — digits only. Strip spaces and dashes
before calling; the package does not normalise anything, and it never puts a
card number or security code into an error message.

## How it fits together

Two questions are kept apart, because only one of them depends on BIN data:

| Question | Answered by | Depends on |
| --- | --- | --- |
| Which brand is this number? | `Table.Identify` | a BIN table |
| What do that brand's cards look like? | `Brand.ValidateNumber`, `Brand.ValidateCVV`, `Brand.Lengths`, `Brand.SecurityCode` | the brand alone |

`Table.Validate` identifies the number once and hands that brand to the rules,
so a card can never be identified by one table and checked by another.
`ValidateExpiry` and `Luhn` depend on neither and are plain functions.

`Default()` returns the package table. Every call returns a `Table` of its
own, so overwriting one cannot change what any other code identifies. There are
deliberately no package-level shortcuts such as `creditcard.Identify`: every
identification names its table.

## Assigning your own BIN ranges

When your BIN data disagrees with upstream — a card network's published IIN
list, say — build a `Table` with the ranges you need:

```go
table, err := creditcard.NewTable(
	// Eight-digit JCB IINs upstream does not know yet.
	creditcard.Assign(creditcard.JCB, creditcard.Range("30910000", "30949999")),
	// A network that accepts the whole 6440-6599 block as Discover.
	creditcard.Assign(creditcard.Discover, creditcard.Range("6440", "6599")),
)

card := creditcard.Card{Number: "6500310000000005", CVV: "123", ExpiryMonth: "12", ExpiryYear: "30"}

brand, err := table.Identify(card.Number)     // discover; Default() says elo
brand, err = table.Validate(card, time.Now()) // discover, nil: checked by Discover's rules
```

- **Assignments win outright.** Once a number is at least as long as an
  assigned range, the assignments win over the package table however specific
  the table's own match is — the four-digit `6440-6599` above beats Elo's
  six-digit `650031`. Among themselves they follow the package table's rule:
  nested ranges of different brands are decided by the longer one once the
  number reaches it, and wait for more digits until then.
- **Short numbers wait.** An assigned range a number has not reached yet is
  one more candidate: `"3001"` stays undecided while both an assigned
  `30010000-30019999` and Diners Club's `300-305` could claim it, and is decided
  once only one brand can.
- **Only existing brands.** `Assign` targets one of `Brands()`. Its numbers
  follow that brand's rules, because the rules belong to the brand.
- **Checked up front.** `NewTable` rejects malformed ranges
  (`ErrInvalidBINRange`), unknown brands (`ErrUnknownBrand`) and a range that
  overlaps one assigned to another brand at the same precision
  (`ErrConflictingAssignment`). Nested ranges of different lengths are allowed;
  the longer one is the more specific.

There is deliberately no way to *remove* a range or a brand. Which cards you
accept is a policy of your system, not a property of the brand: decide it after
identification, in your own code.

A `Table` is immutable and safe for concurrent use. The zero `Table` and a nil
`*Table` behave like `Default()`.

## Errors

Every error matches a sentinel through `errors.Is`:

| Sentinel | Reported when |
| --- | --- |
| `ErrInvalidNumber` | the number is not all digits, has a length its brand does not issue, or fails the Luhn check |
| `ErrUnknownBrand` | no brand matches the number, a partial number could still be several brands, or a `Brand` is not in `Brands()` |
| `ErrInvalidCVV` | the security code does not fit the brand's |
| `ErrInvalidMonth`, `ErrInvalidYear` | the expiry date is malformed |
| `ErrExpired` | the expiry month is past |
| `ErrInvalidBINRange`, `ErrConflictingAssignment` | `NewTable` rejects an assignment |

Two carry details for `errors.As`:

```go
var lengthErr *creditcard.LengthError     // Brand, Length, Lengths
var codeErr *creditcard.SecurityCodeError // Brand, Code
```

`Table.Validate` returns the brand alongside a validation error, so you can say
which brand's rule a card broke.

## Expiry dates

`ValidateExpiry(month, year, now)` takes the clock as an argument. A card is
good through the last day of its expiry month *as seen in `now`'s location*, so
pass `now` in the time zone that should decide. Months are `1`-`12` with or
without a leading zero; years have two or four digits, and two digits are read
in `now`'s century.

## Brands

Exactly credit-card-type's, at the release in `creditcard.UpstreamVersion`
([upstream.go](upstream.go)):

American Express · Diners Club · Discover · Elo · Hiper · Hipercard · JCB ·
Maestro · Mastercard · Mir · Naranja · Troy · UnionPay · Verve · Visa

Every brand has a constant (`creditcard.Visa`, `creditcard.AmericanExpress`,
...), and `creditcard.Brands()` lists them in the order the table breaks ties.

## Where the BIN data comes from

Upstream is Braintree's
[credit-card-type](https://github.com/braintree/credit-card-type) (MIT).
[card-validator](https://github.com/braintree/card-validator) carries no BIN data
of its own — it depends on credit-card-type — so that package is the single
upstream source.

Identification follows credit-card-type's algorithm exactly, and the tests run
upstream's own identification cases against it. Each brand is matched by the
first of its ranges the number matches; once the number is at least as long as
every range it matches, the brand with the **longest** range wins. A full card
number always gets that far. A partial one, as while the user types, is decided
only when a single brand still matches it — `"35"` is JCB — and otherwise
`Identify` reports `ErrUnknownBrand`, noting that more digits are needed, where
credit-card-type would return several candidates.

Number length plays no part in identification: a number too long for its brand
is still that brand, and `ValidateNumber` reports the length, as card-validator
does.

Two caveats worth knowing before you rely on this:

- This is a **prefix heuristic, not a BIN database**. It is good for deciding
  which logo to show, how long the input should be, and how many CVV digits to
  ask for. It cannot tell you the issuer, the country, or whether the card is
  debit or credit.
- Visa and Mastercard moved to **8-digit BINs** in 2022, so a 6-digit prefix no
  longer identifies an issuer uniquely. Anything that affects money — routing,
  interchange, 3DS decisions — should use the brand your payment gateway
  returns, not this package.

## Syncing a new upstream release

[upstream.go](upstream.go) is generated. Do not hand-edit it — bump the
version and regenerate:

```bash
go run ./internal/gen -version 10.4.0 && go test ./...
```

Then bump `defaultVersion` in [internal/gen/main.go](internal/gen/main.go) to
match, and commit it with the regenerated `upstream.go` and
`LICENSE-credit-card-type`. A test fails until `defaultVersion` and
`UpstreamVersion` agree, so a later `go generate` cannot quietly rebuild an
older table. The regenerated file carries the
new version in its `UpstreamVersion` constant, so the recorded version cannot
drift from the data.

The generator reads three upstream files: `src/lib/card-types.ts` for the
ranges, lengths and security codes, `src/index.ts` for `ORIGINAL_TEST_ORDER`,
which decides which brand wins when two ranges are equally specific, and
`LICENSE`, which it copies to `LICENSE-credit-card-type`. It copies the data
without adjustment, and fails loudly rather than emitting a short or malformed
table if upstream's file layout changes or its license stops being MIT.

If a new upstream release adds a brand, `TestBrandConstants` fails until a
constant for it is declared in [brand.go](brand.go).

## Upgrading from v1.1

v1.2.0 replaces the v1.1 API rather than extending it. The package has very few
users, so it stays on the v1 module path instead of moving to `/v2`. It is also
the first release under the [MIT license](LICENSE); earlier releases remain CC
BY-SA 4.0.

| v1.1 | v1.2 |
| --- | --- |
| `card.Method()`, `card.MethodValidate()` | `creditcard.Default().Identify(number)` |
| `Company{Short, Long}` | `Brand`, and `brand.Name()` |
| `SpecFor(id)` | `brand.Lengths()`, `brand.SecurityCode()` |
| `card.Validate(allowTestNumbers...)` | `table.Validate(card, now)` |
| `card.ValidateNumber() bool` | `brand.ValidateNumber(number) error` |
| `card.ValidateCVV()` | `brand.ValidateCVV(cvv)` |
| `card.ValidateExpiration()` | `creditcard.ValidateExpiry(month, year, now)` |
| `Card{Number, Cvv, Month, Year, Company}` | `Card{Number, CVV, ExpiryMonth, ExpiryYear}` |
| `Brands() []string`, untyped brand constants | `Brands() []Brand`, `Brand` constants |
| `ErrUnknownMethod` | `ErrUnknownBrand` |
| `*SecurityCodeError{Name, Brand, Required}` | `*SecurityCodeError{Brand, Code}` |

Behaviour that changed:

- **Only upstream's brands.** Aura, Bankcard, Cabal, Dankort, InterPayment,
  InstaPayment and Visa Electron were maintained by hand and are gone, along
  with the Maestro split that made room for Dankort. Their numbers are reported
  the way credit-card-type reports them — Visa Electron as Visa, Dankort and
  Cabal as Maestro, and so on.
- **Unknown brands are rejected.** A number no brand matches is
  `ErrUnknownBrand`; there is no generic 13-19 digit / 3-4 digit fallback. Use
  `creditcard.Luhn` if you want to check such numbers yourself.
- **Partial numbers follow upstream.** A number that could still be several
  brands is undecided: `"6"` was Maestro and is now `ErrUnknownBrand` until
  more digits arrive.
- **Length no longer steers identification.** A 17-digit number starting `34`
  is American Express with a `*LengthError`, not an unknown brand, and a
  19-digit `650031...` is Elo, not Discover.
- **The number is checked first.** `Validate` reports problems in the order
  number, security code, expiry date; v1.1 checked the expiry date first, so
  an expired card with a bad number now reports `ErrInvalidNumber`, not
  `ErrExpired`.
- **Expiry dates use the clock you pass**, in its location, instead of the
  current UTC month. Years must have two or four digits.
- **The identification result is returned, not stored.** `Card` has no
  `Company` field, so it can no longer go stale when `Number` changes.

Removed without a replacement:

- `LegacyCompany`, `LegacyMethod` and the `Legacy*` constants.
- The test-number blocklist and `ErrTestNumberNotAllowed`. Telling test
  traffic from live traffic is your payment gateway's job.
- `Wipe`. Go strings are immutable, so it could only replace the fields, never
  erase the number from memory.
- `LastFour`, `LastFourDigits` and `ErrNumberTooShort`.

## Relationship to durango/go-credit-card

This started as a fork of
[durango/go-credit-card](https://github.com/durango/go-credit-card). v1.1 moved
it to generated BIN data and longest-BIN matching; v1.2 rebuilt the API around
the split between BIN tables and brand rules. Little of the original code
remains.

## License

[MIT](LICENSE).

The BIN data in [upstream.go](upstream.go) and the identification cases in
[upstream_test.go](upstream_test.go) come from
[credit-card-type](https://github.com/braintree/credit-card-type), Copyright (c)
2009-2018 Braintree, a division of PayPal, Inc., and are MIT licensed too: see
[LICENSE-credit-card-type](LICENSE-credit-card-type). The generator copies that
file from the same upstream release as the data.

Releases before v1.2.0 were distributed under CC BY-SA 4.0, the license of
durango/go-credit-card they were forked from. v1.2.0 is a rewrite that contains
none of durango/go-credit-card's code. What it keeps from v1.1 — the generator
and a few small helpers — was written by this project's maintainer, who
releases it under MIT along with the rest.
