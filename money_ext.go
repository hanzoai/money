// Copyright (c) 2026, Hanzo AI, Inc. MIT OR Apache-2.0.

package money

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/hanzoai/decimal"
)

// Allocate distributes the amount across the given integer ratios, summing EXACTLY to the
// original at the currency's minor-unit granularity — each part floors, then the leftover
// minor units go one-at-a-time to the leading parts. Ratios must be non-negative with a
// positive total.
func (a Amount) Allocate(ratios ...int) ([]Amount, error) {
	if len(ratios) == 0 {
		return nil, ErrNonPositive
	}
	total := 0
	for _, r := range ratios {
		if r < 0 {
			return nil, ErrNonPositive
		}
		total += r
	}
	if total == 0 {
		return nil, ErrNonPositive
	}
	minor := a.Minor()
	tb := big.NewInt(int64(total))
	remainder := new(big.Int).Set(minor)
	out := make([]Amount, len(ratios))
	for i, r := range ratios {
		share := new(big.Int).Mul(minor, big.NewInt(int64(r)))
		share.Quo(share, tb) // floor
		out[i] = FromMinorBig(share, a.cur)
		remainder.Sub(remainder, share)
	}
	one := big.NewInt(1)
	for i := 0; remainder.Sign() > 0 && i < len(out); i++ {
		out[i] = FromMinorBig(new(big.Int).Add(out[i].Minor(), one), a.cur)
		remainder.Sub(remainder, one)
	}
	return out, nil
}

// jsonAmount is the wire shape {"amount":"<decimal>","currency":"USD"}.
type jsonAmount struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// UnmarshalJSON parses {"amount","currency"} exactly (a known code resolves to its full
// Currency; an unknown code is kept as a bare code).
func (a *Amount) UnmarshalJSON(b []byte) error {
	var raw jsonAmount
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	d, err := decimal.Parse(raw.Amount)
	if err != nil {
		return err
	}
	cur, err := ByCode(raw.Currency)
	if err != nil {
		cur = Currency{Code: raw.Currency}
	}
	*a = Amount{v: d, cur: cur}
	return nil
}

// Value implements driver.Valuer — stored as "amount|currency" ("0.00132|USD"), exact and
// currency-carrying.
func (a Amount) Value() (driver.Value, error) { return a.v.String() + "|" + a.cur.Code, nil }

// Scan implements sql.Scanner for the "amount|currency" form.
func (a *Amount) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case nil:
		*a = Amount{}
		return nil
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("money: cannot scan %T", src)
	}
	parts := strings.SplitN(s, "|", 2)
	if len(parts) != 2 {
		return ErrInvalidAmount
	}
	d, err := decimal.Parse(parts[0])
	if err != nil {
		return err
	}
	cur, err := ByCode(parts[1])
	if err != nil {
		cur = Currency{Code: parts[1]}
	}
	*a = Amount{v: d, cur: cur}
	return nil
}
