package arb

import (
	"context"
	"fmt"
	"strings"

	"github.com/L3Sota/arbo/arb/config"
	"github.com/L3Sota/arbo/arb/model"
	"github.com/L3Sota/arbo/c"
	"github.com/L3Sota/arbo/g"
	"github.com/L3Sota/arbo/h"
	"github.com/L3Sota/arbo/k"
	"github.com/L3Sota/arbo/m"
	"github.com/gateio/gateapi-go/v6"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/errgroup"
)

type side struct {
	Book          []model.Order
	I             int
	HeadAmount    decimal.Decimal
	HeadAllowance decimal.Decimal
	LastPrice     [model.ExchangeTypeMax]decimal.Decimal
	Move          bool
}

const (
	profitTemplate = "p %v"
	buyTemplate    = "買@%v $%v, ¢%v [≦ $%v]"
	sellTemplate   = "売@%v $%v, ¢%v [≧ $%v]"
	miscTemplate   = "t ¢%v, (g %v - ¢%v - $%v)"
)

var (
	fees = [model.ExchangeTypeMax]model.Fees{
		m.Fees,
		k.Fees,
		h.Fees,
		c.Fees,
		g.Fees,
	}

	feeRatioCapUSDT = decimal.NewFromInt(3000)

	big        = decimal.New(1, 10)
	bigBalance = model.Balances{
		XCH:  big,
		USDT: big,
	}
	ignoreBalances = [model.ExchangeTypeMax]model.Balances{
		bigBalance,
		bigBalance,
		bigBalance,
		bigBalance,
		bigBalance,
	}

	bb [model.ExchangeTypeMax]model.Balances

	USDTMinSizes = [model.ExchangeTypeMax]decimal.Decimal{
		decimal.Zero,
		decimal.RequireFromString("0.1"),
		decimal.NewFromInt(10),
		decimal.Zero,
		decimal.NewFromInt(1),
	}
	XCHMinSizes = [model.ExchangeTypeMax]decimal.Decimal{
		decimal.Zero,
		decimal.RequireFromString("0.001"),
		decimal.Zero,
		decimal.RequireFromString("0.05"),
		decimal.Zero,
	}
)

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
	ha, hb, err := h.Book()
	if err != nil {
		return nil, nil
	}
	ca, cb, err := c.Book()
	if err != nil {
		return nil, nil
	}
	ga, gb, err := g.Book()
	if err != nil {
		return nil, nil
	}

	a := merge(true, ma, ka, ha, ca, ga)
	b := merge(false, mb, kb, hb, cb, gb)

	return a, b
}

func GatherBooksP() ([]model.Order, []model.Order, error) {
	var ma, ka, ha, ca, ga, mb, kb, hb, cb, gb []model.Order
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
		a, b, err := h.Book()
		if err != nil {
			return fmt.Errorf("h book: %w", err)
		}
		ha = a
		hb = b
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
	eg.Go(func() error {
		a, b, err := g.Book()
		if err != nil {
			return fmt.Errorf("g book: %w", err)
		}
		ga = a
		gb = b
		return nil
	})
	if err := eg.Wait(); err != nil {
		return nil, nil, err
	}

	a := merge(true, ma, ka, ha, ca, ga)
	b := merge(false, mb, kb, hb, cb, gb)

	return a, b, nil
}

