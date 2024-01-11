package k

import (
	"fmt"

	"github.com/Kucoin/kucoin-go-sdk"
	"github.com/L3Sota/arbo/arb/model"
	"github.com/shopspring/decimal"
)

var (
	Fees = model.Fees{
		MakerTakerRatio:    decimal.RequireFromString("0.001"), // 0.1%
		WithdrawalFlatXCH:  decimal.RequireFromString("0.132"),
		WithdrawalFlatUSDT: decimal.RequireFromString("0.8"), // SOL
	}
	AskAddition  = decimal.NewFromInt(1).Add(Fees.MakerTakerRatio)
	BidReduction = decimal.NewFromInt(1).Sub(Fees.MakerTakerRatio)
)

func K() ([]model.Order, []model.Order) {
	s := kucoin.NewApiService()

	resp, err := s.AggregatedPartOrderBook("XCH-USDT", 100)
	if err != nil {
		fmt.Print(err)
		return nil, nil
	}

	var o struct {
		Time     int64
		Sequence string
		Asks     [][]string
		Bids     [][]string
	}
	if err := resp.ReadData(&o); err != nil {
		fmt.Print(err)
		return nil, nil
	}

	a := make([]model.Order, 0, len(o.Asks))
	for _, ask := range o.Asks {
		o := model.Order{
			Ex:     model.Ku,
			Price:  decimal.RequireFromString(ask[0]),
			Amount: decimal.RequireFromString(ask[1]),
		}
		o.EffectivePrice = o.Price.Mul(AskAddition).RoundUp(2)
		a = append(a, o)
	}
	b := make([]model.Order, 0, len(o.Bids))
	for _, bid := range o.Bids {
		o := model.Order{
			Ex:     model.Ku,
			Price:  decimal.RequireFromString(bid[0]),
			Amount: decimal.RequireFromString(bid[1]),
		}
		o.EffectivePrice = o.Price.Mul(BidReduction).RoundDown(2)
		b = append(b, o)
	}

	return a, b
}
