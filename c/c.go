package c

import (
	"encoding/json"
	"fmt"

	"github.com/L3Sota/arbo/arb/model"
	"github.com/shopspring/decimal"
	"gopkg.in/resty.v1"
)

var (
	Fees = model.Fees{
		MakerTakerRatio:    decimal.RequireFromString("0.002"), // 0.2%
		WithdrawalFlatXCH:  decimal.RequireFromString("0.001"),
		WithdrawalFlatUSDT: decimal.RequireFromString("1.4"), // TRC20
	}

	AskAddition  = decimal.NewFromInt(1).Add(Fees.MakerTakerRatio)
	BidReduction = decimal.NewFromInt(1).Sub(Fees.MakerTakerRatio)
)

type book struct {
	Last string
	Time int64
	Asks [][]string
	Bids [][]string
}

func C() ([]model.Order, []model.Order) {
	client := resty.New()

	resp, err := client.R().Get("https://api.coinex.com/v1/market/depth?market=XCHUSDT&merge=0.01&limit=50")
	if err != nil {
		fmt.Println(err)
		return nil, nil
	}

	raw := &struct {
		Data book
	}{}

	if err := json.Unmarshal(resp.Body(), raw); err != nil {
		fmt.Println(err)
		return nil, nil
	}

	o := raw.Data

	a := make([]model.Order, 0, len(o.Asks))
	for _, ask := range o.Asks {
		o := model.Order{
			Ex:     model.Co,
			Price:  decimal.RequireFromString(ask[0]),
			Amount: decimal.RequireFromString(ask[1]),
		}
		o.EffectivePrice = o.Price.Mul(AskAddition).RoundUp(2)
		a = append(a, o)
	}
	b := make([]model.Order, 0, len(o.Bids))
	for _, bid := range o.Bids {
		o := model.Order{
			Ex:     model.Co,
			Price:  decimal.RequireFromString(bid[0]),
			Amount: decimal.RequireFromString(bid[1]),
		}
		o.EffectivePrice = o.Price.Mul(BidReduction).RoundDown(2)
		b = append(b, o)
	}

	return a, b
}
