package model

import (
	"github.com/shopspring/decimal"
)

type ExchangeType uint8

const (
	ME ExchangeType = iota
	Ku
	Hu
	Co
	Ga
)

var ExchangeTypes = [5]ExchangeType{ME, Ku, Hu, Co, Ga}

func (e ExchangeType) String() string {
	switch e {
	case ME:
		return "ME"
	case Ku:
		return "Ku"
	case Hu:
		return "Hu"
	case Co:
		return "Co"
	case Ga:
		return "Ga"
	default:
		return "??"
	}
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
