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
		fmt.Println(ask.Ex, ask.Price.StringFixed(2), ask.Amount.String())
	}
	fmt.Println("---")
	for _, bid := range b {
		fmt.Println(bid.Ex, bid.Price.StringFixed(2), bid.Amount.String())
	}
	fmt.Println("===")

	ai := 0
	bi := 0
	aAmount := a[ai].Amount
	bAmount := b[bi].Amount
	lastA := a[0].Price
	lastB := b[0].Price
	totalTradeQuote := decimal.Zero
	totalBuyBase := decimal.Zero
	totalSellBase := decimal.Zero
	profit := decimal.Zero
	// buy into the low asks, sell off to the high bids
	for {
		ap := a[ai].Price
		bp := b[bi].Price
		if ap.GreaterThanOrEqual(bp) {
			break
		}

		aa := a[ai].Amount
		ae := a[ai].Ex
		ba := b[bi].Amount
		be := b[bi].Ex

		tradeAmount := decimal.Zero
		switch aAmount.Cmp(bAmount) {
		case -1:
			tradeAmount = aAmount
			bAmount = bAmount.Sub(tradeAmount)
			lastA = ap
			ai++
			aAmount = aa
		case 1:
			tradeAmount = bAmount
			aAmount = aAmount.Sub(tradeAmount)
			lastB = bp
			bi++
			bAmount = ba
		case 0:
			tradeAmount = aAmount
			lastA = ap
			lastB = bp
			ai++
			bi++
			aAmount = aa
			bAmount = ba
		}
		totalTradeQuote = totalTradeQuote.Add(tradeAmount)
		profit = profit.Add(tradeAmount.Mul(bp.Sub(ap)))
		totalBuyBase[ae] = totalBuyBase[ae].Add(ap.Mul(tradeAmount))
		totalSellBase[be] = totalSellBase[be].Add(bp.Mul(tradeAmount))
	}

	msg := fmt.Sprintf("Buy $ %v / Sell $ %v ; Asks %v - %v / Bids %v - %v ; amount %v XCH (p %v)",
		totalBuyBase, totalSellBase, a[0].Price, lastA, b[0].Price, lastB, totalTradeQuote, profit)
	fmt.Println(msg)
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
