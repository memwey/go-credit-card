// Identification cases from https://github.com/braintree/credit-card-type v10.3.1
// Copyright (c) 2009-2018 Braintree, a division of PayPal, Inc.
// MIT licensed; see LICENSE-credit-card-type.

package creditcard_test

import (
	"errors"
	"strings"
	"testing"

	creditcard "github.com/memwey/go-credit-card"
)

// The cases in this file are credit-card-type v10.3.1's own identification
// tests (src/__tests__/index.ts), so Identify is checked against upstream's
// answers rather than against this package's reading of them.

// upstreamDecided are numbers creditCardType resolves to exactly one brand.
var upstreamDecided = []struct {
	number string
	brand  creditcard.Brand
}{
	{"411", creditcard.Visa},
	{"4111111111111111", creditcard.Visa},
	{"4012888888881881", creditcard.Visa},
	{"4222222222222", creditcard.Visa},
	{"4462030000000000", creditcard.Visa},
	{"4484070000000000", creditcard.Visa},
	{"411111111111111111", creditcard.Visa},
	{"4111111111111111110", creditcard.Visa},
	{"431274", creditcard.Elo},
	{"451416", creditcard.Elo},
	{"457393", creditcard.Elo},
	{"401178", creditcard.Elo},
	{"401179", creditcard.Elo},
	{"438935", creditcard.Elo},
	{"457631", creditcard.Elo},
	{"457632", creditcard.Elo},
	{"4576321111111111", creditcard.Elo},
	{"5066991111111118", creditcard.Elo},
	{"504175", creditcard.Elo},
	{"6277809", creditcard.Elo},
	{"6277809990229178", creditcard.Elo},
	{"650033", creditcard.Elo},
	{"6500331111111111", creditcard.Elo},
	{"650057", creditcard.Elo},
	{"650081", creditcard.Elo},
	{"650069", creditcard.Elo},
	{"6500571111111111", creditcard.Elo},
	{"6500811111111111", creditcard.Elo},
	{"651680", creditcard.Elo},
	{"651704", creditcard.Elo},
	{"651692", creditcard.Elo},
	{"6516801111111111", creditcard.Elo},
	{"6517041111111111", creditcard.Elo},
	{"2221", creditcard.Mastercard},
	{"2222", creditcard.Mastercard},
	{"2223", creditcard.Mastercard},
	{"2224", creditcard.Mastercard},
	{"2225", creditcard.Mastercard},
	{"2226", creditcard.Mastercard},
	{"2225", creditcard.Mastercard},
	{"2226", creditcard.Mastercard},
	{"223", creditcard.Mastercard},
	{"2239", creditcard.Mastercard},
	{"23", creditcard.Mastercard},
	{"24", creditcard.Mastercard},
	{"25", creditcard.Mastercard},
	{"26", creditcard.Mastercard},
	{"27", creditcard.Mastercard},
	{"270", creditcard.Mastercard},
	{"271", creditcard.Mastercard},
	{"272", creditcard.Mastercard},
	{"2720", creditcard.Mastercard},
	{"589562", creditcard.Naranja},
	{"5895626292990446", creditcard.Naranja},
	{"402918", creditcard.Naranja},
	{"527572", creditcard.Naranja},
	{"9792100000000001", creditcard.Troy},
	{"9792090000000004", creditcard.Troy},
	{"6502740000000001", creditcard.Troy},
	{"9792", creditcard.Troy},
	{"650274", creditcard.Troy},
	{"6509230000000000", creditcard.Troy},
	{"650052", creditcard.Troy},
	{"650082", creditcard.Troy},
	{"506099", creditcard.Verve},
	{"506100", creditcard.Verve},
	{"506127", creditcard.Verve},
	{"506129", creditcard.Verve},
	{"506133", creditcard.Verve},
	{"506150", creditcard.Verve},
	{"506158", creditcard.Verve},
	{"506163", creditcard.Verve},
	{"506166", creditcard.Verve},
	{"506170", creditcard.Verve},
	{"506176", creditcard.Verve},
	{"506180", creditcard.Verve},
	{"506195", creditcard.Verve},
	{"507865", creditcard.Verve},
	{"507888", creditcard.Verve},
	{"507941", creditcard.Verve},
	{"5061001234567890123", creditcard.Verve},
	{"51", creditcard.Mastercard},
	{"53", creditcard.Mastercard},
	{"54", creditcard.Mastercard},
	{"55", creditcard.Mastercard},
	{"5555555555554444", creditcard.Mastercard},
	{"5454545454545454", creditcard.Mastercard},
	{"34", creditcard.AmericanExpress},
	{"37", creditcard.AmericanExpress},
	{"341", creditcard.AmericanExpress},
	{"34343434343434", creditcard.AmericanExpress},
	{"378282246310005", creditcard.AmericanExpress},
	{"371449635398431", creditcard.AmericanExpress},
	{"378734493671000", creditcard.AmericanExpress},
	{"30", creditcard.DinersClub},
	{"300", creditcard.DinersClub},
	{"36", creditcard.DinersClub},
	{"38", creditcard.DinersClub},
	{"39", creditcard.DinersClub},
	{"30569309025904", creditcard.DinersClub},
	{"38520000023237", creditcard.DinersClub},
	{"36700102000000", creditcard.DinersClub},
	{"36148900647913", creditcard.DinersClub},
	{"6011", creditcard.Discover},
	{"644", creditcard.Discover},
	{"644", creditcard.Discover},
	{"645", creditcard.Discover},
	{"646", creditcard.Discover},
	{"647", creditcard.Discover},
	{"648", creditcard.Discover},
	{"649", creditcard.Discover},
	{"6011000400000000", creditcard.Discover},
	{"6011111111111117", creditcard.Discover},
	{"6011000990139424", creditcard.Discover},
	{"62123456789002", creditcard.UnionPay},
	{"621234567890003", creditcard.UnionPay},
	{"6221258812340000", creditcard.UnionPay},
	{"622018111111111111", creditcard.UnionPay},
	{"6212345678900000003", creditcard.UnionPay},
	{"56", creditcard.Maestro},
	{"57", creditcard.Maestro},
	{"59", creditcard.Maestro},
	{"67", creditcard.Maestro},
	{"6304000000000000", creditcard.Maestro},
	{"6799990100000000019", creditcard.Maestro},
	{"62183", creditcard.Maestro},
	{"1", creditcard.JCB},
	{"35", creditcard.JCB},
	{"2131", creditcard.JCB},
	{"21312", creditcard.JCB},
	{"1800", creditcard.JCB},
	{"18002", creditcard.JCB},
	{"3530111333300000", creditcard.JCB},
	{"3566002020360505", creditcard.JCB},
	{"35308796121637357", creditcard.JCB},
	{"353445444300732639", creditcard.JCB},
	{"3537286818376838569", creditcard.JCB},
	{"6221260000000000", creditcard.UnionPay},
	{"6221260000000000000", creditcard.UnionPay},
	{"6222000000000000", creditcard.UnionPay},
	{"6228000000000000", creditcard.UnionPay},
	{"6229250000000000", creditcard.UnionPay},
	{"6229250000000000000", creditcard.UnionPay},
	{"6240000000000000", creditcard.UnionPay},
	{"6260000000000000000", creditcard.UnionPay},
	{"6282000000000000", creditcard.UnionPay},
	{"6289000000000000000", creditcard.UnionPay},
	{"6221558812340000", creditcard.UnionPay},
	{"6269992058134322", creditcard.UnionPay},
	{"622018111111111111", creditcard.UnionPay},
	{"8", creditcard.UnionPay},
	{"8100513433325374", creditcard.UnionPay},
	{"8111700872004845", creditcard.UnionPay},
	{"8141618644273338", creditcard.UnionPay},
	{"8158163233706018", creditcard.UnionPay},
	{"8168524506870054", creditcard.UnionPay},
	{"220", creditcard.Mir},
	{"2200", creditcard.Mir},
	{"2204", creditcard.Mir},
	{"22000000000000000", creditcard.Mir},
	{"22049999999999999", creditcard.Mir},
	{"6062820524845321", creditcard.Hipercard},
	{"6062820000", creditcard.Hipercard},
	{"6370950000000005", creditcard.Hiper},
	{"637095", creditcard.Hiper},
	{"637609", creditcard.Hiper},
	{"637599", creditcard.Hiper},
	{"637612", creditcard.Hiper},
	{"637568", creditcard.Hiper},
	{"63737423", creditcard.Hiper},
	{"63743358", creditcard.Hiper},
}

