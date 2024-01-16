package arb

import (
	"context"
	"fmt"

	"github.com/L3Sota/arbo/arb/config"
	"github.com/L3Sota/arbo/arb/model"
	"github.com/L3Sota/arbo/c"
	"github.com/L3Sota/arbo/g"
	"github.com/L3Sota/arbo/k"
	"github.com/L3Sota/arbo/m"
	"github.com/gregdel/pushover"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

var index = map[model.ExchangeType][]model.Order{}
var book = []model.Order{}

var fees = map[model.ExchangeType]model.Fees{
	model.ME: m.Fees,
	model.Ku: k.Fees,
	model.Ga: g.Fees,
	model.Co: c.Fees,
}

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

func GatherBooks() ([]model.Order, []model.Order) {
	ma, mb := m.Book()
	ka, kb := k.Book()
	ga, gb := g.Book()
	ca, cb := c.Book()

	a := merge(true, ma, ka, ga, ca)
	b := merge(false, mb, kb, gb, cb)

	return a, b
}

func GatherBooksP() ([]model.Order, []model.Order) {
	var ma, ka, ga, ca, mb, kb, gb, cb []model.Order
	eg, _ := errgroup.WithContext(context.TODO())
	eg.Go(func() error {
		ma, mb = m.Book()
		return nil
	})
	eg.Go(func() error {
		ka, kb = k.Book()
		return nil
	})
	eg.Go(func() error {
		ga, gb = g.Book()
		return nil
	})
	eg.Go(func() error {
		ca, cb = c.Book()
		return nil
	})
	eg.Wait()

	a := merge(true, ma, ka, ga, ca)
	b := merge(false, mb, kb, gb, cb)

	return a, b
}

func Book(c *config.Config) {
	a, b := GatherBooksP()

	ai := 0
	bi := 0
	aAmount := a[ai].Amount
	bAmount := b[bi].Amount
	lastA := a[0].Price
	lastB := b[0].Price
	totalTradeQuote := decimal.Zero
	totalBuyBase := make(map[model.ExchangeType]decimal.Decimal, len(model.ExchangeTypes))
	totalSellBase := make(map[model.ExchangeType]decimal.Decimal, len(model.ExchangeTypes))
	for _, t := range model.ExchangeTypes {
		totalBuyBase[t] = decimal.Zero
		totalSellBase[t] = decimal.Zero
	}
	gain := decimal.Zero
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
		gain = gain.Add(tradeAmount.Mul(bp.Sub(ap)))
		totalBuyBase[ae] = totalBuyBase[ae].Add(ap.Mul(tradeAmount))
		totalSellBase[be] = totalSellBase[be].Add(bp.Mul(tradeAmount))
	}

	// buy XCH -> withdraw XCH
	withdrawXCH := decimal.Zero
	for e, b := range totalBuyBase {
		if b.IsPositive() {
			withdrawXCH = withdrawXCH.Add(fees[e].WithdrawalFlatXCH)
		}
	}
	// sell XCH -> withdraw USDT
	withdrawUSDT := decimal.Zero
	for e, s := range totalSellBase {
		if s.IsPositive() {
			withdrawUSDT = withdrawUSDT.Add(fees[e].WithdrawalFlatUSDT)
		}
	}

	profit := gain.Sub(withdrawUSDT).Sub(withdrawXCH.Mul(lastA))

	msg := fmt.Sprintf("Buy $ %v / Sell $ %v ; Asks %v - %v / Bids %v - %v ; trade %v XCH (g %v - %v XCH - %v USDT = p %v)",
		totalBuyBase, totalSellBase, a[0].Price, lastA, b[0].Price, lastB, totalTradeQuote, gain, withdrawXCH, withdrawUSDT, profit)

	if profit.IsPositive() && c.PEnable {
		p := pushover.New(c.PKey)
		r := pushover.NewRecipient(c.PUser)
		resp, err := p.SendMessage(&pushover.Message{
			Message: msg,
		}, r)
		if err != nil {
			fmt.Print(err)
			return
		}
		fmt.Println(resp.String())
	}

	fmt.Println("===")
	for i, ask := range a {
		if i > ai+5 {
			break
		}
		fmt.Println(ask.Ex, ask.EffectivePrice.StringFixed(2), ask.Price.StringFixed(2), ask.Amount.String())
	}
	fmt.Println("---")
	for i, bid := range b {
		if i > ai+5 {
			break
		}
		fmt.Println(bid.Ex, bid.EffectivePrice.StringFixed(2), bid.Price.StringFixed(2), bid.Amount.String())
	}
	fmt.Println("===")
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
				asc && xs[i][j].EffectivePrice.LessThan(best),
				!asc && xs[i][j].EffectivePrice.GreaterThan(best):
				which = i
				where = j
				best = xs[i][j].EffectivePrice
			}
		}
		if which == -1 {
			return m
		}
		m = append(m, xs[which][where])
		wheres[which]++
	}
}
