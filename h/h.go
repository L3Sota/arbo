package main

import (
	"fmt"
	"log/slog"

	"github.com/linstohu/nexapi/htx/spot/marketws"
)

func main() {
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
