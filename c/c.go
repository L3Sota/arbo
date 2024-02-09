package c

import (
	"encoding/json"
	"fmt"

	"github.com/L3Sota/arbo/arb/model"
	"github.com/shopspring/decimal"
	"gopkg.in/resty.v1"
)

var (
	Fees = model.Fees{
		MakerTakerRatio:    decimal.RequireFromString("0.002"), // 0.2%
		WithdrawalFlatXCH:  decimal.RequireFromString("0.001"),
		WithdrawalFlatUSDT: decimal.RequireFromString("1.4"), // TRC20
	}

	AskAddition  = decimal.NewFromInt(1).Add(Fees.MakerTakerRatio)
	BidReduction = decimal.NewFromInt(1).Sub(Fees.MakerTakerRatio)
)

type book struct {
	Last string
	Time int64
	Asks [][]string
	Bids [][]string
}

func Book() ([]model.Order, []model.Order, error) {
	client := resty.New()

	resp, err := client.R().Get("https://api.coinex.com/v1/market/depth?market=XCHUSDT&merge=0.01&limit=50")
	if err != nil {
		return nil, nil, err
	}

	raw := &struct {
		Data book
	}{}

	if err := json.Unmarshal(resp.Body(), raw); err != nil {
		return nil, nil, err
	}

	o := raw.Data

	a := make([]model.Order, 0, len(o.Asks))
	for _, ask := range o.Asks {
		o := model.Order{
			Ex:     model.ExchangeTypeCo,
			Price:  decimal.RequireFromString(ask[0]),
			Amount: decimal.RequireFromString(ask[1]),
		}
		o.EffectivePrice = o.Price.Mul(AskAddition)
		a = append(a, o)
	}
	b := make([]model.Order, 0, len(o.Bids))
	for _, bid := range o.Bids {
		o := model.Order{
			Ex:     model.ExchangeTypeCo,
			Price:  decimal.RequireFromString(bid[0]),
			Amount: decimal.RequireFromString(bid[1]),
		}
		o.EffectivePrice = o.Price.Mul(BidReduction)
		b = append(b, o)
	}

	return a, b, nil
}

func Balances() (b model.Balances, err error) {
	//Inquire account asset constructure
	accooutRespBody, err := GetAccount()
	if err != nil {
		return b, err
	}
	var a BalanceResp
	json.Unmarshal(accooutRespBody, &a)

	if a.Code != 0 {
		return b, fmt.Errorf("[Error %d] %v", a.Code, a.Message)
	}

	for currency, aa := range a.AssetBalance {
		switch currency {
		case "USDT":
			b.USDT = decimal.RequireFromString(aa.Available)
		case "XCH":
			b.XCH = decimal.RequireFromString(aa.Available)
		}
	}

	return b, nil
}

func Buy(price, size decimal.Decimal) (*OrderResp, error) {
	//put limit order
	limitOrderRespBody, err := PutLimitOrder(
		size.String(),
		price.String(),
		"buy",
		"XCHUSDT")
	if err != nil {
		return nil, err
	}
	var putLimitOrderResp OrderResp
	if err := json.Unmarshal(limitOrderRespBody, &putLimitOrderResp); err != nil {
		return nil, err
	}
	return &putLimitOrderResp, nil
}

func Sell(price, size decimal.Decimal) (*OrderResp, error) {
	//put limit order
	limitOrderRespBody, err := PutLimitOrder(
		size.String(),
		price.String(),
		"sell",
		"XCHUSDT")
	if err != nil {
		return nil, err
	}
	var putLimitOrderResp OrderResp
	if err := json.Unmarshal(limitOrderRespBody, &putLimitOrderResp); err != nil {
		return nil, err
	}
	return &putLimitOrderResp, nil
}

func OrderTest() {
	putLimitOrderResp, err := Buy(decimal.NewFromInt(20),
		decimal.NewFromInt(1).Div(decimal.NewFromInt(10)))
	if err != nil {
		fmt.Printf("PutLimitOrder Error: %v\n", err)
		return
	}
	fmt.Printf("PutLimitOrder: %v\n", putLimitOrderResp)
}
