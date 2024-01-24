package arb

import (
	"testing"

	"github.com/L3Sota/arbo/arb/model"
	"github.com/L3Sota/arbo/k"
	"github.com/L3Sota/arbo/m"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"
)

func TestArbo(t *testing.T) {
	type arboOut struct {
		As            side
		Bs            side
		TotalTradeXCH decimal.Decimal
		Gain          decimal.Decimal
		WithdrawUSDT  decimal.Decimal
		WithdrawXCH   decimal.Decimal
		Profit        decimal.Decimal
		TotalBuyUSDT  map[model.ExchangeType]decimal.Decimal
		TotalSellUSDT map[model.ExchangeType]decimal.Decimal
	}

	empty := map[model.ExchangeType]decimal.Decimal{
		model.ME: decimal.Zero,
		model.Ku: decimal.Zero,
		model.Hu: decimal.Zero,
		model.Co: decimal.Zero,
		model.Ga: decimal.Zero,
	}
	defaultOut := arboOut{
		As: side{
			I:          0,
			HeadAmount: decimal.Zero,
			LastPrice:  decimal.Zero,
			Move:       false,
		},
		Bs: side{
			I:          0,
			HeadAmount: decimal.Zero,
			LastPrice:  decimal.Zero,
			Move:       false,
		},
		TotalTradeXCH: decimal.Zero,
		Gain:          decimal.Zero,
		WithdrawUSDT:  decimal.Zero,
		WithdrawXCH:   decimal.Zero,
		Profit:        decimal.Zero,
		TotalBuyUSDT:  empty,
		TotalSellUSDT: empty,
	}

	for name, tc := range map[string]struct {
		a      []model.Order
		b      []model.Order
		result arboOut
	}{
		"empty": {
			a:      nil,
			b:      nil,
			result: defaultOut,
		},
		"no match (effective price)": {
			a: []model.Order{
				{
					Ex:             model.ME,
					Price:          decimal.NewFromInt(28),
					EffectivePrice: decimal.NewFromInt(30),
					Amount:         decimal.NewFromInt(1),
				},
			},
			b: []model.Order{
				{
					Ex:             model.Ku,
					Price:          decimal.NewFromInt(31),
					EffectivePrice: decimal.NewFromInt(29),
					Amount:         decimal.NewFromInt(1),
				},
			},
			result: func() arboOut {
				out := defaultOut
				out.As.HeadAmount = decimal.NewFromInt(1)
				out.As.LastPrice = decimal.NewFromInt(28)
				out.Bs.HeadAmount = decimal.NewFromInt(1)
				out.Bs.LastPrice = decimal.NewFromInt(31)
				return out
			}(),
		},
		"match a < b": {
			a: []model.Order{
				{
					Ex:             model.ME,
					Price:          decimal.NewFromInt(20),
					EffectivePrice: decimal.NewFromInt(25),
					Amount:         decimal.NewFromInt(1),
				},
			},
			b: []model.Order{
				{
					Ex:             model.Ku,
					Price:          decimal.NewFromInt(28),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(3),
				},
			},
			result: arboOut{
				As: side{
					I:          1,
					HeadAmount: decimal.NewFromInt(1),
					LastPrice:  decimal.NewFromInt(20),
					Move:       true,
				},
				Bs: side{
					I:          0,
					HeadAmount: decimal.NewFromInt(2), // 3 - 1
					LastPrice:  decimal.NewFromInt(28),
					Move:       false,
				},
				TotalTradeXCH: decimal.NewFromInt(1),
				Gain:          decimal.NewFromInt(2),
				WithdrawUSDT:  k.Fees.WithdrawalFlatUSDT,
				WithdrawXCH:   m.Fees.WithdrawalFlatXCH,
				Profit:        decimal.NewFromInt(2).Sub(k.Fees.WithdrawalFlatUSDT).Sub(m.Fees.WithdrawalFlatXCH.Mul(decimal.NewFromInt(28))),
				TotalBuyUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ME: decimal.NewFromInt(20),
					model.Ku: decimal.Zero,
					model.Hu: decimal.Zero,
					model.Co: decimal.Zero,
					model.Ga: decimal.Zero,
				},
				TotalSellUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ME: decimal.Zero,
					model.Ku: decimal.NewFromInt(28),
					model.Hu: decimal.Zero,
					model.Co: decimal.Zero,
					model.Ga: decimal.Zero,
				},
			},
		},
		"match a < b, a > b": {
			a: []model.Order{
				{
					Ex:             model.ME,
					Price:          decimal.NewFromInt(20),
					EffectivePrice: decimal.NewFromInt(25),
					Amount:         decimal.NewFromInt(1),
				},
				{
					Ex:             model.ME,
					Price:          decimal.NewFromInt(21),
					EffectivePrice: decimal.NewFromInt(26),
					Amount:         decimal.NewFromInt(3),
				},
			},
			b: []model.Order{
				{
					Ex:             model.Ku,
					Price:          decimal.NewFromInt(28),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(3),
				},
			},
			result: arboOut{
				As: side{
					I:          1,
					HeadAmount: decimal.NewFromInt(1),
					LastPrice:  decimal.NewFromInt(21),
					Move:       false,
				},
				Bs: side{
					I:          1,
					HeadAmount: decimal.NewFromInt(2),
					LastPrice:  decimal.NewFromInt(28),
					Move:       true,
				},
				TotalTradeXCH: decimal.NewFromInt(3),
				Gain:          decimal.NewFromInt(4), // 2 + 2
				WithdrawUSDT:  k.Fees.WithdrawalFlatUSDT,
				WithdrawXCH:   m.Fees.WithdrawalFlatXCH,
				Profit:        decimal.NewFromInt(4).Sub(k.Fees.WithdrawalFlatUSDT).Sub(m.Fees.WithdrawalFlatXCH.Mul(decimal.NewFromInt(28))),
				TotalBuyUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ME: decimal.NewFromInt(62), // 20 + 2*21
					model.Ku: decimal.Zero,
					model.Hu: decimal.Zero,
					model.Co: decimal.Zero,
					model.Ga: decimal.Zero,
				},
				TotalSellUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ME: decimal.Zero,
					model.Ku: decimal.NewFromInt(84), // 3*28
					model.Hu: decimal.Zero,
					model.Co: decimal.Zero,
					model.Ga: decimal.Zero,
				},
			},
		},
		"match a < b, a > b, a = b": {
			a: []model.Order{
				{
					Ex:             model.ME,
					Price:          decimal.NewFromInt(20),
					EffectivePrice: decimal.NewFromInt(25),
					Amount:         decimal.NewFromInt(1),
				},
				{
					Ex:             model.ME,
					Price:          decimal.NewFromInt(21),
					EffectivePrice: decimal.NewFromInt(26),
					Amount:         decimal.NewFromInt(3),
				},
			},
			b: []model.Order{
				{
					Ex:             model.Ku,
					Price:          decimal.NewFromInt(28),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(3),
				},
				{
					Ex:             model.Hu,
					Price:          decimal.NewFromInt(29),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(1),
				},
			},
			result: arboOut{
				As: side{
					I:          2,
					HeadAmount: decimal.NewFromInt(1),
					LastPrice:  decimal.NewFromInt(21),
					Move:       true,
				},
				Bs: side{
					I:          2,
					HeadAmount: decimal.NewFromInt(1),
					LastPrice:  decimal.NewFromInt(29),
					Move:       true,
				},
				TotalTradeXCH: decimal.NewFromInt(4),
				Gain:          decimal.NewFromInt(5), // 2 + 2 + 1
				WithdrawUSDT:  k.Fees.WithdrawalFlatUSDT,
				WithdrawXCH:   m.Fees.WithdrawalFlatXCH,
				Profit:        decimal.NewFromInt(5).Sub(k.Fees.WithdrawalFlatUSDT).Sub(m.Fees.WithdrawalFlatXCH.Mul(decimal.NewFromInt(29))),
				TotalBuyUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ME: decimal.NewFromInt(83), // 20 + 3*21
					model.Ku: decimal.Zero,
					model.Hu: decimal.Zero,
					model.Co: decimal.Zero,
					model.Ga: decimal.Zero,
				},
				TotalSellUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ME: decimal.Zero,
					model.Ku: decimal.NewFromInt(84), // 3*28
					model.Hu: decimal.NewFromInt(29), // 29
					model.Co: decimal.Zero,
					model.Ga: decimal.Zero,
				},
			},
		},
		"match a < b, a > b, a = b, no further match": {
			a: []model.Order{
				{
					Ex:             model.ME,
					Price:          decimal.NewFromInt(20),
					EffectivePrice: decimal.NewFromInt(25),
					Amount:         decimal.NewFromInt(1),
				},
				{
					Ex:             model.ME,
					Price:          decimal.NewFromInt(21),
					EffectivePrice: decimal.NewFromInt(26),
					Amount:         decimal.NewFromInt(3),
				},
				{
					Ex:             model.Co,
					Price:          decimal.NewFromInt(22),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(100),
				},
			},
			b: []model.Order{
				{
					Ex:             model.Ku,
					Price:          decimal.NewFromInt(28),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(3),
				},
				{
					Ex:             model.Hu,
					Price:          decimal.NewFromInt(29),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(1),
				},
				{
					Ex:             model.Ga,
					Price:          decimal.NewFromInt(30),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(10),
				},
			},
			result: arboOut{
				As: side{
					I:          2,
					HeadAmount: decimal.NewFromInt(100),
					LastPrice:  decimal.NewFromInt(21),
					Move:       false,
				},
				Bs: side{
					I:          2,
					HeadAmount: decimal.NewFromInt(10),
					LastPrice:  decimal.NewFromInt(29),
					Move:       false,
				},
				TotalTradeXCH: decimal.NewFromInt(4),
				Gain:          decimal.NewFromInt(5), // 2 + 2 + 1
				WithdrawUSDT:  k.Fees.WithdrawalFlatUSDT,
				WithdrawXCH:   m.Fees.WithdrawalFlatXCH,
				Profit:        decimal.NewFromInt(5).Sub(k.Fees.WithdrawalFlatUSDT).Sub(m.Fees.WithdrawalFlatXCH.Mul(decimal.NewFromInt(29))),
				TotalBuyUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ME: decimal.NewFromInt(83), // 20 + 3*21
					model.Ku: decimal.Zero,
					model.Hu: decimal.Zero,
					model.Co: decimal.Zero,
					model.Ga: decimal.Zero,
				},
				TotalSellUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ME: decimal.Zero,
					model.Ku: decimal.NewFromInt(84), // 3*28
					model.Hu: decimal.NewFromInt(29), // 29
					model.Co: decimal.Zero,
					model.Ga: decimal.Zero,
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			tc.result.As.Book = tc.a
			tc.result.Bs.Book = tc.b
			as, bs, totalTradeQuote, gain, withdrawUSDT, withdrawXCH, profit, totalBuyBase, totalSellBase := arbo(tc.a, tc.b)
			if diff := cmp.Diff(tc.result, arboOut{
				as, bs, totalTradeQuote, gain, withdrawUSDT, withdrawXCH, profit, totalBuyBase, totalSellBase,
			}, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("-want/+got: %v", diff)
			}
		})
	}
}

func BenchmarkGatherBooksP(b *testing.B) {
	GatherBooksP()
}

func BenchmarkGatherBooks(b *testing.B) {
	GatherBooks()
}
