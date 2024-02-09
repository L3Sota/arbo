package arb

import (
	"testing"

	"github.com/L3Sota/arbo/arb/config"
	"github.com/L3Sota/arbo/arb/model"
	"github.com/L3Sota/arbo/g"
	"github.com/L3Sota/arbo/k"
	"github.com/L3Sota/arbo/m"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"
)

func TestArbo(t *testing.T) {
	t.Parallel()

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
		TotalBuyXCH   map[model.ExchangeType]decimal.Decimal
		TotalSellXCH  map[model.ExchangeType]decimal.Decimal
	}

	empty := map[model.ExchangeType]decimal.Decimal{
		model.ExchangeTypeMe: decimal.Zero,
		model.ExchangeTypeKu: decimal.Zero,
		model.ExchangeTypeHu: decimal.Zero,
		model.ExchangeTypeCo: decimal.Zero,
		model.ExchangeTypeGa: decimal.Zero,
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
		TotalBuyXCH:   empty,
		TotalSellXCH:  empty,
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
					Ex:             model.ExchangeTypeMe,
					Price:          decimal.NewFromInt(28),
					EffectivePrice: decimal.NewFromInt(30),
					Amount:         decimal.NewFromInt(1),
				},
			},
			b: []model.Order{
				{
					Ex:             model.ExchangeTypeKu,
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
					Ex:             model.ExchangeTypeMe,
					Price:          decimal.NewFromInt(20),
					EffectivePrice: decimal.NewFromInt(25),
					Amount:         decimal.NewFromInt(1),
				},
			},
			b: []model.Order{
				{
					Ex:             model.ExchangeTypeKu,
					Price:          decimal.NewFromInt(28),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(3),
				},
			},
			result: arboOut{
				As: side{
					I:          1,
					HeadAmount: decimal.Zero,
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
					model.ExchangeTypeMe: decimal.NewFromInt(25),
					model.ExchangeTypeKu: decimal.Zero,
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
				TotalSellUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.Zero,
					model.ExchangeTypeKu: decimal.NewFromInt(27),
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
				TotalBuyXCH: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.NewFromInt(1),
					model.ExchangeTypeKu: decimal.Zero,
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
				TotalSellXCH: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.Zero,
					model.ExchangeTypeKu: decimal.NewFromInt(1),
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
			},
		},
		"match a < b, a > b": {
			a: []model.Order{
				{
					Ex:             model.ExchangeTypeMe,
					Price:          decimal.NewFromInt(20),
					EffectivePrice: decimal.NewFromInt(25),
					Amount:         decimal.NewFromInt(1),
				},
				{
					Ex:             model.ExchangeTypeMe,
					Price:          decimal.NewFromInt(21),
					EffectivePrice: decimal.NewFromInt(26),
					Amount:         decimal.NewFromInt(3),
				},
			},
			b: []model.Order{
				{
					Ex:             model.ExchangeTypeKu,
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
					HeadAmount: decimal.Zero,
					LastPrice:  decimal.NewFromInt(28),
					Move:       true,
				},
				TotalTradeXCH: decimal.NewFromInt(3),
				Gain:          decimal.NewFromInt(4), // 2 + 2
				WithdrawUSDT:  k.Fees.WithdrawalFlatUSDT,
				WithdrawXCH:   m.Fees.WithdrawalFlatXCH,
				Profit:        decimal.NewFromInt(4).Sub(k.Fees.WithdrawalFlatUSDT).Sub(m.Fees.WithdrawalFlatXCH.Mul(decimal.NewFromInt(28))),
				TotalBuyUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.NewFromInt(77), // 25 + 2*26
					model.ExchangeTypeKu: decimal.Zero,
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
				TotalSellUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.Zero,
					model.ExchangeTypeKu: decimal.NewFromInt(81), // 3*27
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
				TotalBuyXCH: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.NewFromInt(3),
					model.ExchangeTypeKu: decimal.Zero,
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
				TotalSellXCH: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.Zero,
					model.ExchangeTypeKu: decimal.NewFromInt(3),
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
			},
		},
		"match a < b, a > b, a = b": {
			a: []model.Order{
				{
					Ex:             model.ExchangeTypeMe,
					Price:          decimal.NewFromInt(20),
					EffectivePrice: decimal.NewFromInt(25),
					Amount:         decimal.NewFromInt(1),
				},
				{
					Ex:             model.ExchangeTypeMe,
					Price:          decimal.NewFromInt(21),
					EffectivePrice: decimal.NewFromInt(26),
					Amount:         decimal.NewFromInt(3),
				},
			},
			b: []model.Order{
				{
					Ex:             model.ExchangeTypeKu,
					Price:          decimal.NewFromInt(28),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(3),
				},
				{
					Ex:             model.ExchangeTypeHu,
					Price:          decimal.NewFromInt(29),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(1),
				},
			},
			result: arboOut{
				As: side{
					I:          2,
					HeadAmount: decimal.Zero,
					LastPrice:  decimal.NewFromInt(21),
					Move:       true,
				},
				Bs: side{
					I:          2,
					HeadAmount: decimal.Zero,
					LastPrice:  decimal.NewFromInt(29),
					Move:       true,
				},
				TotalTradeXCH: decimal.NewFromInt(4),
				Gain:          decimal.NewFromInt(5), // 2 + 2 + 1
				WithdrawUSDT:  k.Fees.WithdrawalFlatUSDT,
				WithdrawXCH:   m.Fees.WithdrawalFlatXCH,
				Profit:        decimal.NewFromInt(5).Sub(k.Fees.WithdrawalFlatUSDT).Sub(m.Fees.WithdrawalFlatXCH.Mul(decimal.NewFromInt(29))),
				TotalBuyUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.NewFromInt(103), // 25 + 3*26
					model.ExchangeTypeKu: decimal.Zero,
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
				TotalSellUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.Zero,
					model.ExchangeTypeKu: decimal.NewFromInt(81), // 3*27
					model.ExchangeTypeHu: decimal.NewFromInt(27), // 27
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
				TotalBuyXCH: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.NewFromInt(4),
					model.ExchangeTypeKu: decimal.Zero,
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
				TotalSellXCH: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.Zero,
					model.ExchangeTypeKu: decimal.NewFromInt(3),
					model.ExchangeTypeHu: decimal.NewFromInt(1),
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
			},
		},
		"match a < b, a > b, a = b, no further match": {
			a: []model.Order{
				{
					Ex:             model.ExchangeTypeMe,
					Price:          decimal.NewFromInt(20),
					EffectivePrice: decimal.NewFromInt(25),
					Amount:         decimal.NewFromInt(1),
				},
				{
					Ex:             model.ExchangeTypeMe,
					Price:          decimal.NewFromInt(21),
					EffectivePrice: decimal.NewFromInt(26),
					Amount:         decimal.NewFromInt(3),
				},
				{
					Ex:             model.ExchangeTypeCo,
					Price:          decimal.NewFromInt(22),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(100),
				},
			},
			b: []model.Order{
				{
					Ex:             model.ExchangeTypeKu,
					Price:          decimal.NewFromInt(28),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(3),
				},
				{
					Ex:             model.ExchangeTypeHu,
					Price:          decimal.NewFromInt(29),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(1),
				},
				{
					Ex:             model.ExchangeTypeGa,
					Price:          decimal.NewFromInt(30),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(10),
				},
			},
			result: arboOut{
				As: side{
					I:          2,
					HeadAmount: decimal.Zero,
					LastPrice:  decimal.NewFromInt(21),
					Move:       true,
				},
				Bs: side{
					I:          2,
					HeadAmount: decimal.Zero,
					LastPrice:  decimal.NewFromInt(29),
					Move:       true,
				},
				TotalTradeXCH: decimal.NewFromInt(4),
				Gain:          decimal.NewFromInt(5), // 2 + 2 + 1
				WithdrawUSDT:  k.Fees.WithdrawalFlatUSDT,
				WithdrawXCH:   m.Fees.WithdrawalFlatXCH,
				Profit:        decimal.NewFromInt(5).Sub(k.Fees.WithdrawalFlatUSDT).Sub(m.Fees.WithdrawalFlatXCH.Mul(decimal.NewFromInt(29))),
				TotalBuyUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.NewFromInt(103), // 25 + 3*26
					model.ExchangeTypeKu: decimal.Zero,
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
				TotalSellUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.Zero,
					model.ExchangeTypeKu: decimal.NewFromInt(81), // 3*27
					model.ExchangeTypeHu: decimal.NewFromInt(27), // 27
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
				TotalBuyXCH: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.NewFromInt(4),
					model.ExchangeTypeKu: decimal.Zero,
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
				TotalSellXCH: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.Zero,
					model.ExchangeTypeKu: decimal.NewFromInt(3),
					model.ExchangeTypeHu: decimal.NewFromInt(1),
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
			},
		},
	} {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.result.As.Book = tc.a
			tc.result.Bs.Book = tc.b
			as, bs, totalTradeQuote, gain, withdrawUSDT, withdrawXCH, profit, totalBuyBase, totalSellBase, totalBuyXCH, totalSellXCH := arbo(tc.a, tc.b, ignoreBalances, &config.Config{})
			if diff := cmp.Diff(tc.result, arboOut{
				as, bs, totalTradeQuote, gain, withdrawUSDT, withdrawXCH, profit, totalBuyBase, totalSellBase, totalBuyXCH, totalSellXCH,
			}, cmpopts.EquateEmpty(), cmpopts.IgnoreFields(side{}, "HeadAllowance")); diff != "" {
				t.Errorf("-want/+got: %v", diff)
			}
		})
	}

	for name, tc := range map[string]struct {
		a        []model.Order
		b        []model.Order
		balances map[model.ExchangeType]model.Balances
		result   arboOut
	}{
		"match a < b, a > b, balance constrained on a, no further match": {
			a: []model.Order{
				{
					Ex:             model.ExchangeTypeMe,
					Price:          decimal.NewFromInt(20),
					EffectivePrice: decimal.NewFromInt(25),
					Amount:         decimal.NewFromInt(1),
				},
				{
					Ex:             model.ExchangeTypeMe,
					Price:          decimal.NewFromInt(21),
					EffectivePrice: decimal.NewFromInt(26),
					Amount:         decimal.NewFromInt(3),
				},
				{
					Ex:             model.ExchangeTypeCo,
					Price:          decimal.NewFromInt(22),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(100),
				},
			},
			b: []model.Order{
				{
					Ex:             model.ExchangeTypeKu,
					Price:          decimal.NewFromInt(28),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(3),
				},
				{
					Ex:             model.ExchangeTypeGa,
					Price:          decimal.NewFromInt(29),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(1),
				},
				{
					Ex:             model.ExchangeTypeGa,
					Price:          decimal.NewFromInt(30),
					EffectivePrice: decimal.NewFromInt(27),
					Amount:         decimal.NewFromInt(10),
				},
			},
			balances: map[model.ExchangeType]model.Balances{
				model.ExchangeTypeMe: {
					XCH:  decimal.Zero,
					USDT: decimal.NewFromInt(90),
				},
				model.ExchangeTypeKu: bigBalance,
				model.ExchangeTypeHu: bigBalance,
				model.ExchangeTypeCo: bigBalance,
				model.ExchangeTypeGa: bigBalance,
			},
			result: arboOut{
				As: side{
					I:             2,
					HeadAmount:    decimal.NewFromFloat(0.5),
					HeadAllowance: decimal.NewFromFloat(0.5),
					LastPrice:     decimal.NewFromInt(21),
					Move:          true,
				},
				Bs: side{
					I:             1,
					HeadAmount:    decimal.NewFromFloat(0.5),
					HeadAllowance: big.Mul(g.BidReduction),
					LastPrice:     decimal.NewFromInt(29),
					Move:          false,
				},
				TotalTradeXCH: decimal.NewFromFloat(3.5),
				Gain:          decimal.NewFromFloat(4.5), // 2 + 2 + 0.5
				WithdrawUSDT:  k.Fees.WithdrawalFlatUSDT.Add(g.Fees.WithdrawalFlatUSDT),
				WithdrawXCH:   m.Fees.WithdrawalFlatXCH,
				Profit:        decimal.NewFromFloat(4.5).Sub(k.Fees.WithdrawalFlatUSDT).Sub(g.Fees.WithdrawalFlatUSDT).Sub(m.Fees.WithdrawalFlatXCH.Mul(decimal.NewFromInt(29))),
				TotalBuyUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.NewFromInt(90), // balance
					model.ExchangeTypeKu: decimal.Zero,
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
				TotalSellUSDT: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.Zero,
					model.ExchangeTypeKu: decimal.NewFromInt(81), // 3*27
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.NewFromFloat(13.5), // 0.5*27
				},
				TotalBuyXCH: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.NewFromFloat(3.5),
					model.ExchangeTypeKu: decimal.Zero,
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.Zero,
				},
				TotalSellXCH: map[model.ExchangeType]decimal.Decimal{
					model.ExchangeTypeMe: decimal.Zero,
					model.ExchangeTypeKu: decimal.NewFromInt(3),
					model.ExchangeTypeHu: decimal.Zero,
					model.ExchangeTypeCo: decimal.Zero,
					model.ExchangeTypeGa: decimal.NewFromFloat(0.5),
				},
			},
		},
	} {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.result.As.Book = tc.a
			tc.result.Bs.Book = tc.b
			as, bs, totalTradeQuote, gain, withdrawUSDT, withdrawXCH, profit, totalBuyBase, totalSellBase, totalBuyXCH, totalSellXCH := arbo(tc.a, tc.b, tc.balances, &config.Config{})
			if diff := cmp.Diff(tc.result, arboOut{
				as, bs, totalTradeQuote, gain, withdrawUSDT, withdrawXCH, profit, totalBuyBase, totalSellBase, totalBuyXCH, totalSellXCH,
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