func GatherBalancesP(conf *config.Config) (m [model.ExchangeTypeMax]model.Balances, err error) {
	eg, _ := errgroup.WithContext(context.Background())
	// eg.Go(func() error {
	// 	m[model.ExchangeTypeMe] = model.Balances{}
	// 	return nil
	// })
	eg.Go(func() error {
		b, err := k.Balances()
		if err != nil {
			return fmt.Errorf("k balances: %w", err)
		}
		m[model.ExchangeTypeKu] = b
		return nil
	})
	eg.Go(func() error {
		b, err := h.Balances()
		if err != nil {
			return fmt.Errorf("h balances: %w", err)
		}
		m[model.ExchangeTypeHu] = b
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
	eg.Go(func() error {
		b, err := g.Balances(conf)
		if err != nil {
			return fmt.Errorf("g balances: %w", err)
		}
		m[model.ExchangeTypeGa] = b
		return nil
	})

	if err := eg.Wait(); err != nil {
		return m, err
	}

	return m, nil
}

func Book(gatherBalances bool, conf *config.Config) (bool, []string, error) {
	messages := make([]string, 0, 2)
	var (
		msg    string
		traded bool
	)

	if gatherBalances {
		balances, err := GatherBalancesP(conf)
		if err != nil {
			return false, nil, fmt.Errorf("balances: %w", err)
		}
		bb = balances
	}

	for e, b := range bb {
		if b.XCH.IsZero() && b.USDT.IsZero() {
			fmt.Printf("warning: %v balances are zero\n", model.ExchangeType(e).String())
		}
	}

	fmt.Println(bb)

	a, b, err := GatherBooksP()
	if err != nil {
		return false, nil, fmt.Errorf("books: %w", err)
	}

	as, bs, totalTradeXCH, gain, withdrawUSDT, withdrawXCH, profit, totalBuyUSDT, totalSellUSDT, totalBuyXCH, totalSellXCH := arbo(a, b, bb, conf)

	if conf.ExecuteTrades && profit.IsPositive() {
		kOrderID, hOrderID, cOrder, gOrder, err := trade(totalBuyUSDT, totalSellUSDT, totalBuyXCH, totalSellXCH, as.LastPrice, bs.LastPrice, conf)
		if err != nil {
			return false, nil, fmt.Errorf("trade: %w", err)
		}

		fmt.Printf("k: %v\nh: %v\nc: %+v\ng: %+v\n", kOrderID, hOrderID, cOrder, gOrder)
		traded = true

		if kOrderID == "" && hOrderID == "" && cOrder == nil && gOrder == nil {
			traded = false
		}

		if conf.PEnable {
			trades := []string{fmt.Sprintf(profitTemplate, profit)}
			for e, b := range totalBuyXCH {
				if b.IsPositive() {
					trades = append(trades, fmt.Sprintf(buyTemplate, model.ExchangeType(e).String(), totalBuyUSDT[e], b, as.LastPrice[e]))
				}
			}
			for e, s := range totalSellXCH {
				if s.IsPositive() {
					trades = append(trades, fmt.Sprintf(sellTemplate, model.ExchangeType(e).String(), totalSellUSDT[e], s, bs.LastPrice[e]))
				}
			}
			trades = append(trades, fmt.Sprintf(miscTemplate, totalTradeXCH, gain, withdrawXCH, withdrawUSDT))
			if !traded {
				trades = append(trades, "(skipped: below min order threshold)")
			}

			msg = strings.Join(trades, "\n")
			messages = append(messages, msg)
		}
	}

	depth := "ex eff pr amt\n===\n%v\n---\n%v\n===\n"
	increase := 5
	if profit.IsZero() {
		increase = 0
	}
	aDepth := make([]string, 0, as.I+increase)
	for _, ask := range a[:as.I+increase] {
		aDepth = append(aDepth, strings.Join([]string{ask.Ex.String(), ask.EffectivePrice.StringFixed(4), ask.Price.StringFixed(4), ask.Amount.String()}, " "))
	}
	bDepth := make([]string, 0, bs.I+increase)
	for _, bid := range b[:bs.I+increase] {
		bDepth = append(bDepth, strings.Join([]string{bid.Ex.String(), bid.EffectivePrice.StringFixed(4), bid.Price.StringFixed(4), bid.Amount.String()}, " "))
	}

	if len(aDepth) > 0 {
		fmt.Printf(depth, strings.Join(aDepth, "\n"), strings.Join(bDepth, "\n"))
	}

	trades := []string{fmt.Sprintf(profitTemplate, profit)}
	for e, b := range totalBuyXCH {
		if b.IsPositive() {
			trades = append(trades, fmt.Sprintf(buyTemplate, model.ExchangeType(e).String(), totalBuyUSDT[e], b, as.LastPrice[e]))
		}
	}
	for e, s := range totalSellXCH {
		if s.IsPositive() {
			trades = append(trades, fmt.Sprintf(sellTemplate, model.ExchangeType(e).String(), totalSellUSDT[e], s, bs.LastPrice[e]))
		}
	}
	trades = append(trades, fmt.Sprintf(miscTemplate, totalTradeXCH, gain, withdrawXCH, withdrawUSDT))
	msg = strings.Join(trades, "\n")
	fmt.Println(msg)

	as, bs, totalTradeXCH, gain, withdrawUSDT, withdrawXCH, profit, totalBuyUSDT, totalSellUSDT, totalBuyXCH, totalSellXCH = arbo(a, b, ignoreBalances, conf)

	trades = []string{fmt.Sprintf(profitTemplate, profit)}
	for e, b := range totalBuyXCH {
		if b.IsPositive() {
			trades = append(trades, fmt.Sprintf(buyTemplate, model.ExchangeType(e).String(), totalBuyUSDT[e], b, as.LastPrice[e]))
		}
	}
	for e, s := range totalSellXCH {
		if s.IsPositive() {
			trades = append(trades, fmt.Sprintf(sellTemplate, model.ExchangeType(e).String(), totalSellUSDT[e], s, bs.LastPrice[e]))
		}
	}
	trades = append(trades, fmt.Sprintf(miscTemplate, totalTradeXCH, gain, withdrawXCH, withdrawUSDT))
	msg2 := strings.Join(trades, "\n")

	if msg2 != msg {
		fmt.Println("when ignoring balances:")
		fmt.Println(msg2)
	}

	if len(messages) > 0 {
		messages = append(messages, "when ignoring balances: "+msg2)
	}

	return traded, messages, nil
}

func arbo(a, b []model.Order, balances [model.ExchangeTypeMax]model.Balances, c *config.Config) (side, side, decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal, decimal.Decimal, [model.ExchangeTypeMax]decimal.Decimal, [model.ExchangeTypeMax]decimal.Decimal, [model.ExchangeTypeMax]decimal.Decimal, [model.ExchangeTypeMax]decimal.Decimal) {
	totalTradeXCH := decimal.Zero
	totalBuyUSDT := [model.ExchangeTypeMax]decimal.Decimal{}
	totalSellUSDT := [model.ExchangeTypeMax]decimal.Decimal{}
	totalBuyXCH := [model.ExchangeTypeMax]decimal.Decimal{}
	totalSellXCH := [model.ExchangeTypeMax]decimal.Decimal{}
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
			s.LastPrice = [model.ExchangeTypeMax]decimal.Decimal{s.Book[0].Price, s.Book[0].Price, s.Book[0].Price, s.Book[0].Price, s.Book[0].Price}
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
			s.LastPrice[s.Book[s.I].Ex] = s.Book[s.I].Price
			if s.Move {
				s.I++
			}
		}
	}

	// TODO walk back (handle > 1 count), also check for unprofitable exchanges (subtract withdrawal fees)
	for e, bXCH := range totalBuyXCH {
		mXCH := XCHMinSizes[e]
		if !mXCH.IsZero() && bXCH.IsPositive() && bXCH.LessThan(mXCH) {
			sellCount := 0
			lastIndex := 0
			for i, sXCH := range totalSellXCH {
				if sXCH.IsPositive() {
					sellCount++
					lastIndex = i
				}
			}
			if sellCount == 1 {
				totalSellXCH[lastIndex] = totalSellXCH[lastIndex].Sub(bXCH)
				totalSellUSDT[lastIndex] = totalSellUSDT[lastIndex].Sub(totalBuyUSDT[e]) // approximate
				totalBuyXCH[e] = decimal.Zero
				totalBuyUSDT[e] = decimal.Zero
			}
			continue
		}
		bUSDT := totalBuyUSDT[e]
		mUSDT := USDTMinSizes[e]
		if !mUSDT.IsZero() && bUSDT.IsPositive() && bUSDT.LessThan(mUSDT) {
			sellCount := 0
			lastIndex := 0
			for i, sXCH := range totalSellXCH {
				if sXCH.IsPositive() {
					sellCount++
					lastIndex = i
				}
			}
			if sellCount == 1 {
				totalSellXCH[lastIndex] = totalSellXCH[lastIndex].Sub(bXCH)
				totalSellUSDT[lastIndex] = totalSellUSDT[lastIndex].Sub(totalBuyUSDT[e]) // approximate
				totalBuyXCH[e] = decimal.Zero
				totalBuyUSDT[e] = decimal.Zero
			}
			continue
		}
	}

	for e, sXCH := range totalSellXCH {
		mXCH := XCHMinSizes[e]
		if !mXCH.IsZero() && sXCH.IsPositive() && sXCH.LessThan(mXCH) {
			buyCount := 0
			lastIndex := 0
			for i, bXCH := range totalBuyXCH {
				if bXCH.IsPositive() {
					buyCount++
					lastIndex = i
				}
			}
			if buyCount == 1 {
				totalBuyXCH[lastIndex] = totalBuyXCH[lastIndex].Sub(sXCH)
				totalBuyUSDT[lastIndex] = totalBuyUSDT[lastIndex].Sub(totalBuyUSDT[e]) // approximate
				totalSellXCH[e] = decimal.Zero
				totalSellUSDT[e] = decimal.Zero
			}
			continue
		}
		sUSDT := totalSellUSDT[e]
		mUSDT := USDTMinSizes[e]
		if !mUSDT.IsZero() && sUSDT.IsPositive() && sUSDT.LessThan(mUSDT) {
			buyCount := 0
			lastIndex := 0
			for i, bXCH := range totalBuyXCH {
				if bXCH.IsPositive() {
					buyCount++
					lastIndex = i
				}
			}
			if buyCount == 1 {
				totalBuyXCH[lastIndex] = totalBuyXCH[lastIndex].Sub(sXCH)
				totalBuyUSDT[lastIndex] = totalBuyUSDT[lastIndex].Sub(totalBuyUSDT[e]) // approximate
				totalSellXCH[e] = decimal.Zero
				totalSellUSDT[e] = decimal.Zero
			}
			continue
		}
	}

	// buy XCH -> withdraw XCH
	withdrawXCH := decimal.Zero
	withdrawXCHAsUSDT := decimal.Zero
	for e, b := range totalBuyUSDT {
		if b.IsPositive() {
			ratio := decimal.NewFromInt(1)
			if b.LessThan(feeRatioCapUSDT) {
				ratio = b.Div(feeRatioCapUSDT)
			}
			fee := fees[e].WithdrawalFlatXCH.Mul(ratio)
			withdrawXCH = withdrawXCH.Add(fee)
			withdrawXCHAsUSDT = withdrawXCHAsUSDT.Add(fee.Mul(bs.LastPrice[e]))
		}
	}
	// sell XCH -> withdraw USDT
	withdrawUSDT := decimal.Zero
	for e, s := range totalSellUSDT {
		if s.IsPositive() {
			ratio := decimal.NewFromInt(1)
			if s.LessThan(feeRatioCapUSDT) {
				ratio = s.Div(feeRatioCapUSDT)
			}
			withdrawUSDT = withdrawUSDT.Add(fees[e].WithdrawalFlatUSDT.Mul(ratio))
		}
	}

	profit := gain.Sub(withdrawUSDT).Sub(withdrawXCHAsUSDT)

	return *as, *bs, totalTradeXCH, gain, withdrawUSDT, withdrawXCH, profit, totalBuyUSDT, totalSellUSDT, totalBuyXCH, totalSellXCH
}

