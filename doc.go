// Package creditcard identifies the brand of a payment card number from its
// BIN and validates card numbers, security codes and expiry dates by that
// brand's rules.
//
// The package keeps two questions apart:
//
//   - Which brand does a number belong to? That depends on a BIN table. Default
//     is the package table, generated from Braintree's credit-card-type and
//     following it exactly. NewTable layers ranges of your own on top of it
//     when your BIN data disagrees with upstream.
//   - What do that brand's numbers and security codes look like? That is a
//     fact about the brand, not about any table, so those rules are methods on
//     Brand and never look a number up again.
//
// Table.Validate does both in order: it identifies the number once, then
// applies that brand's rules. A number can therefore never be identified by one
// table and validated by another.
//
// Card numbers are taken exactly as given: digits only, no spaces or dashes.
// Nothing is normalised, and no error returned by this package contains a card
// number or security code.
//
// The package table is generated from Braintree's credit-card-type, which is
// MIT licensed like this package; its copyright notice and license are in
// LICENSE-credit-card-type.
package creditcard
