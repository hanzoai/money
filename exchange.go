// Copyright (c) 2026, Hanzo AI, Inc. BSD-3-Clause.

package money

import (
	"strings"

	"github.com/hanzoai/decimal"
)

// Exchange is an in-memory table of conversion rates. A rate from → to means one unit of
// `from` buys `rate` units of `to`. Not safe for concurrent writes; build it once, then
// read.
type Exchange struct {
	rates map[string]decimal.Decimal
}

// NewExchange returns an empty rate table.
func NewExchange() *Exchange { return &Exchange{rates: map[string]decimal.Decimal{}} }

func rateKey(from, to Currency) string {
	return strings.ToUpper(from.Code) + ">" + strings.ToUpper(to.Code)
}

// AddRate registers (or replaces) the from→to rate from a decimal string ("0.85"). The
// inverse is NOT implied — add it explicitly if you need both directions.
func (e *Exchange) AddRate(from, to Currency, rate string) error {
	r, err := decimal.Parse(rate)
	if err != nil {
		return err
	}
	e.rates[rateKey(from, to)] = r
	return nil
}

// Rate returns the from→to rate (identity 1 when from == to).
func (e *Exchange) Rate(from, to Currency) (decimal.Decimal, error) {
	if strings.EqualFold(from.Code, to.Code) {
		return decimal.New(1, 0), nil
	}
	if r, ok := e.rates[rateKey(from, to)]; ok {
		return r, nil
	}
	return decimal.Decimal{}, ErrNoRate
}

// Convert converts amount into `to`, rounded to `to`'s decimals, using the registered rate.
func (e *Exchange) Convert(amount Amount, to Currency) (Amount, error) {
	r, err := e.Rate(amount.cur, to)
	if err != nil {
		return Amount{}, err
	}
	return ConvertWithRate(amount, to, r), nil
}

// ConvertWithRate converts amount into `to` at an explicit rate (rounded to `to`'s decimals).
func ConvertWithRate(amount Amount, to Currency, rate decimal.Decimal) Amount {
	return Amount{v: amount.v.Mul(rate).Round(to.Decimals), cur: to}
}
