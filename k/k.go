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

	apiService *kucoin.ApiService
	public     *kucoin.ApiService
)

func LoadClient(c *config.Config) {
	apiService = kucoin.NewApiService(
		kucoin.ApiKeyOption(c.KKey),
		kucoin.ApiKeyVersionOption(kucoin.ApiKeyVersionV2),
		kucoin.ApiPassPhraseOption(c.KPass),
		kucoin.ApiSecretOption(c.KSec),
	)

	public = kucoin.NewApiService()
}

func Book() ([]model.Order, []model.Order, error) {
	resp, err := public.AggregatedPartOrderBook("XCH-USDT", 100)
	if err != nil {
		return nil, nil, fmt.Errorf("order book: %w", err)
	}

	var o kucoin.PartOrderBookModel
	if err := resp.ReadData(&o); err != nil {
		return nil, nil, fmt.Errorf("read data: %w; resp: %+v", err, resp)
	}

	a := make([]model.Order, 0, len(o.Asks))
	for _, ask := range o.Asks {
		p, err := decimal.NewFromString(ask[0])
		if err != nil {
			return nil, nil, fmt.Errorf("tried to parse %v, got err: %w", ask[0], err)
		}
		amt, err := decimal.NewFromString(ask[1])
		if err != nil {
			return nil, nil, fmt.Errorf("tried to parse %v, got err: %w", ask[1], err)
		}
		o := model.Order{
			Ex:     model.ExchangeTypeKu,
			Price:  p,
			Amount: amt,
		}
		o.EffectivePrice = o.Price.Mul(AskAddition)
		a = append(a, o)
	}
	b := make([]model.Order, 0, len(o.Bids))
	for _, bid := range o.Bids {
		p, err := decimal.NewFromString(bid[0])
		if err != nil {
			return nil, nil, fmt.Errorf("tried to parse %v, got err: %w", bid[0], err)
		}
		amt, err := decimal.NewFromString(bid[1])
		if err != nil {
			return nil, nil, fmt.Errorf("tried to parse %v, got err: %w", bid[1], err)
		}
		o := model.Order{
			Ex:     model.ExchangeTypeKu,
			Price:  p,
			Amount: amt,
		}
		o.EffectivePrice = o.Price.Mul(BidReduction)
		b = append(b, o)
	}

	return a, b, nil
}

func Balances() (b model.Balances, err error) {
	resp, err := apiService.Accounts("", "")
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
				usdt, err := decimal.NewFromString(aa.Available)
				if err != nil {
					return b, fmt.Errorf("failed to parse %v into decimal: %w", aa.Available, err)
				}
				b.USDT = usdt
			case "XCH":
				xch, err := decimal.NewFromString(aa.Available)
				if err != nil {
					return b, fmt.Errorf("failed to parse %v into decimal: %w", aa.Available, err)
				}
				b.XCH = xch
			}
		}
	}

	return b, nil
}

func Buy(price, size decimal.Decimal) (string, error) {
	resp, err := apiService.CreateOrder(&kucoin.CreateOrderModel{
		// BASE PARAMETERS
		ClientOid: uuid.New().String(),
		Side:      "buy",
		Symbol:    "XCH-USDT",
		Type:      "limit",
		STP:       "DC",

		// LIMIT ORDER PARAMETERS
		Price:       price.String(),
		Size:        size.RoundDown(4).String(),
		TimeInForce: "GTC",
	})
	if err != nil {
		return "", err
	}

	var o kucoin.CreateOrderResultModel
	if err := resp.ReadData(&o); err != nil {
		return "", err
	}

	return o.OrderId, nil
}

func Sell(price, size decimal.Decimal) (string, error) {
	resp, err := apiService.CreateOrder(&kucoin.CreateOrderModel{
		// BASE PARAMETERS
		ClientOid: uuid.New().String(),
		Side:      "sell",
		Symbol:    "XCH-USDT",
		Type:      "limit",
		STP:       "DC",

		// LIMIT ORDER PARAMETERS
		Price:       price.String(),
		Size:        size.RoundDown(4).String(),
		TimeInForce: "GTC",
	})
	if err != nil {
		return "", err
	}

	var o kucoin.CreateOrderResultModel
	if err := resp.ReadData(&o); err != nil {
		return "", err
	}

	return o.OrderId, nil
}

func GetOrder(id string) (*kucoin.OrderModel, error) {
	resp, err := apiService.Order(id)
	if err != nil {
		return nil, err
	}

	var o kucoin.OrderModel
	if err := resp.ReadData(&o); err != nil {
		return nil, err
	}

	return &o, nil
}

// {65a8928fcf1c7f00074b0ea7}
// {Id:65a8928fcf1c7f00074b0ea7 Symbol:XCH-USDT OpType:DEAL Type:limit Side:buy Price:20 Size:0.1 Funds:0 DealFunds:0 DealSize:0 Fee:0 FeeCurrency:USDT Stp: Stop: StopTriggered:false StopPrice:0 TimeInForce:IOC PostOnly:false Hidden:false IceBerg:false VisibleSize:0 CancelAfter:0 Channel:API ClientOid:d6c51d3e-2f72-4e38-a9a6-2afc14041e2e Remark: Tags: IsActive:false CancelExist:true CreatedAt:1705546383349 TradeType:TRADE}
func OrderTest() {
	oid, err := Buy(decimal.NewFromInt(20), decimal.NewFromInt(1).Div(decimal.NewFromInt(10)))
	fmt.Println(oid, err)
	if err != nil {
		return
	}

	resp, err := apiService.Order(oid)
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
func QueryFee() {
	resp, err := apiService.ActualFee("XCH-USDT")
	if err != nil {
		fmt.Println(err)
		return
	}

	var f kucoin.TradeFeesResultModel
	if err := resp.ReadData(&f); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%+v\n", f)
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
