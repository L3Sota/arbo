package main

import (
	"fmt"

	"github.com/L3Sota/arbo/arb/config"
	"github.com/L3Sota/arbo/h"
)

func main() {
	h.LoadClient(config.Load())
	book()
	balances()
	h.OrderTest()
}

func book() {
	a, b, err := h.Book()

	fmt.Println(a)
	fmt.Println(b)
	fmt.Println(err)
}

func balances() {
	b, err := h.Balances()

	fmt.Println(b.USDT.String())
	fmt.Println(b.XCH.String())
	fmt.Println(err)
}