// upstreamCandidates are numbers for which creditCardType returns these
// candidates, in this order. One candidate decides the number; several mean
// more digits are needed. Upstream's "" case is left out: an empty number is
// ErrInvalidNumber here.
var upstreamCandidates = []struct {
	number string
	brands []creditcard.Brand
}{
	{"2", []creditcard.Brand{creditcard.Mastercard, creditcard.JCB, creditcard.Mir}},
	{"3", []creditcard.Brand{creditcard.AmericanExpress, creditcard.DinersClub, creditcard.JCB}},
	{"5", []creditcard.Brand{creditcard.Mastercard, creditcard.Naranja, creditcard.Verve, creditcard.Maestro, creditcard.Elo}},
	{"50", []creditcard.Brand{creditcard.Verve, creditcard.Maestro, creditcard.Elo}},
	{"6", []creditcard.Brand{creditcard.Discover, creditcard.Troy, creditcard.UnionPay, creditcard.Maestro, creditcard.Elo, creditcard.Hiper, creditcard.Hipercard}},
	{"60", []creditcard.Brand{creditcard.Discover, creditcard.Maestro, creditcard.Hipercard}},
	{"601", []creditcard.Brand{creditcard.Discover, creditcard.Maestro}},
	{"64", []creditcard.Brand{creditcard.Discover, creditcard.Maestro}},
	{"62", []creditcard.Brand{creditcard.UnionPay, creditcard.Maestro, creditcard.Elo}},
	{"4", []creditcard.Brand{creditcard.Visa, creditcard.Naranja, creditcard.Maestro, creditcard.Elo}},
	{"43", []creditcard.Brand{creditcard.Visa, creditcard.Elo}},
	{"431", []creditcard.Brand{creditcard.Visa, creditcard.Elo}},
	{"4312", []creditcard.Brand{creditcard.Visa, creditcard.Elo}},
	{"43127", []creditcard.Brand{creditcard.Visa, creditcard.Elo}},
	{"45141", []creditcard.Brand{creditcard.Visa, creditcard.Elo}},
	{"45739", []creditcard.Brand{creditcard.Visa, creditcard.Elo}},
	{"40117", []creditcard.Brand{creditcard.Visa, creditcard.Elo}},
	{"43893", []creditcard.Brand{creditcard.Visa, creditcard.Elo}},
	{"45763", []creditcard.Brand{creditcard.Visa, creditcard.Elo}},
	{"52", []creditcard.Brand{creditcard.Mastercard, creditcard.Naranja}},
	{"58", []creditcard.Brand{creditcard.Naranja, creditcard.Maestro}},
	{"58956", []creditcard.Brand{creditcard.Naranja, creditcard.Maestro}},
	{"589562", []creditcard.Brand{creditcard.Naranja}},
	{"9", []creditcard.Brand{creditcard.Troy}},
	{"97", []creditcard.Brand{creditcard.Troy}},
	{"979", []creditcard.Brand{creditcard.Troy}},
	{"650274", []creditcard.Brand{creditcard.Troy}},
	{"650923", []creditcard.Brand{creditcard.Troy}},
	{"506099", []creditcard.Brand{creditcard.Verve}},
	{"506100", []creditcard.Brand{creditcard.Verve}},
	{"506195", []creditcard.Brand{creditcard.Verve}},
	{"507865", []creditcard.Brand{creditcard.Verve}},
	{"507888", []creditcard.Brand{creditcard.Verve}},
	{"50794", []creditcard.Brand{creditcard.Verve, creditcard.Maestro}},
	{"507941", []creditcard.Brand{creditcard.Verve}},
	{"6277", []creditcard.Brand{creditcard.UnionPay, creditcard.Maestro, creditcard.Elo}},
	{"62778", []creditcard.Brand{creditcard.UnionPay, creditcard.Maestro, creditcard.Elo}},
	{"63", []creditcard.Brand{creditcard.Maestro, creditcard.Elo, creditcard.Hiper}},
	{"636", []creditcard.Brand{creditcard.Maestro, creditcard.Elo}},
	{"6362", []creditcard.Brand{creditcard.Maestro, creditcard.Elo}},
	{"63629", []creditcard.Brand{creditcard.Maestro, creditcard.Elo}},
	{"637", []creditcard.Brand{creditcard.Maestro, creditcard.Hiper}},
	{"637374", []creditcard.Brand{creditcard.Maestro, creditcard.Hiper}},
	{"637433", []creditcard.Brand{creditcard.Maestro, creditcard.Hiper}},
	{"606", []creditcard.Brand{creditcard.Maestro, creditcard.Hipercard}},
	{"627", []creditcard.Brand{creditcard.UnionPay, creditcard.Maestro, creditcard.Elo}},
	{"6062", []creditcard.Brand{creditcard.Maestro, creditcard.Hipercard}},
	{"6370", []creditcard.Brand{creditcard.Maestro, creditcard.Hiper}},
	{"6376", []creditcard.Brand{creditcard.Maestro, creditcard.Hiper}},
	{"6375", []creditcard.Brand{creditcard.Maestro, creditcard.Hiper}},
	{"65", []creditcard.Brand{creditcard.Discover, creditcard.Troy, creditcard.Maestro, creditcard.Elo}},
	{"655", []creditcard.Brand{creditcard.Discover, creditcard.Maestro, creditcard.Elo}},
	{"6550", []creditcard.Brand{creditcard.Discover, creditcard.Maestro, creditcard.Elo}},
	{"65502", []creditcard.Brand{creditcard.Discover, creditcard.Maestro, creditcard.Elo}},
}

