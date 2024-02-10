package main

import (
	"fmt"
	"time"

	"github.com/L3Sota/arbo/arb"
	"github.com/L3Sota/arbo/arb/config"
	"github.com/L3Sota/arbo/c"
	"github.com/L3Sota/arbo/g"
	"github.com/L3Sota/arbo/k"
)

func main() {
	deadline := time.NewTimer(59 * time.Second)
	ticker := time.NewTicker(500 * time.Millisecond)

	conf := config.Load()
	k.LoadClient(conf)
	c.LoadClient(conf)
	g.LoadClient()

	for {
		fmt.Println("arb at " + time.Now().String())
		arb.Book(conf)

		select {
		case t <- deadline.C:
			fmt.Println("deadline reached, ending at " + t.String())
			return
		case <-ticker.C:
			continue
		}
	}
}
