package arb

import (
	"fmt"

	"github.com/L3Sota/arbo/arb/model"
	"github.com/L3Sota/arbo/k"
	"github.com/L3Sota/arbo/m"
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

	a := merge(ma, ka, true)
	b := merge(mb, kb, false)

	for i, ask := range a {
		fmt.Println(i, ask.Price.StringFixed(2), ask.Amount.String())
	}
	fmt.Println("---")
	for i, bid := range b {
		fmt.Println(i, bid.Price.StringFixed(2), bid.Amount.String())
	}

	if a[0].Price.LessThan(b[0].Price) {
		msg := fmt.Sprintf("%v (%v) < %v (%v)", a[0].Price.String(), a[0].Amount.String(), b[0].Price.String(), b[0].Amount.String())
		fmt.Println(msg)
	}
}

func merge(x, y []model.Order, asc bool) []model.Order {
	m := make([]model.Order, 0, len(x)+len(y))
	i := 0
	j := 0
	for {
		if i == len(x) {
			m = append(m, y[j:]...)
			return m
		}
		if j == len(y) {
			m = append(m, x[i:]...)
			return m
		}

		if asc {
			if x[i].Price.LessThan(y[j].Price) {
				m = append(m, x[i])
				i++
			} else {
				m = append(m, y[j])
				j++
			}
		} else {
			if x[i].Price.GreaterThan(y[j].Price) {
				m = append(m, x[i])
				i++
			} else {
				m = append(m, y[j])
				j++
			}
		}
	}
}
