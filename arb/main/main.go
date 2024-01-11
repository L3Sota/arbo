package main

import (
	"github.com/L3Sota/arbo/arb"
	"github.com/L3Sota/arbo/arb/config"
	"github.com/kelseyhightower/envconfig"
)

func main() {
	c := &config.Config{}
	envconfig.MustProcess("ARBO", c)

	arb.Book(c)
}
