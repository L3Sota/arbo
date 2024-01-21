package main

import (
	"github.com/L3Sota/arbo/arb"
	"github.com/L3Sota/arbo/arb/config"
)

func main() {
	// arb.GatherBalancesP(config.Load())
	arb.Book(config.Load())
}
