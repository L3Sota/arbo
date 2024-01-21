package main

import (
	"fmt"

	"github.com/L3Sota/arbo/arb/config"
	"github.com/L3Sota/arbo/g"
)

func main() {
	balances()
	// g.OrderTest(config.Load())
	// g.QueryFee(config.Load())
	// book()
}

func book() {
	a, b, err := g.Book()

	fmt.Println(a)
	fmt.Println(b)
	fmt.Println(err)
}

func balances() {
	b, err := g.Balances(config.Load())

	fmt.Println(b.USDT.String())
	fmt.Println(b.XCH.String())
	fmt.Println(err)
}
