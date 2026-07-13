// Copyright (c) 2026, Hanzo AI, Inc. BSD-3-Clause.

package money

import (
	"math/big"
	"strconv"
	"strings"

	"github.com/hanzoai/decimal"
)

// FromFloat builds an Amount from a float — USE ONLY for converting user input; floats are
// imprecise, so prefer Parse for exact decimals. The float is rendered at the currency's
// decimals and parsed exactly from there.
func FromFloat(f float64, cur Currency) Amount {
	s := strconv.FormatFloat(f, 'f', int(cur.Decimals), 64)
	d, _ := decimal.Parse(s)
	return Amount{v: d, cur: cur}
}

// AsMajorUnits returns the value as a float in major units (e.g. dollars) — for DISPLAY
// only; never round-trip money through it.
func (a Amount) AsMajorUnits() float64 {
	f, _ := strconv.ParseFloat(a.v.String(), 64)
	return f
}

// Display renders the value at the currency's decimals with grouped thousands and the
// currency symbol ("$1,234.56", "¥10000", "100,003.01 HUSD" when no symbol). Exact —
// derived from the integer value, not a float.
func (a Amount) Display() string {
	d := a.v.Rescale(a.cur.Decimals)
	neg := d.Sign() < 0
	s := d.Abs().String()

	intPart, frac := s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i+1:]
	}
	// pad fraction to the currency's decimals for a stable display
	for int32(len(frac)) < a.cur.Decimals {
		frac += "0"
	}
	grouped := group(intPart)
	out := grouped
	if a.cur.Decimals > 0 {
		out += "." + frac
	}
	sym := a.cur.Symbol
	switch {
	case neg && sym != "":
		return "-" + sym + out
	case sym != "":
		return sym + out
	case neg:
		return "-" + out + " " + a.cur.Code
	default:
		return out + " " + a.cur.Code
	}
}

// group inserts thousands separators into a non-negative integer string.
func group(s string) string {
	n := len(s)
	if n <= 3 {
		return s
	}
	var b strings.Builder
	pre := n % 3
	if pre > 0 {
		b.WriteString(s[:pre])
		if n > pre {
			b.WriteByte(',')
		}
	}
	for i := pre; i < n; i += 3 {
		b.WriteString(s[i : i+3])
		if i+3 < n {
			b.WriteByte(',')
		}
	}
	return b.String()
}

// FromUSD builds a USD amount from integer cents. The FromXXX helpers mirror gocanto's
// convenience constructors; they take the currency's smallest unit.
func FromUSD(minor int64) Amount { return FromMinor(minor, USD) }

// FromEUR builds a EUR amount from integer cents.
func FromEUR(minor int64) Amount { return FromMinor(minor, EUR) }

// FromGBP builds a GBP amount from integer pence.
func FromGBP(minor int64) Amount { return FromMinor(minor, GBP) }

// FromJPY builds a JPY amount from integer yen (0 decimals).
func FromJPY(minor int64) Amount { return FromMinor(minor, JPY) }

// FromHUSD builds a Hanzo-USD amount from integer 18-decimal wei.
func FromHUSD(wei *big.Int) Amount { return FromMinorBig(wei, HUSD) }
