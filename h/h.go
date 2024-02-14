package h

import (
	"fmt"
	"log/slog"

	"github.com/L3Sota/arbo/arb/model"
	"github.com/huobirdcenter/huobi_golang/config"
	"github.com/huobirdcenter/huobi_golang/pkg/client"
	"github.com/huobirdcenter/huobi_golang/pkg/model/market"
	"github.com/linstohu/nexapi/htx/spot/marketws"
	"github.com/shopspring/decimal"
)

var (
	// TODO
	AskAddition  = decimal.NewFromInt(1)
	BidReduction = decimal.NewFromInt(1)

	mc *client.MarketClient
)

func LoadClient() {
	mc = new(client.MarketClient).Init(config.Host)
}

func Book() ([]model.Order, []model.Order, error) {
	o, err := mc.GetDepth("xchusdt", "step0", market.GetDepthOptionalRequest{})
	if err != nil {
		return nil, nil, err
	}

	a := make([]model.Order, 0, len(o.Asks))
	for _, ask := range o.Asks {
		o := model.Order{
			Ex:     model.ExchangeTypeHu,
			Price:  ask[0],
			Amount: ask[1],
		}
		o.EffectivePrice = o.Price.Mul(AskAddition)
		a = append(a, o)
	}
	b := make([]model.Order, 0, len(o.Bids))
	for _, bid := range o.Bids {
		o := model.Order{
			Ex:     model.ExchangeTypeHu,
			Price:  bid[0],
			Amount: bid[1],
		}
		o.EffectivePrice = o.Price.Mul(BidReduction)
		b = append(b, o)
	}

	return a, b, nil
}

func WSTest() {
	// c := &client.MarketClient{}
	// c.Init(config.Host)

	// d, err := c.GetDepth("xchusdt", "step0", market.GetDepthOptionalRequest{})
	// if err != nil {
	// 	fmt.Print(err)
	// 	return
	// }

	// fmt.Printf("d: %v\n", d)

	nex, err := marketws.NewMarketWsClient(&marketws.MarketWsClientCfg{
		BaseURL:       "wss://api.huobi.pro/ws",
		AutoReconnect: true,
		Logger:        slog.Default(),
	})

	if err != nil {
		fmt.Print(err)
		return
	}
	s, err := nex.GetDepthTopic(&marketws.DepthTopicParam{
		Symbol: "xchusdt",
		Type:   "step0",
	})

	if err != nil {
		fmt.Print(err)
		return
	}

	fmt.Print(s)

	if err := nex.Open(); err != nil {
		fmt.Print(err)
		return
	}
	if err := nex.Subscribe(s); err != nil {
		fmt.Print(err)
		return
	}

	if err := nex.Close(); err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println("done")
}
