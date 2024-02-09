package m

import (
	"context"
	"log/slog"

	"github.com/L3Sota/arbo/arb/model"
	"github.com/linstohu/nexapi/mexc/spot/marketdata"
	"github.com/linstohu/nexapi/mexc/spot/marketdata/types"
	spotutils "github.com/linstohu/nexapi/mexc/spot/utils"
	"github.com/shopspring/decimal"
)

var (
	Fees = model.Fees{
		MakerTakerRatio:    decimal.RequireFromString("0.001"), // 0.1%
		WithdrawalFlatXCH:  decimal.RequireFromString("0.0005"),
		WithdrawalFlatUSDT: decimal.NewFromInt(1), // TRC20 ARB OP
	}
	AskAddition  = decimal.NewFromInt(1).Add(Fees.MakerTakerRatio)
	BidReduction = decimal.NewFromInt(1).Sub(Fees.MakerTakerRatio)
)

func Book() ([]model.Order, []model.Order, error) {
	nex, err := marketdata.NewSpotMarketDataClient(&spotutils.SpotClientCfg{
		BaseURL: "https://api.mexc.com/",
		Logger:  slog.Default(),
	})
	if err != nil {
		return nil, nil, err
	}
	o, err := nex.GetOrderbook(context.TODO(), types.GetOrderbookParams{
		Symbol: "XCHUSDT",
	})
	if err != nil {
		return nil, nil, err
	}

	a := make([]model.Order, 0, len(o.Asks))
	for _, ask := range o.Asks {
		o := model.Order{
			Ex:     model.ExchangeTypeMe,
			Price:  decimal.RequireFromString(ask[0]),
			Amount: decimal.RequireFromString(ask[1]),
		}
		o.EffectivePrice = o.Price.Mul(AskAddition)
		a = append(a, o)
	}
	b := make([]model.Order, 0, len(o.Bids))
	for _, bid := range o.Bids {
		o := model.Order{
			Ex:     model.ExchangeTypeMe,
			Price:  decimal.RequireFromString(bid[0]),
			Amount: decimal.RequireFromString(bid[1]),
		}
		o.EffectivePrice = o.Price.Mul(BidReduction)
		b = append(b, o)
	}

	return a, b, nil
}
