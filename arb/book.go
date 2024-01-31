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

type side struct {
	Book       []model.Order
	I          int
	HeadAmount decimal.Decimal
	LastPrice  decimal.Decimal
	Move       bool
}

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
	ma, mb, err := m.Book()
	if err != nil {
		return nil, nil
	}
	ka, kb, err := k.Book()
	if err != nil {
		return nil, nil
	}
	ga, gb, err := g.Book()
	if err != nil {
		return nil, nil
	}
	ca, cb, err := c.Book()
	if err != nil {
		return nil, nil
	}

	a := merge(true, ma, ka, ga, ca)
	b := merge(false, mb, kb, gb, cb)

	return a, b
}

func GatherBooksP() ([]model.Order, []model.Order) {
	var ma, ka, ga, ca, mb, kb, gb, cb []model.Order
	eg, _ := errgroup.WithContext(context.Background())
	eg.Go(func() error {
		a, b, err := m.Book()
		if err != nil {
			return fmt.Errorf("m book: %w", err)
		}
		ma = a
		mb = b
		return nil
	})
	eg.Go(func() error {
		a, b, err := k.Book()
		if err != nil {
			return fmt.Errorf("k book: %w", err)
		}
		ka = a
		kb = b
		return nil
	})
	eg.Go(func() error {
		a, b, err := g.Book()
		if err != nil {
			return fmt.Errorf("g book: %w", err)
		}
		ga = a
		gb = b
		return nil
	})
	eg.Go(func() error {
		a, b, err := c.Book()
		if err != nil {
			return fmt.Errorf("c book: %w", err)
		}
		ca = a
		cb = b
		return nil
	})
	if err := eg.Wait(); err != nil {
		fmt.Println(err)
		return nil, nil
	}

	a := merge(true, ma, ka, ga, ca)
	b := merge(false, mb, kb, gb, cb)

	return a, b
}

func Book(c *config.Config) {
	a, b := GatherBooksP()

	as, bs, totalTradeXCH, gain, withdrawUSDT, withdrawXCH, profit, totalBuyUSDT, totalSellUSDT := arbo(a, b)

	msg := fmt.Sprintf("Buy $ %v / Sell $ %v ; Asks %v - %v / Bids %v - %v ; trade %v XCH (g %v - %v XCH - %v USDT = p %v)",
		totalBuyUSDT, totalSellUSDT, a[0].Price, as.LastPrice, b[0].Price, bs.LastPrice, totalTradeXCH, gain, withdrawXCH, withdrawUSDT, profit)

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
		if i > as.I+5 {
			break
		}
		fmt.Println(ask.Ex, ask.EffectivePrice.StringFixed(2), ask.Price.StringFixed(2), ask.Amount.String())
	}
	fmt.Println("---")
	for i, bid := range b {
		if i > bs.I+5 {
			break
		}
		fmt.Println(bid.Ex, bid.EffectivePrice.StringFixed(2), bid.Price.StringFixed(2), bid.Amount.String())
	}
	fmt.Println("===")
	fmt.Println(msg)
}

func arbo(a, b []model.Order) (side, side, decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal, map[model.ExchangeType]decimal.Decimal, map[model.ExchangeType]decimal.Decimal) {
	totalTradeXCH := decimal.Zero
	totalBuyUSDT := make(map[model.ExchangeType]decimal.Decimal, len(model.ExchangeTypes))
	totalSellUSDT := make(map[model.ExchangeType]decimal.Decimal, len(model.ExchangeTypes))
	for _, t := range model.ExchangeTypes {
		totalBuyUSDT[t] = decimal.Zero
		totalSellUSDT[t] = decimal.Zero
	}
	gain := decimal.Zero

	as := &side{
		Book: a,
	}
	bs := &side{
		Book: b,
	}

	sides := [2]*side{as, bs}

	for _, s := range sides {
		if len(s.Book) > 0 {
			s.HeadAmount = s.Book[0].Amount
			s.LastPrice = s.Book[0].Price
		}
	}

	// buy into the low asks, sell off to the high bids
	for as.I < len(a) && bs.I < len(b) {
		// deferred update to prevent running off end of array
		for _, s := range sides {
			if s.Move {
				s.HeadAmount = s.Book[s.I].Amount
				s.Move = false
			}
		}

		aa := a[as.I]
		bb := b[bs.I]

		// no match
		if aa.EffectivePrice.GreaterThanOrEqual(bb.EffectivePrice) {
			break
		}

		// arb
		tradeAmount := decimal.Zero
		switch as.HeadAmount.Cmp(bs.HeadAmount) {
		case -1:
			tradeAmount = as.HeadAmount
			bs.HeadAmount = bs.HeadAmount.Sub(tradeAmount)
			as.Move = true
		case 1:
			tradeAmount = bs.HeadAmount
			as.HeadAmount = as.HeadAmount.Sub(tradeAmount)
			bs.Move = true
		case 0:
			tradeAmount = as.HeadAmount
			as.Move = true
			bs.Move = true
		}
		totalTradeXCH = totalTradeXCH.Add(tradeAmount)
		gain = gain.Add(tradeAmount.Mul(bb.EffectivePrice.Sub(aa.EffectivePrice)))
		totalBuyUSDT[aa.Ex] = totalBuyUSDT[aa.Ex].Add(aa.Price.Mul(tradeAmount))
		totalSellUSDT[bb.Ex] = totalSellUSDT[bb.Ex].Add(bb.Price.Mul(tradeAmount))

		for _, s := range sides {
			s.LastPrice = s.Book[s.I].Price
			if s.Move {
				s.I++
			}
		}
	}

	// buy XCH -> withdraw XCH
	withdrawXCH := decimal.Zero
	for e, b := range totalBuyUSDT {
		if b.IsPositive() {
			withdrawXCH = withdrawXCH.Add(fees[e].WithdrawalFlatXCH)
		}
	}
	// sell XCH -> withdraw USDT
	withdrawUSDT := decimal.Zero
	for e, s := range totalSellUSDT {
		if s.IsPositive() {
			withdrawUSDT = withdrawUSDT.Add(fees[e].WithdrawalFlatUSDT)
		}
	}

	profit := gain.Sub(withdrawUSDT).Sub(withdrawXCH.Mul(bs.LastPrice))

	return *as, *bs, totalTradeXCH, gain, withdrawUSDT, withdrawXCH, profit, totalBuyUSDT, totalSellUSDT
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
