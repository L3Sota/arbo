package model

import (
	"github.com/shopspring/decimal"
)

type ExchangeType uint8

const (
	ME = iota
	Ku
	Hu
	Co
	Ga
)

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
	Ex     ExchangeType
	Price  decimal.Decimal
	Amount decimal.Decimal
}
