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
	Book          []model.Order
	I             int
	HeadAmount    decimal.Decimal
	HeadAllowance decimal.Decimal
	LastPrice     decimal.Decimal
	Move          bool
}

var fees = map[model.ExchangeType]model.Fees{
	model.ExchangeTypeMe: m.Fees,
	model.ExchangeTypeKu: k.Fees,
	model.ExchangeTypeCo: c.Fees,
	model.ExchangeTypeGa: g.Fees,
}

var big = decimal.New(1, 10)
var bigBalance = model.Balances{
	XCH:  big,
	USDT: big,
}

var ignoreBalances = map[model.ExchangeType]model.Balances{
	model.ExchangeTypeMe: bigBalance,
	model.ExchangeTypeKu: bigBalance,
	model.ExchangeTypeHu: bigBalance,
	model.ExchangeTypeCo: bigBalance,
	model.ExchangeTypeGa: bigBalance,
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

func GatherBalancesP(conf *config.Config) map[model.ExchangeType]model.Balances {
	m := make(map[model.ExchangeType]model.Balances, len(model.ExchangeTypes))
	eg, _ := errgroup.WithContext(context.Background())
	eg.Go(func() error {
		m[model.ExchangeTypeMe] = model.Balances{}
		return nil
	})
	eg.Go(func() error {
		b, err := k.Balances(conf)
		if err != nil {
			return fmt.Errorf("k balances: %w", err)
		}
		m[model.ExchangeTypeKu] = b
		return nil
	})
	eg.Go(func() error {
		b, err := g.Balances(conf)
		if err != nil {
			return fmt.Errorf("g balances: %w", err)
		}
		m[model.ExchangeTypeGa] = b
		return nil
	})
	eg.Go(func() error {
		b, err := c.Balances()
		if err != nil {
			return fmt.Errorf("c balances: %w", err)
		}
		m[model.ExchangeTypeCo] = b
		return nil
	})

	if err := eg.Wait(); err != nil {
		fmt.Println(err)
		return nil
	}

	return m
}

func Book(c *config.Config) {
	bb := GatherBalancesP(c)

	for e, b := range bb {
		if b.XCH.IsZero() && b.USDT.IsZero() {
			fmt.Printf("warning: %v balances are zero\n", e.String())
		}
	}

	fmt.Println(bb)

	a, b := GatherBooksP()

	as, bs, totalTradeXCH, gain, withdrawUSDT, withdrawXCH, profit, totalBuyUSDT, totalSellUSDT, totalBuyXCH, totalSellXCH := arbo(a, b, bb, c)
	template := `Buy $ %v , XCH %v / Sell $ %v , XCH %v ; Asks %v - %v / Bids %v - %v ; trade %v XCH (g %v - %v XCH - %v USDT = p %v)`
	msg := fmt.Sprintf(template,
		totalBuyUSDT, totalBuyXCH, totalSellUSDT, totalSellXCH, a[0].Price, as.LastPrice, b[0].Price, bs.LastPrice, totalTradeXCH, gain, withdrawXCH, withdrawUSDT, profit)

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

	as, bs, totalTradeXCH, gain, withdrawUSDT, withdrawXCH, profit, totalBuyUSDT, totalSellUSDT, totalBuyXCH, totalSellXCH = arbo(a, b, ignoreBalances, c)

	msg = fmt.Sprintf(template,
		totalBuyUSDT, totalBuyXCH, totalSellUSDT, totalSellXCH, a[0].Price, as.LastPrice, b[0].Price, bs.LastPrice, totalTradeXCH, gain, withdrawXCH, withdrawUSDT, profit)

	fmt.Println("when ignoring balances:")
	fmt.Println(msg)
}

func arbo(a, b []model.Order, balances map[model.ExchangeType]model.Balances, c *config.Config) (side, side, decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal, map[model.ExchangeType]decimal.Decimal, map[model.ExchangeType]decimal.Decimal, map[model.ExchangeType]decimal.Decimal, map[model.ExchangeType]decimal.Decimal) {
	totalTradeXCH := decimal.Zero
	totalBuyUSDT := make(map[model.ExchangeType]decimal.Decimal, len(model.ExchangeTypes))
	totalSellUSDT := make(map[model.ExchangeType]decimal.Decimal, len(model.ExchangeTypes))
	totalBuyXCH := make(map[model.ExchangeType]decimal.Decimal, len(model.ExchangeTypes))
	totalSellXCH := make(map[model.ExchangeType]decimal.Decimal, len(model.ExchangeTypes))
	for _, t := range model.ExchangeTypes {
		totalBuyUSDT[t] = decimal.Zero
		totalSellUSDT[t] = decimal.Zero
		totalBuyXCH[t] = decimal.Zero
		totalSellXCH[t] = decimal.Zero
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
		aa := a[as.I]
		bb := b[bs.I]

		// no match
		if aa.EffectivePrice.GreaterThanOrEqual(bb.EffectivePrice) {
			break
		}

		// deferred update to prevent running off end of array
		for _, s := range sides {
			if s.Move {
				s.HeadAmount = s.Book[s.I].Amount
				s.Move = false
			}
		}

		// arb
		// consider balances (allowances)
		buyAllowanceUSDT := balances[aa.Ex].USDT.Sub(totalBuyUSDT[aa.Ex])
		as.HeadAllowance = buyAllowanceUSDT.Div(aa.EffectivePrice).RoundDown(3)
		bs.HeadAllowance = balances[bb.Ex].XCH.Sub(totalSellXCH[bb.Ex]).Mul(decimal.NewFromInt(1).Sub(fees[bb.Ex].MakerTakerRatio))
		tradeAmount := decimal.Min(as.HeadAmount, bs.HeadAmount, as.HeadAllowance, bs.HeadAllowance)

		for _, s := range sides {
			if s.HeadAmount.Equal(tradeAmount) || s.HeadAllowance.Equal(tradeAmount) {
				s.Move = true
			}
			s.HeadAmount = s.HeadAmount.Sub(tradeAmount)
		}

		// trade executes internally
		totalTradeXCH = totalTradeXCH.Add(tradeAmount)
		gain = gain.Add(tradeAmount.Mul(bb.EffectivePrice.Sub(aa.EffectivePrice)))
		totalBuyUSDT[aa.Ex] = totalBuyUSDT[aa.Ex].Add(aa.EffectivePrice.Mul(tradeAmount))
		totalSellUSDT[bb.Ex] = totalSellUSDT[bb.Ex].Add(bb.EffectivePrice.Mul(tradeAmount))
		totalBuyXCH[aa.Ex] = totalBuyXCH[aa.Ex].Add(tradeAmount)
		totalSellXCH[bb.Ex] = totalSellXCH[bb.Ex].Add(tradeAmount)

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

	return *as, *bs, totalTradeXCH, gain, withdrawUSDT, withdrawXCH, profit, totalBuyUSDT, totalSellUSDT, totalBuyXCH, totalSellXCH
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
