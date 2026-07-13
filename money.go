// Package money is the ONE exact money value for the Hanzo stack: an amount in a
// currency, backed by github.com/hanzoai/decimal (big.Int fixed-point). Unlike the
// int64/uint64 money libraries (which cap around 19 digits — about $9.20 at 18 decimals),
// this holds a fiat cent AND an 18-decimal on-chain balance with equal exactness, because
// the value is a big.Int decimal. No float64 ever touches money.
//
// The design is the community-standard money pattern (never floats, integer-exact,
// Amount + Currency, Split/Allocate so a division loses no sub-unit) with the precision
// ceiling removed. An Amount is IMMUTABLE. Arithmetic between two Amounts requires the
// SAME currency, so you can never silently add USD to EUR.
package money

import (
	"fmt"
	"math/big"

	"github.com/hanzoai/decimal"
)

// Currency is a currency's identity: an ISO-4217-style code and the number of decimal
// places in its smallest unit (USD 2, JPY 0, USDC 6, an 18-decimal ERC-20 18). Custom
// currencies are allowed — construct one directly.
type Currency struct {
	Code     string
	Decimals int32
}

// Common currencies. Fiat use the ISO minor unit; crypto/stablecoins use their on-chain
// token decimals so an off-chain Amount and an on-chain balance share one scale.
var (
	USD  = Currency{"USD", 2}
	EUR  = Currency{"EUR", 2}
	GBP  = Currency{"GBP", 2}
	JPY  = Currency{"JPY", 0}
	USDC = Currency{"USDC", 6}
	USDT = Currency{"USDT", 6}
	ETH  = Currency{"ETH", 18}
	HUSD = Currency{"HUSD", 18} // Hanzo USD stablecoin (ERC-20, 18 decimals)
)

// Amount is an exact money value: a decimal in a currency.
type Amount struct {
	v   decimal.Decimal
	cur Currency
}

// New wraps an exact decimal value in a currency.
func New(v decimal.Decimal, cur Currency) Amount { return Amount{v: v, cur: cur} }

// Zero is 0 in the currency.
func Zero(cur Currency) Amount { return Amount{cur: cur} }

// FromMinor builds an Amount from an integer count of the currency's smallest unit
// (e.g. cents for USD): value = units × 10^-cur.Decimals.
func FromMinor(units int64, cur Currency) Amount {
	return Amount{v: decimal.New(units, cur.Decimals), cur: cur}
}

// FromMinorBig is FromMinor for a big.Int count (an on-chain wei balance).
func FromMinorBig(units *big.Int, cur Currency) Amount {
	return Amount{v: decimal.NewFromBig(units, cur.Decimals), cur: cur}
}

// FromCents builds a USD Amount from integer cents.
func FromCents(cents int64) Amount { return FromMinor(cents, USD) }

// Parse reads a decimal string ("6.60", "-0.00132") as an exact Amount in the currency.
func Parse(s string, cur Currency) (Amount, error) {
	d, err := decimal.Parse(s)
	if err != nil {
		return Amount{}, err
	}
	return Amount{v: d, cur: cur}, nil
}

// Currency returns the amount's currency.
func (a Amount) Currency() Currency { return a.cur }

// Decimal returns the exact underlying value.
func (a Amount) Decimal() decimal.Decimal { return a.v }

// Minor returns the value as a big.Int count of the currency's smallest unit
// (round-half-up on any sub-unit precision) — the storage/on-chain integer.
func (a Amount) Minor() *big.Int { return a.v.Rescale(a.cur.Decimals).Coef() }

// MinorString is the minor-unit integer as a decimal string (the canonical storage form).
func (a Amount) MinorString() string { return a.Minor().String() }

func (a Amount) sameCur(b Amount) error {
	if a.cur.Code != b.cur.Code {
		return fmt.Errorf("money: currency mismatch %s vs %s", a.cur.Code, b.cur.Code)
	}
	return nil
}

// Add returns a + b; both must be the same currency.
func (a Amount) Add(b Amount) (Amount, error) {
	if err := a.sameCur(b); err != nil {
		return Amount{}, err
	}
	return Amount{v: a.v.Add(b.v), cur: a.cur}, nil
}

// Sub returns a − b; both must be the same currency.
func (a Amount) Sub(b Amount) (Amount, error) {
	if err := a.sameCur(b); err != nil {
		return Amount{}, err
	}
	return Amount{v: a.v.Sub(b.v), cur: a.cur}, nil
}

// Mul scales the amount by an exact decimal factor (e.g. a token count or a fee rate),
// keeping the currency.
func (a Amount) Mul(factor decimal.Decimal) Amount {
	return Amount{v: a.v.Mul(factor), cur: a.cur}
}

// Neg returns −a.
func (a Amount) Neg() Amount { return Amount{v: a.v.Neg(), cur: a.cur} }

// Cmp reports −1,0,+1 as a <,==,> b (same currency required; panics on mismatch — use
// after checking Currency for untrusted inputs).
func (a Amount) Cmp(b Amount) int {
	if a.cur.Code != b.cur.Code {
		panic("money: Cmp across currencies " + a.cur.Code + "/" + b.cur.Code)
	}
	return a.v.Cmp(b.v)
}

// Sign reports −1,0,+1 as a <,==,> 0.
func (a Amount) Sign() int { return a.v.Sign() }

// IsZero reports whether a == 0.
func (a Amount) IsZero() bool { return a.v.IsZero() }

// IsNeg reports whether a < 0.
func (a Amount) IsNeg() bool { return a.v.Sign() < 0 }

// Split divides the amount into n parts that sum EXACTLY to the original at the currency's
// minor-unit granularity — the remainder is distributed one minor unit at a time to the
// leading parts, so no sub-unit is created or lost. n must be positive.
func (a Amount) Split(n int) ([]Amount, error) {
	if n <= 0 {
		return nil, fmt.Errorf("money: split into %d parts", n)
	}
	total := a.Minor() // integer minor units
	nb := big.NewInt(int64(n))
	base, rem := new(big.Int).QuoRem(total, nb, new(big.Int))
	remN := rem.Int64() // |rem| < n, fits int64
	neg := remN < 0
	if neg {
		remN = -remN
	}
	out := make([]Amount, n)
	for i := 0; i < n; i++ {
		part := new(big.Int).Set(base)
		if int64(i) < remN {
			if neg {
				part.Sub(part, decimalOne)
			} else {
				part.Add(part, decimalOne)
			}
		}
		out[i] = FromMinorBig(part, a.cur)
	}
	return out, nil
}

// String renders the exact decimal value ("0.00132").
func (a Amount) String() string { return a.v.String() }

// Display renders the value rounded to the currency's decimals with the code
// ("100003.01 USD") — the human form.
func (a Amount) Display() string {
	return a.v.Rescale(a.cur.Decimals).String() + " " + a.cur.Code
}

// MarshalJSON emits {"amount":"<decimal>","currency":"USD"} — exact strings, no float.
func (a Amount) MarshalJSON() ([]byte, error) {
	return []byte(`{"amount":"` + a.v.String() + `","currency":"` + a.cur.Code + `"}`), nil
}

var decimalOne = big.NewInt(1)
