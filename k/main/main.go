package main

import (
	"fmt"

	"github.com/L3Sota/arbo/arb/config"
	"github.com/L3Sota/arbo/k"
)

func main() {
	k.LoadClient(config.Load())
	k.OrderTest()
	// k.QueryFee()
	// book()
	// balances()
}

func book() {
	a, b, err := k.Book()

	fmt.Print(a)
	fmt.Print(b)
	fmt.Print(err)
}

func balances() {
	b, err := k.Balances()

	fmt.Println(b.USDT.String())
	fmt.Println(b.XCH.String())
	fmt.Println(err)
}
