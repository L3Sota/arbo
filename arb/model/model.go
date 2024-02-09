package model

import (
	"github.com/shopspring/decimal"
)

type ExchangeType uint8

const (
	ExchangeTypeMe ExchangeType = iota
	ExchangeTypeKu
	ExchangeTypeHu
	ExchangeTypeCo
	ExchangeTypeGa
	ExchangeTypeMax
)

var ExchangeTypes = [ExchangeTypeMax]ExchangeType{ExchangeTypeMe, ExchangeTypeKu, ExchangeTypeHu, ExchangeTypeCo, ExchangeTypeGa}

func (e ExchangeType) String() string {
	switch e {
	case ExchangeTypeMe:
		return "Me"
	case ExchangeTypeKu:
		return "Ku"
	case ExchangeTypeHu:
		return "Hu"
	case ExchangeTypeCo:
		return "Co"
	case ExchangeTypeGa:
		return "Ga"
	default:
		return "??"
	}
}

type Balances struct {
	XCH  decimal.Decimal
	USDT decimal.Decimal
}

type Order struct {
	Ex             ExchangeType
	Price          decimal.Decimal
	EffectivePrice decimal.Decimal
	Amount         decimal.Decimal
}

type Fees struct {
	MakerTakerRatio    decimal.Decimal
	WithdrawalFlatXCH  decimal.Decimal
	WithdrawalFlatUSDT decimal.Decimal
}
