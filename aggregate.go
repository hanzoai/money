// Copyright (c) 2026, Hanzo AI, Inc. MIT OR Apache-2.0.

package money

import "github.com/hanzoai/decimal"

// Sum adds all amounts, which must share a currency (a zero-value operand is agnostic).
// The empty sum is a zero-value Amount.
func Sum(amounts ...Amount) (Amount, error) {
	acc := Amount{}
	for _, a := range amounts {
		var err error
		if acc, err = acc.Add(a); err != nil {
			return Amount{}, err
		}
	}
	return acc, nil
}

// Min returns the smallest amount (same currency). Errors on an empty list or a currency
// mismatch.
func Min(amounts ...Amount) (Amount, error) {
	if len(amounts) == 0 {
		return Amount{}, ErrNonPositive
	}
	m := amounts[0]
	for _, a := range amounts[1:] {
		if _, err := m.resolveCur(a); err != nil {
			return Amount{}, err
		}
		if a.Cmp(m) < 0 {
			m = a
		}
	}
	return m, nil
}

// Max returns the largest amount (same currency).
func Max(amounts ...Amount) (Amount, error) {
	if len(amounts) == 0 {
		return Amount{}, ErrNonPositive
	}
	m := amounts[0]
	for _, a := range amounts[1:] {
		if _, err := m.resolveCur(a); err != nil {
			return Amount{}, err
		}
		if a.Cmp(m) > 0 {
			m = a
		}
	}
	return m, nil
}

// Avg returns the mean, rounded to the currency's decimals (half-away-from-zero). Errors
// on an empty list or a currency mismatch.
func Avg(amounts ...Amount) (Amount, error) {
	if len(amounts) == 0 {
		return Amount{}, ErrNonPositive
	}
	s, err := Sum(amounts...)
	if err != nil {
		return Amount{}, err
	}
	cur := s.cur
	if cur.Code == "" {
		cur = amounts[0].cur
	}
	mean := s.v.Quo(decimal.New(int64(len(amounts)), 0), cur.Decimals)
	return Amount{v: mean, cur: cur}, nil
}
