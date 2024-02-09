package g

import (
	"context"
	"fmt"

	"github.com/L3Sota/arbo/arb/config"
	"github.com/L3Sota/arbo/arb/model"
	"github.com/gateio/gateapi-go/v6"
	"github.com/shopspring/decimal"
)

var (
	Fees = model.Fees{
		MakerTakerRatio:    decimal.RequireFromString("0.002"),  // 0.2%
		WithdrawalFlatXCH:  decimal.RequireFromString("0.0145"), // variable?
		WithdrawalFlatUSDT: decimal.RequireFromString("0.5"),    // SOL
	}
	AskAddition  = decimal.NewFromInt(1).Add(Fees.MakerTakerRatio)
	BidReduction = decimal.NewFromInt(1).Sub(Fees.MakerTakerRatio)

	client *gateapi.APIClient
)

func Book() ([]model.Order, []model.Order, error) {
	// uncomment the next line if your are testing against testnet
	// client.ChangeBasePath("https://fx-api-testnet.gateio.ws/api/v4")

	o, _, err := client.SpotApi.ListOrderBook(context.Background(), "XCH_USDT", nil)
	if err != nil {
		if e, ok := err.(gateapi.GateAPIError); ok {
			fmt.Printf("gate api error: %s\n", e.Error())
		} else {
			fmt.Printf("generic error: %s\n", err.Error())
		}
		return nil, nil, err
	}

	a := make([]model.Order, 0, len(o.Asks))
	for _, ask := range o.Asks {
		o := model.Order{
			Ex:     model.ExchangeTypeGa,
			Price:  decimal.RequireFromString(ask[0]),
			Amount: decimal.RequireFromString(ask[1]),
		}
		o.EffectivePrice = o.Price.Mul(AskAddition)
		a = append(a, o)
	}
	b := make([]model.Order, 0, len(o.Bids))
	for _, bid := range o.Bids {
		o := model.Order{
			Ex:     model.ExchangeTypeGa,
			Price:  decimal.RequireFromString(bid[0]),
			Amount: decimal.RequireFromString(bid[1]),
		}
		o.EffectivePrice = o.Price.Mul(BidReduction)
		b = append(b, o)
	}

	return a, b, nil
}

func Balances(c *config.Config) (b model.Balances, err error) {
	client = gateapi.NewAPIClient(gateapi.NewConfiguration())

	ctx := context.WithValue(context.Background(),
		gateapi.ContextGateAPIV4,
		gateapi.GateAPIV4{
			Key:    c.GKey,
			Secret: c.GSec,
		},
	)

	a, _, err := client.SpotApi.ListSpotAccounts(ctx, nil)
	if err != nil {
		return b, err
	}

	for _, aa := range a {
		switch aa.Currency {
		case "USDT":
			b.USDT = decimal.RequireFromString(aa.Available)
		case "XCH":
			b.XCH = decimal.RequireFromString(aa.Available)
		}
	}

	return b, nil
}

func Buy(price, size decimal.Decimal, c *config.Config) (gateapi.Order, error) {
	ctx := context.WithValue(context.Background(),
		gateapi.ContextGateAPIV4,
		gateapi.GateAPIV4{
			Key:    c.GKey,
			Secret: c.GSec,
		},
	)

	// min order size 1 USDT
	o, _, err := client.SpotApi.CreateOrder(ctx, gateapi.Order{
		CurrencyPair: "XCH_USDT",
		Type:         "limit",
		Account:      "spot",
		Side:         "buy",
		Amount:       size.String(),  // Amount in XCH (base currency)
		Price:        price.String(), // Price in USDT (quote currency)
		TimeInForce:  "gtc",
	})

	return o, err
}

func Sell(price, size decimal.Decimal, c *config.Config) (gateapi.Order, error) {
	ctx := context.WithValue(context.Background(),
		gateapi.ContextGateAPIV4,
		gateapi.GateAPIV4{
			Key:    c.GKey,
			Secret: c.GSec,
		},
	)

	// min order size 1 USDT
	o, _, err := client.SpotApi.CreateOrder(ctx, gateapi.Order{
		CurrencyPair: "XCH_USDT",
		Type:         "limit",
		Account:      "spot",
		Side:         "sell",
		Amount:       size.String(),  // Amount in XCH (base currency)
		Price:        price.String(), // Price in USDT (quote currency)
		TimeInForce:  "gtc",
	})

	return o, err
}

// order: {Id:489126754641 Text:apiv4 AmendText:- CreateTime:1705482626 UpdateTime:1705482626 CreateTimeMs:1705482626977 UpdateTimeMs:1705482626977 Status:cancelled CurrencyPair:XCH_USDT Type:limit Account:spot Side:buy Amount:0.1 Price:20 TimeInForce:ioc Iceberg:0 AutoBorrow:false AutoRepay:false Left:0.1 FillPrice:0 FilledTotal:0 AvgDealPrice: Fee:0 FeeCurrency:XCH PointFee:0 GtFee:0 GtMakerFee:0 GtTakerFee:0 GtDiscount:false RebatedFee:0 RebatedFeeCurrency:USDT StpId:0 StpAct: FinishAs:ioc}
func OrderTest(c *config.Config) {
	client = gateapi.NewAPIClient(gateapi.NewConfiguration())

	o, err := Buy(decimal.NewFromInt(20), decimal.NewFromInt(1).Div(decimal.NewFromInt(10)), c)

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("order: %+v\n", o)
}

// {14541031 0.002 0.002 false 0 0 0.18 1 0.0005 0.00015 0.00016 -0.00015}
// ^ 0.2% maker taker
func QueryFee(c *config.Config) {
	client := gateapi.NewAPIClient(gateapi.NewConfiguration())

	ctx := context.WithValue(context.Background(),
		gateapi.ContextGateAPIV4,
		gateapi.GateAPIV4{
			Key:    c.GKey,
			Secret: c.GSec,
		},
	)

	fee, _, err := client.WalletApi.GetTradeFee(ctx, nil)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(fee)
}
