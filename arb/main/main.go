package main

import (
	"github.com/L3Sota/arbo/arb"
	"github.com/L3Sota/arbo/arb/config"
	"github.com/L3Sota/arbo/g"
	"github.com/L3Sota/arbo/k"
)

func main() {
	conf := config.Load()
	k.LoadClient(conf)
	g.LoadClient()
	// arb.GatherBalancesP(conf)
	arb.Book(conf)
}
