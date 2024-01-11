package m

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/L3Sota/arbo/arb/model"
	"github.com/linstohu/nexapi/mexc/spot/marketdata"
	"github.com/linstohu/nexapi/mexc/spot/marketdata/types"
	spotutils "github.com/linstohu/nexapi/mexc/spot/utils"
	"github.com/shopspring/decimal"
)

func M() ([]model.Order, []model.Order) {
	nex, err := marketdata.NewSpotMarketDataClient(&spotutils.SpotClientCfg{
		BaseURL: "https://api.mexc.com/",
		Logger:  slog.Default(),
	})
	if err != nil {
		fmt.Print(err)
		return nil, nil
	}
	o, err := nex.GetOrderbook(context.TODO(), types.GetOrderbookParams{
		Symbol: "XCHUSDT",
	})
	if err != nil {
		fmt.Print(err)
		return nil, nil
	}

	a := make([]model.Order, 0, len(o.Asks))
	for _, ka := range o.Asks {
		a = append(a, model.Order{
			Ex:     model.ME,
			Price:  decimal.RequireFromString(ka[0]),
			Amount: decimal.RequireFromString(ka[1]),
		})
	}
	b := make([]model.Order, 0, len(o.Bids))
	for _, kb := range o.Bids {
		b = append(b, model.Order{
			Ex:     model.ME,
			Price:  decimal.RequireFromString(kb[0]),
			Amount: decimal.RequireFromString(kb[1]),
		})
	}

	return a, b
}
