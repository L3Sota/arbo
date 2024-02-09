package k

import (
	"fmt"

	"github.com/Kucoin/kucoin-go-sdk"
	"github.com/L3Sota/arbo/arb/config"
	"github.com/L3Sota/arbo/arb/model"
	"github.com/google/uuid"
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

func Book() ([]model.Order, []model.Order, error) {
	s := kucoin.NewApiService()

	resp, err := s.AggregatedPartOrderBook("XCH-USDT", 100)
	if err != nil {
		return nil, nil, err
	}

	var o kucoin.PartOrderBookModel
	if err := resp.ReadData(&o); err != nil {
		return nil, nil, err
	}

	a := make([]model.Order, 0, len(o.Asks))
	for _, ask := range o.Asks {
		o := model.Order{
			Ex:     model.ExchangeTypeKu,
			Price:  decimal.RequireFromString(ask[0]),
			Amount: decimal.RequireFromString(ask[1]),
		}
		o.EffectivePrice = o.Price.Mul(AskAddition)
		a = append(a, o)
	}
	b := make([]model.Order, 0, len(o.Bids))
	for _, bid := range o.Bids {
		o := model.Order{
			Ex:     model.ExchangeTypeKu,
			Price:  decimal.RequireFromString(bid[0]),
			Amount: decimal.RequireFromString(bid[1]),
		}
		o.EffectivePrice = o.Price.Mul(BidReduction)
		b = append(b, o)
	}

	return a, b, nil
}

func Balances(c *config.Config) (b model.Balances, err error) {
	s := kucoin.NewApiService(
		kucoin.ApiKeyOption(c.KKey),
		kucoin.ApiKeyVersionOption(kucoin.ApiKeyVersionV2),
		kucoin.ApiPassPhraseOption(c.KPass),
		kucoin.ApiSecretOption(c.KSec),
	)

	resp, err := s.Accounts("", "")
	if err != nil {
		fmt.Println(err)
		return
	}

	var a kucoin.AccountsModel
	if err = resp.ReadData(&a); err != nil {
		return
	}

	for _, aa := range a {
		if aa.Type == "trade" {
			switch aa.Currency {
			case "USDT":
				b.USDT = decimal.RequireFromString(aa.Available)
			case "XCH":
				b.XCH = decimal.RequireFromString(aa.Available)
			}
		}
	}

	return b, nil
}

// {65a8928fcf1c7f00074b0ea7}
// {Id:65a8928fcf1c7f00074b0ea7 Symbol:XCH-USDT OpType:DEAL Type:limit Side:buy Price:20 Size:0.1 Funds:0 DealFunds:0 DealSize:0 Fee:0 FeeCurrency:USDT Stp: Stop: StopTriggered:false StopPrice:0 TimeInForce:IOC PostOnly:false Hidden:false IceBerg:false VisibleSize:0 CancelAfter:0 Channel:API ClientOid:d6c51d3e-2f72-4e38-a9a6-2afc14041e2e Remark: Tags: IsActive:false CancelExist:true CreatedAt:1705546383349 TradeType:TRADE}
func OrderTest(c *config.Config) {
	s := kucoin.NewApiService(
		kucoin.ApiKeyOption(c.KKey),
		kucoin.ApiKeyVersionOption(kucoin.ApiKeyVersionV2),
		kucoin.ApiPassPhraseOption(c.KPass),
		kucoin.ApiSecretOption(c.KSec),
	)

	resp, err := s.CreateOrder(&kucoin.CreateOrderModel{
		// BASE PARAMETERS
		ClientOid: uuid.New().String(),
		Side:      "buy",
		Symbol:    "XCH-USDT",
		Type:      "limit",

		// LIMIT ORDER PARAMETERS
		Price:       decimal.NewFromInt(20).String(),
		Size:        decimal.NewFromInt(1).Div(decimal.NewFromInt(10)).String(),
		TimeInForce: "IOC",
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	var o kucoin.CreateOrderResultModel
	if err := resp.ReadData(&o); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(o)

	resp, err = s.Order(o.OrderId)
	if err != nil {
		fmt.Println(err)
		return
	}

	var oo kucoin.OrderModel
	if err := resp.ReadData(&oo); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%+v\n", oo)
}

// [{XCH-USDT 0.001 0.001}]
func QueryFee(c *config.Config) {
	s := kucoin.NewApiService(
		kucoin.ApiKeyOption(c.KKey),
		kucoin.ApiKeyVersionOption(kucoin.ApiKeyVersionV2),
		kucoin.ApiPassPhraseOption(c.KPass),
		kucoin.ApiSecretOption(c.KSec),
	)

	resp, err := s.ActualFee("XCH-USDT")
	if err != nil {
		fmt.Println(err)
		return
	}

	var f kucoin.TradeFeesResultModel
	if err := resp.ReadData(&f); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(f)
}

/*
> http https://api.kucoin.com/api/v2/symbols | jq '.data | map(select(.symbol == "XCH-USDT")) | .[0]'
{
  "symbol": "XCH-USDT",
  "name": "XCH-USDT",
  "baseCurrency": "XCH",
  "quoteCurrency": "USDT",
  "feeCurrency": "USDT",
  "market": "USDS",
  "baseMinSize": "0.001",
  "quoteMinSize": "0.1",
  "baseMaxSize": "10000000000",
  "quoteMaxSize": "99999999",
  "baseIncrement": "0.0001",
  "quoteIncrement": "0.001",
  "priceIncrement": "0.001",
  "priceLimitRate": "0.1",
  "minFunds": "0.1",
  "isMarginEnabled": false,
  "enableTrading": true
}
*/
