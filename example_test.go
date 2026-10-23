package creditcard_test

import (
	"fmt"
	"time"

	creditcard "github.com/memwey/go-credit-card"
)

func ExampleTable_Validate() {
	now := time.Date(2026, time.October, 23, 0, 0, 0, 0, time.UTC)
	card := creditcard.Card{Number: "378282246310005", CVV: "123", ExpiryMonth: "12", ExpiryYear: "30"}

	brand, err := creditcard.Default().Validate(card, now)
	fmt.Println(brand, err)

	// Output:
	// american-express invalid security code: American Express uses a 4 digit CID
}

func ExampleBrand_ValidateNumber() {
	fmt.Println(creditcard.Visa.ValidateNumber("4242424242424242"))
	fmt.Println(creditcard.AmericanExpress.ValidateNumber("4242424242424242"))

	// Output:
	// <nil>
	// invalid card number: American Express numbers have 15 digits, not 16
}

// A card network publishes its own BIN ranges: eight-digit JCB IINs upstream
// does not know yet, and the 6440-6599 block it accepts as Discover even where
// upstream files a BIN under Elo.
func ExampleNewTable() {
	table, err := creditcard.NewTable(
		creditcard.Assign(creditcard.JCB, creditcard.Range("30910000", "30949999")),
		creditcard.Assign(creditcard.Discover, creditcard.Range("6440", "6599")),
	)
	if err != nil {
		panic(err)
	}

	for _, number := range []string{"3091000000000000", "6500310000000000", "4242424242424242"} {
		upstream, _ := creditcard.Default().Identify(number)
		ours, _ := table.Identify(number)

		fmt.Printf("%s  upstream=%-8q table=%q\n", number, upstream, ours)
	}

	// Validate checks the card by the rules of the brand table identified.
	card := creditcard.Card{Number: "6500310000000005", CVV: "123", ExpiryMonth: "12", ExpiryYear: "30"}
	now := time.Date(2026, time.October, 23, 0, 0, 0, 0, time.UTC)
	fmt.Println(table.Validate(card, now))

	// Output:
	// 3091000000000000  upstream=""       table="jcb"
	// 6500310000000000  upstream="elo"    table="discover"
	// 4242424242424242  upstream="visa"   table="visa"
	// discover <nil>
}
