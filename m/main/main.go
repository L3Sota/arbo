package main

import (
	"fmt"

	"github.com/L3Sota/arbo/m"
)

func main() {
	a, b, err := m.Book()

	fmt.Println(a)
	fmt.Println(b)
	fmt.Println(err)
}
