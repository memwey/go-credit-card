# go-credit-card

[![Go Reference](https://pkg.go.dev/badge/github.com/memwey/go-credit-card.svg)](https://pkg.go.dev/github.com/memwey/go-credit-card)

Credit card brand identification and validation for Go.

No dependencies — `go.mod` declares none, and the tests use the standard library
only.

Brand identification uses BIN data synced from Braintree's
[credit-card-type](https://github.com/braintree/credit-card-type). Number
validation uses the [Luhn algorithm](https://en.wikipedia.org/wiki/Luhn_algorithm)
together with the brand's own length and security-code rules.

## Relationship to durango/go-credit-card

This started as a fork of
[durango/go-credit-card](https://github.com/durango/go-credit-card) and has
diverged substantially:

- The BIN table is **generated** from credit-card-type rather than hand-written,
  and is kept in sync by `go run ./internal/gen`.
- Identification uses **longest-BIN-wins** matching instead of a hand-ordered
  `switch`, so a specific six- or eight-digit BIN always beats a broad two-digit
  prefix. Eight-digit BINs are supported; the original's `[6]int` prefix buffer
  could not hold them.
- Brand ids follow upstream's naming (`american-express`, not `amex`).
  [`LegacyCompany`](#migrating-from-the-original-brand-names) reports the old
  names for callers that need them.
- `Validate`, `ValidateCVV` and `ValidateNumber` apply the identified brand's
  rules instead of a blanket 13-19 digit / 3-4 digit CVV check.
- Error messages follow Go conventions — lowercase, no trailing punctuation
  (`"invalid year"`, not `"Invalid year"`). Callers that match error text
  verbatim need updating.
- Mir, Hiper, Troy and Verve were added; several wrong Elo ranges, a dead Maestro
  prefix check and a Diners Club length check were fixed.
- Tests use the standard library and cover 100% of statements.

## Installation

```bash
go get github.com/memwey/go-credit-card
```

## Quick start

```go
card := creditcard.Card{Number: "4242424242424242", Cvv: "111", Month: "02", Year: "2030"}

// Which brand does this card belong to?
err := card.Method() // card.Company == Company{Short: "visa", Long: "Visa"}

// What do this brand's numbers look like?
spec, ok := creditcard.SpecFor(card.Company.Short)
// spec == Spec{Lengths: []int{16, 18, 19}, Code: SecurityCode{Name: "CVV", Size: 3}}

// Last four digits, for display
lastFour, err := card.LastFour() // "4242"

// Expiry, number and CVV in one call
err = card.Validate()     // rejects numbers on the compatibility test-card blocklist
err = card.Validate(true) // allows numbers on that blocklist
```

`Validate`, `ValidateCVV` and `ValidateNumber` use the identified brand's own
rules — American Express wants exactly 15 digits and a 4 digit CID, Visa wants
16, 18 or 19 digits and a 3 digit CVV, and so on. Numbers matching no known brand
fall back to the generic 13-19 digit and 3-4 digit CVV rules.

The test-number check is an intentionally finite compatibility blocklist. It
does not identify every test number from every payment provider and must not be
used to separate production from sandbox traffic; use the payment gateway's
live/test credentials and environment controls for that boundary.

## Error handling

Validation and identification errors expose stable sentinel values for
`errors.Is`, including `ErrInvalidNumber`, `ErrExpired`, `ErrInvalidCVV` and
`ErrUnknownMethod`:

```go
if errors.Is(err, creditcard.ErrExpired) {
	// Ask for a different card.
}
```

A brand-specific CVV failure is also a `*creditcard.SecurityCodeError`, so its
details are available through `errors.As` while it still matches
`ErrInvalidCVV`:

```go
var codeErr *creditcard.SecurityCodeError
if errors.As(err, &codeErr) {
	// codeErr.Name, codeErr.Brand, codeErr.Required
}
```

`Company` holds only `Short` and `Long`, so it and `Card` stay comparable and
usable as map keys. Per-brand lengths and security codes are static facts about
the brand rather than about the card, so they live behind `SpecFor`, which
accepts current and legacy ids alike.

Every brand id has a constant, so there is no need to hand-write the strings:

```go
switch card.Company.Short {
case creditcard.Visa, creditcard.Mastercard:
	// ...
case creditcard.AmericanExpress:
	// ...
}
```

`creditcard.Brands()` returns every id the table can report, in the order it
tests them.

## Brands

Tracked from upstream ([cardtypes_upstream.go](cardtypes_upstream.go)):

American Express · China UnionPay · Diners Club · Discover · Elo · Hiper ·
Hipercard · JCB · Maestro · Mastercard · Mir · Naranja · Troy · Verve · Visa

Maintained here ([cardtypes_extra.go](cardtypes_extra.go)), because
credit-card-type does not model them:

Aura \* · Bankcard · Cabal · Dankort · InterPayment · InstaPayment ·
Visa Electron

\* Aura is registered under the bare `50` prefix, which Maestro covers with
six-digit ranges. Maestro matches on more digits and wins, so Aura is currently
unreachable. Add a specific Aura BIN range to `cardtypes_extra.go` to change
that.

## Where the BIN data comes from

Upstream is Braintree's
[credit-card-type](https://github.com/braintree/credit-card-type) (MIT), pinned
at the release in `creditcard.UpstreamVersion`.
[card-validator](https://github.com/braintree/card-validator) carries no BIN data
of its own — it depends on credit-card-type — so that package is the single
upstream source.

Identification follows credit-card-type's algorithm: every pattern the number
satisfies is collected, and the brand matching on the **longest** BIN pattern
wins. Patterns match partial numbers too, so a brand can be identified while the
user is still typing.

Two caveats worth knowing before you rely on this:

- This is a **prefix heuristic, not a BIN database**. It is good for deciding
  which logo to show, how long the input should be, and how many CVV digits to
  ask for. It cannot tell you the issuer, the country, or whether the card is
  debit or credit.
- Visa and Mastercard moved to **8-digit BINs** in 2022, so a 6-digit prefix no
  longer identifies an issuer uniquely. Anything that affects money — routing,
  interchange, 3DS decisions — should use the brand your payment gateway returns,
  not this package.

## Syncing a new upstream release

[cardtypes_upstream.go](cardtypes_upstream.go) is generated. Do not hand-edit it
— bump the version and regenerate:

```bash
go run ./internal/gen -version 10.4.0 && go test .
```

Then commit the resulting diff along with the version bump in
[internal/gen/main.go](internal/gen/main.go). The regenerated file carries the
new version in its `UpstreamVersion` constant, so the recorded version cannot
drift from the data.

The generator reads two upstream files: `src/lib/card-types.ts` for the patterns,
lengths and security codes, and `src/index.ts` for `ORIGINAL_TEST_ORDER`, which
decides which brand wins when two patterns are equally specific. It fails loudly
rather than silently emitting a short table if upstream's file layout changes.

Local deviations live in the generator's `splits` table and are reproduced as
`DEVIATION:` comments in the generated output. There is currently one: Maestro's
`500000-504174` is split around `501900-501999` so Dankort keeps its own
identity, mirroring how upstream already splits that range around Elo's `504175`.

If a new upstream release adds a brand, `TestBrandConstants` fails until a
constant for it is declared in [brands.go](brands.go).

## Migrating from the original brand names

Aligning with upstream renamed some brands and folded the three Diners Club
sub-brands into one. `LegacyCompany` and `LegacyMethod` report the old names, so
callers written against durango/go-credit-card keep working while they migrate:

```go
card := creditcard.Card{Number: "3782822463"}

err := card.Method()       // card.Company.Short == "american-express"
err = card.LegacyMethod()  // card.Company.Short == "amex"
```

| Original | Now | Constants |
| --- | --- | --- |
| `amex` | `american-express` | `LegacyAmericanExpress` / `AmericanExpress` |
| `china unionpay` / `China UnionPay` | `unionpay` / `UnionPay` | `LegacyUnionPay` / `UnionPay` |
| `mastercard` / `MasterCard` | `mastercard` / `Mastercard` | `Mastercard` |
| `visa electron` | `visa-electron` | `LegacyVisaElectron` / `VisaElectron` |
| `diners club carte blanche` | `diners-club` | `LegacyDinersClubCarteBlanche` / `DinersClub` |
| `diners club enroute` | *(no longer recognised)* | `LegacyDinersClubEnRoute` |
| `diners club international` | `diners-club` | `LegacyDinersClubInternational` / `DinersClub` |

`LegacyCompany` also restores the two prefixes the alignment dropped — enRoute's
`2014` and `2149`, and International's `309` — and splits Diners Club back into
Carte Blanche and International by number length. `SpecFor` accepts the legacy
ids too.

It does **not** reproduce the original table's mistakes. BINs that belong to
Hiper, Troy, Verve or Mir are reported under those brands even through
`LegacyCompany`, because the original table simply lacked them and misfiled
`637095` as InstaPayment. The three broken Elo ranges (`650035-650033`,
`650901-650920`, `655021-655021`) stay corrected too.

There is no legacy path for validation: `Validate`, `ValidateCVV` and
`ValidateNumber` always apply the brand's own rules.

## License

Code is [CC BY-SA 4.0](LICENSE.md), inherited from durango/go-credit-card.

The BIN data in [cardtypes_upstream.go](cardtypes_upstream.go) is derived from
[credit-card-type](https://github.com/braintree/credit-card-type), which is MIT
licensed.