func trade(totalBuyUSDT, totalSellUSDT, totalBuyXCH, totalSellXCH, askPrices, bidPrices [model.ExchangeTypeMax]decimal.Decimal, conf *config.Config) (string, string, *c.OrderResp, *gateapi.Order, error) {
	var (
		kOrderID string
		hOrderID string
		cOrder   *c.OrderResp
		gOrder   gateapi.Order
	)

	for e, bXCH := range totalBuyXCH {
		mXCH := XCHMinSizes[e]
		if !mXCH.IsZero() && bXCH.IsPositive() && bXCH.LessThan(mXCH) {
			return "", "", nil, nil, nil
		}
		bUSDT := totalBuyUSDT[e]
		mUSDT := USDTMinSizes[e]
		if !mUSDT.IsZero() && bUSDT.IsPositive() && bUSDT.LessThan(mUSDT) {
			return "", "", nil, nil, nil
		}
	}

	for e, sXCH := range totalSellXCH {
		mXCH := XCHMinSizes[e]
		if !mXCH.IsZero() && sXCH.IsPositive() && sXCH.LessThan(mXCH) {
			return "", "", nil, nil, nil
		}
		sUSDT := totalSellUSDT[e]
		mUSDT := USDTMinSizes[e]
		if !mUSDT.IsZero() && sUSDT.IsPositive() && sUSDT.LessThan(mUSDT) {
			return "", "", nil, nil, nil
		}
	}

	eg, _ := errgroup.WithContext(context.Background())

	// if totalBuyXCH[model.ExchangeTypeMe].IsPositive() {
	// 	eg.Go(func() error {
	// 		// buy
	// 		return nil
	// 	})
	// } else if totalSellXCH[model.ExchangeTypeMe].IsPositive() {
	// 	eg.Go(func() error {
	// 		// sell
	// 		return nil
	// 	})
	// }
	if totalBuyXCH[model.ExchangeTypeKu].IsPositive() {
		eg.Go(func() error {
			oid, err := k.Buy(askPrices[model.ExchangeTypeKu], totalBuyXCH[model.ExchangeTypeKu])
			if err != nil {
				return fmt.Errorf("k buy: %w", err)
			}
			kOrderID = oid
			return nil
		})
	} else if totalSellXCH[model.ExchangeTypeKu].IsPositive() {
		eg.Go(func() error {
			oid, err := k.Sell(bidPrices[model.ExchangeTypeKu], totalSellXCH[model.ExchangeTypeKu])
			if err != nil {
				return fmt.Errorf("k sell: %w", err)
			}
			kOrderID = oid
			return nil
		})
	}
	if totalBuyXCH[model.ExchangeTypeHu].IsPositive() {
		eg.Go(func() error {
			oid, err := h.Buy(askPrices[model.ExchangeTypeHu], totalBuyXCH[model.ExchangeTypeHu])
			if err != nil {
				return fmt.Errorf("h buy: %w", err)
			}
			hOrderID = oid
			return nil
		})
	} else if totalSellXCH[model.ExchangeTypeHu].IsPositive() {
		eg.Go(func() error {
			oid, err := h.Sell(bidPrices[model.ExchangeTypeHu], totalSellXCH[model.ExchangeTypeHu])
			if err != nil {
				return fmt.Errorf("h sell: %w", err)
			}
			hOrderID = oid
			return nil
		})
	}
	if totalBuyXCH[model.ExchangeTypeCo].IsPositive() {
		eg.Go(func() error { // TODO from here errors
			resp, err := c.Buy(askPrices[model.ExchangeTypeCo], totalBuyXCH[model.ExchangeTypeCo])
			if err != nil {
				return fmt.Errorf("c buy: %w", err)
			}
			cOrder = resp
			return nil
		})
	} else if totalSellXCH[model.ExchangeTypeCo].IsPositive() {
		eg.Go(func() error {
			resp, err := c.Sell(bidPrices[model.ExchangeTypeCo], totalSellXCH[model.ExchangeTypeCo])
			if err != nil {
				return fmt.Errorf("c sell: %w", err)
			}
			cOrder = resp
			return nil
		})
	}
	if totalBuyXCH[model.ExchangeTypeGa].IsPositive() {
		eg.Go(func() error {
			resp, err := g.Buy(askPrices[model.ExchangeTypeGa], totalBuyXCH[model.ExchangeTypeGa], conf)
			if err != nil {
				return fmt.Errorf("g buy: %w", err)
			}
			gOrder = resp
			return nil
		})
	} else if totalSellXCH[model.ExchangeTypeGa].IsPositive() {
		eg.Go(func() error {
			resp, err := g.Sell(bidPrices[model.ExchangeTypeGa], totalSellXCH[model.ExchangeTypeGa], conf)
			if err != nil {
				return fmt.Errorf("g sell: %w", err)
			}
			gOrder = resp
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return "", "", nil, nil, err
	}

	return kOrderID, hOrderID, cOrder, &gOrder, nil
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
