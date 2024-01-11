package k

import (
	"fmt"

	"github.com/Kucoin/kucoin-go-sdk"
	"github.com/L3Sota/arbo/arb/model"
	"github.com/shopspring/decimal"
)

func K() ([]model.Order, []model.Order) {
	s := kucoin.NewApiService()

	resp, err := s.AggregatedPartOrderBook("XCH-USDT", 100)
	if err != nil {
		fmt.Print(err)
		return nil, nil
	}

	var k struct {
		Time     int64
		Sequence string
		Asks     [][]string
		Bids     [][]string
	}
	if err := resp.ReadData(&k); err != nil {
		fmt.Print(err)
		return nil, nil
	}

	a := make([]model.Order, 0, len(k.Asks))
	for _, ka := range k.Asks {
		a = append(a, model.Order{
			Ex:     model.Ku,
			Price:  decimal.RequireFromString(ka[0]),
			Amount: decimal.RequireFromString(ka[1]),
		})
	}
	b := make([]model.Order, 0, len(k.Bids))
	for _, kb := range k.Bids {
		b = append(b, model.Order{
			Ex:     model.Ku,
			Price:  decimal.RequireFromString(kb[0]),
			Amount: decimal.RequireFromString(kb[1]),
		})
	}

	return a, b
}
