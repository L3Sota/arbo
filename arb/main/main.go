package main

import (
	"github.com/L3Sota/arbo/arb"
	"github.com/L3Sota/arbo/arb/config"
	"github.com/L3Sota/arbo/g"
)

func main() {
	conf := config.Load()
	g.LoadClient()
	// arb.GatherBalancesP(conf)
	arb.Book(conf)
}
