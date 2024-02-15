package h

import (
	"fmt"
	"log/slog"
	"strconv"

	arboconfig "github.com/L3Sota/arbo/arb/config"
	"github.com/L3Sota/arbo/arb/model"
	"github.com/huobirdcenter/huobi_golang/config"
	"github.com/huobirdcenter/huobi_golang/pkg/client"
	"github.com/huobirdcenter/huobi_golang/pkg/model/market"
	"github.com/linstohu/nexapi/htx/spot/marketws"
	"github.com/shopspring/decimal"
)

var (
	Fees = model.Fees{
		MakerTakerRatio:    decimal.RequireFromString("0.0017"), // 0.17%
		WithdrawalFlatXCH:  decimal.RequireFromString("0.0005"),
		WithdrawalFlatUSDT: decimal.NewFromInt(1), // TRC20
	}
	AskAddition  = decimal.NewFromInt(1).Add(Fees.MakerTakerRatio)
	BidReduction = decimal.NewFromInt(1).Sub(Fees.MakerTakerRatio)

	mc *client.MarketClient
	ac *client.AccountClient
)

func LoadClient(conf *arboconfig.Config) {
	mc = new(client.MarketClient).Init(config.Host)
	ac = new(client.AccountClient).Init(conf.HKey, conf.HSec, config.Host)
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

func Balances() (b model.Balances, err error) {
	accs, err := ac.GetAccountInfo()
	if err != nil {
		panic(err)
	}
	var id string
	for _, a := range accs {
		if a.Type == "spot" {
			id = strconv.FormatInt(a.Id, 10)
		}
	}

	a, err := ac.GetAccountBalance(id)
	if err != nil {
		panic(err)
	}

	for _, aa := range a.List {
		if aa.Type == "trade" {
			switch aa.Currency {
			case "usdt":
				b.USDT = decimal.RequireFromString(aa.Balance)
			case "xch":
				b.XCH = decimal.RequireFromString(aa.Balance)
			}
		}
	}

	return b, nil
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
