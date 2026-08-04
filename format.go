// Copyright (c) 2026, Hanzo AI, Inc. MIT OR Apache-2.0.

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
// only, and for the last inch of an external API whose schema genuinely demands a JSON
// number; never round-trip money through it, and never store what it returns. When you
// want the amount as text, that is MajorString, which is exact.
func (a Amount) AsMajorUnits() float64 {
	f, _ := strconv.ParseFloat(a.v.String(), 64)
	return f
}

// majorParts rescales the value to the currency's decimals and splits it into its sign,
// its integer digits and its zero-padded fraction. It is the ONE place the currency's
// scale is applied, so the two renderings below cannot drift apart, and it is derived
// from the integer value — no float64 is involved.
func (a Amount) majorParts() (neg bool, intPart, frac string) {
	d := a.v.Rescale(a.cur.Decimals)
	neg = d.Sign() < 0
	s := d.Abs().String()

	intPart, frac = s, ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i+1:]
	}
	// pad the fraction to the currency's decimals for a stable, fixed-scale rendering
	for int32(len(frac)) < a.cur.Decimals {
		frac += "0"
	}
	return neg, intPart, frac
}

// MajorString renders the value in MAJOR units as a plain fixed-scale decimal string —
// "10.99", "1234567.89", "500" for a zero-decimal currency, "-19.99" for a refund. No
// symbol, no thousands separators: this is the form every payment API means when it
// documents an amount field as a decimal string, and it is the exact inverse of
// ParseMinor.
//
// String() is NOT this conversion, and reaching for it is the easy mistake. String()
// renders the value's own scale, so it drops trailing zeros: FromCents(100).String() is
// "1", not "1.00". Both parse back to the same amount — which is why the round-trip law
// holds for either — but an API handed "1" where its schema says two decimals is a
// support ticket, and a ledger export that renders $1.00 as "1" no longer reconciles
// column-wise. MajorString always pads to the currency's decimals.
//
// The other spelling to avoid is the hand-rolled one: print cents/100, a dot, then
// cents%100. Go's % keeps the sign of the dividend, so -1999 comes out as "-19.-99" and
// -1 as "0.-1" — and a refund is the most common negative amount there is. Deriving the
// digits from the rescaled integer has no such case.
func (a Amount) MajorString() string {
	neg, intPart, frac := a.majorParts()
	out := intPart
	if a.cur.Decimals > 0 {
		out += "." + frac
	}
	if neg {
		return "-" + out
	}
	return out
}

// Display renders the value at the currency's decimals with grouped thousands and the
// currency symbol ("$1,234.56", "¥10000", "100,003.01 HUSD" when no symbol) — the
// human-facing form of MajorString. Exact, derived from the integer value, not a float.
func (a Amount) Display() string {
	neg, intPart, frac := a.majorParts()
	out := group(intPart)
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