// upstreamUnknown are numbers creditCardType matches to no brand.
var upstreamUnknown = []string{"0", "12", "123", "181", "1802", "221", "222099", "2721", "212", "2132", "306", "31", "32", "33", "7"}

func TestIdentifyAgreesWithUpstream(t *testing.T) {
	for _, tc := range upstreamDecided {
		if got, err := creditcard.Default().Identify(tc.number); err != nil || got != tc.brand {
			t.Errorf("Identify(%q) = (%q, %v), want %q", tc.number, got, err, tc.brand)
		}
	}

	for _, tc := range upstreamCandidates {
		got, err := creditcard.Default().Identify(tc.number)

		if len(tc.brands) == 1 {
			if err != nil || got != tc.brands[0] {
				t.Errorf("Identify(%q) = (%q, %v), want %q, its only candidate", tc.number, got, err, tc.brands[0])
			}

			continue
		}

		if !errors.Is(err, creditcard.ErrUnknownBrand) || !strings.Contains(err.Error(), "more digits") || got != "" {
			t.Errorf("Identify(%q) = (%q, %v), want more digits needed to choose among %q", tc.number, got, err, tc.brands)
		}
	}

	for _, number := range upstreamUnknown {
		if got, err := creditcard.Default().Identify(number); err != creditcard.ErrUnknownBrand || got != "" {
			t.Errorf("Identify(%q) = (%q, %v), want ErrUnknownBrand and no more", number, got, err)
		}
	}
}
