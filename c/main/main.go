package main

import (
	"fmt"

	"github.com/L3Sota/arbo/c"
)

func main() {
	a, b, err := c.Book()

	fmt.Println(a)
	fmt.Println(b)
	fmt.Println(err)
}
