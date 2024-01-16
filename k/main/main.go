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
	u, x := k.Balances(config.Load())

	fmt.Println(u.String())
	fmt.Println(x.String())
}
