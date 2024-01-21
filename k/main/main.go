package main

import (
	"fmt"

	"github.com/L3Sota/arbo/k"
)

func main() {
	book()
}

func book() {
	a, b, err := k.Book()

	fmt.Print(a)
	fmt.Print(b)
	fmt.Print(err)
}
