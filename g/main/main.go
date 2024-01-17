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
	u, x := g.Balances(config.Load())

	fmt.Println(u.String())
	fmt.Println(x.String())
}
