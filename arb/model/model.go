package model

import (
	"github.com/shopspring/decimal"
)

type ExchangeType uint8

const (
	MEX = iota
	Ku
	Hu
	Co
)

type Order struct {
	Price  decimal.Decimal
	Amount decimal.Decimal
}
