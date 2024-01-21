package main

import (
	"fmt"

	"github.com/L3Sota/arbo/arb/config"
	"github.com/L3Sota/arbo/k"
)

func main() {
	k.OrderTest(config.Load())
	// k.QueryFee(config.Load())
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
	b, err := k.Balances(config.Load())

	fmt.Println(b.USDT.String())
	fmt.Println(b.XCH.String())
	fmt.Println(err)
}
