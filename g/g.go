package g

import (
	"context"
	"fmt"

	"github.com/L3Sota/arbo/arb/model"
	"github.com/gateio/gateapi-go/v6"
	"github.com/shopspring/decimal"
)

func G() ([]model.Order, []model.Order) {
	client := gateapi.NewAPIClient(gateapi.NewConfiguration())
	// uncomment the next line if your are testing against testnet
	// client.ChangeBasePath("https://fx-api-testnet.gateio.ws/api/v4")
	ctx := context.WithValue(context.Background(),
		gateapi.ContextGateAPIV4,
		gateapi.GateAPIV4{
			Key:    "YOUR_API_KEY",
			Secret: "YOUR_API_SECRET",
		},
	)

	result, _, err := client.SpotApi.ListOrderBook(ctx, "XCH_USDT", nil)
	if err != nil {
		if e, ok := err.(gateapi.GateAPIError); ok {
			fmt.Printf("gate api error: %s\n", e.Error())
		} else {
			fmt.Printf("generic error: %s\n", err.Error())
		}
		return nil, nil
	}
	a := make([]model.Order, 0, len(result.Asks))
	for _, ask := range result.Asks {
		a = append(a, model.Order{
			Ex:     model.Ga,
			Price:  decimal.RequireFromString(ask[0]),
			Amount: decimal.RequireFromString(ask[1]),
		})
	}
	b := make([]model.Order, 0, len(result.Bids))
	for _, bid := range result.Bids {
		b = append(b, model.Order{
			Ex:     model.Ga,
			Price:  decimal.RequireFromString(bid[0]),
			Amount: decimal.RequireFromString(bid[1]),
		})
	}

	return a, b
}
