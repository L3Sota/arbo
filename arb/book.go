package arb

import (
	"fmt"

	"github.com/L3Sota/arbo/arb/model"
	"github.com/L3Sota/arbo/g"
	"github.com/L3Sota/arbo/k"
	"github.com/L3Sota/arbo/m"
	"github.com/shopspring/decimal"
)

var index = map[model.ExchangeType][]model.Order{}
var book = []model.Order{}

// gather price information from all exchanges
// REST to get initial book state
// WS to get streaming updates
// + rate limiter to prevent throttling (start with some reasonable #)

// buy orders --->| (mid-market price) |<--- sell orders
// if the books cross...
// buy orders --->|
// |<--- sell orders
// ...then we can arb!
// |<--- arb  --->|
// ^buy here      ^ sell here

// + keep track of funding info to deposit/transfer/withdraw as necessary

func Book() {
	ma, mb := m.M()
	ka, kb := k.K()
	ga, gb := g.G()

	a := merge(true, ma, ka, ga)
	b := merge(false, mb, kb, gb)

	fmt.Println("===")
	for _, ask := range a {
		fmt.Println(ask.Price.StringFixed(2), ask.Amount.String())
	}
	fmt.Println("---")
	for _, bid := range b {
		fmt.Println(bid.Price.StringFixed(2), bid.Amount.String())
	}
	fmt.Println("===")

	if a[0].Price.LessThan(b[0].Price) {
		msg := fmt.Sprintf("%v (%v) < %v (%v)", a[0].Price.String(), a[0].Amount.String(), b[0].Price.String(), b[0].Amount.String())
		fmt.Println(msg)
	}
}

func merge(asc bool, xs ...[]model.Order) []model.Order {
	wheres := make([]int, len(xs))
	s := 0
	for _, x := range xs {
		s += len(x)
	}
	m := make([]model.Order, 0, s)

	for {
		which := -1
		where := -1
		best := decimal.Zero
		for i, j := range wheres {
			switch {
			case j == len(xs[i]):
			case which == -1,
				asc && xs[i][j].Price.LessThan(best),
				!asc && xs[i][j].Price.GreaterThan(best):
				which = i
				where = j
				best = xs[i][j].Price
			}
		}
		if which == -1 {
			return m
		}
		m = append(m, xs[which][where])
		wheres[which]++
	}
}
