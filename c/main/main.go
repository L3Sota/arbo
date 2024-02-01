package main

import (
	"fmt"

	"github.com/L3Sota/arbo/c"
)

func main() {
	balances()
	// book()
}

func book() {
	a, b, err := c.Book()

	fmt.Println(a)
	fmt.Println(b)
	fmt.Println(err)
}

func balances() {
	b, err := c.Balances()

	fmt.Println(b.USDT.String())
	fmt.Println(b.XCH.String())
	fmt.Println(err)
}
