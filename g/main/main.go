package main

import (
	"fmt"

	"github.com/L3Sota/arbo/g"
)

func main() {
	book()
}

func book() {
	a, b, err := g.Book()

	fmt.Println(a)
	fmt.Println(b)
	fmt.Println(err)
}
