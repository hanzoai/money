// Copyright (c) 2026, Hanzo AI, Inc. MIT OR Apache-2.0.

package money

import (
	"fmt"
	"strings"

	"github.com/hanzoai/decimal"
)

// symbolCurrency maps a display symbol to its default currency (ambiguous symbols like $
// resolve to the most common — USD; pass an explicit code to disambiguate).
var symbolCurrency = map[string]Currency{
	"$": USD, "€": EUR, "£": GBP, "¥": JPY, "₿": BTC,
	"₹": INR, "₩": KRW, "₺": TRY, "₽": RUB, "R$": BRL,
}

// ParseAmount parses a human-entered money string — "$1,234.56", "EUR 250,75", "100.50 USD",
// "£99.99" — into an exact Amount. The currency comes from a leading symbol, or a code
// token (prefix or suffix), or the optional defaultCode. Thousands separators are handled;
// a comma-only decimal is treated as a thousands separator (use ParseAmountEuropean for
// "10,50" == 10.50).
func ParseAmount(s string, defaultCode ...string) (Amount, error) {
	return parseAmount(s, false, defaultCode...)
}

// ParseAmountEuropean is ParseAmount but treats a comma-only separator as the DECIMAL point
// ("€10,50" == €10.50).
func ParseAmountEuropean(s string, defaultCode ...string) (Amount, error) {
	return parseAmount(s, true, defaultCode...)
}

func parseAmount(s string, commaDecimal bool, defaultCode ...string) (Amount, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Amount{}, ErrInvalidAmount
	}
	var cur Currency
	found := false

	// leading symbol
	for sym, c := range symbolCurrency {
		if strings.HasPrefix(s, sym) {
			cur, found = c, true
			s = strings.TrimSpace(strings.TrimPrefix(s, sym))
			break
		}
	}
	// code token (prefix or suffix), e.g. "USD 100" or "100 USD"
	if fields := strings.Fields(s); len(fields) == 2 {
		if c, err := ByCode(fields[0]); err == nil {
			cur, found, s = c, true, fields[1]
		} else if c, err := ByCode(fields[1]); err == nil {
			cur, found, s = c, true, fields[0]
		}
	}
	if !found {
		if len(defaultCode) == 1 {
			c, err := ByCode(defaultCode[0])
			if err != nil {
				return Amount{}, err
			}
			cur, found = c, true
		} else {
			return Amount{}, ErrUnknownCurrency
		}
	}

	num, err := cleanNumber(s, commaDecimal)
	if err != nil {
		return Amount{}, err
	}
	d, err := decimal.Parse(num)
	if err != nil {
		return Amount{}, ErrInvalidAmount
	}
	return Amount{v: d, cur: cur}, nil
}

// ParseMinor parses a decimal string of MAJOR units — "19.99", "1,234.56", "-5.00" —
// into an exact integer count of cur's minor units. It is the inverse of FromMinor,
// and it is the conversion every ledger edge needs, because upstream billing APIs
// (DigitalOcean, AWS, GCP) report money as a decimal STRING while every ledger stores
// an integer.
//
// The whole point is that it goes nowhere near float64. The obvious spelling of this
// conversion — ParseFloat then multiply by 100 — loses a cent on ordinary prices:
// 19.99 has no exact binary representation, so f*100 is 1998.999… and truncation
// gives 1998. Rounding papers over that one, but the float is still an approximation
// of the money and the error is unbounded in the large: past 2^53 minor units the
// nearest float64 is not the amount at all. Parsing the digits exactly has no such
// regime.
//
// Errors are RETURNED, never absorbed. Mapping an unparseable amount to zero is the
// expensive bug in this family: zero is a legal amount, so a caller that cannot see
// the difference between "the account spent nothing" and "we could not read what the
// account spent" reports a confident $0 and the spend silently vanishes from the
// ledger. The caller decides what an unreadable row means; this function only
// reports that it was unreadable. An empty string is an error for the same reason —
// an absent field is a fact the caller must handle, not a zero.
//
// A NEGATIVE PARSES AND STAYS NEGATIVE. On these APIs a negative row is a credit or
// a refund, and clamping it to zero overstates spend by exactly the amount refunded.
// Rounding is half-away-from-zero, so a credit rounds like the charge it reverses.
func ParseMinor(s string, cur Currency) (int64, error) {
	d, err := ParseDecimal(s)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrInvalidAmount, s)
	}
	minor := d.Rescale(cur.Decimals).Coef()
	if !minor.IsInt64() {
		return 0, fmt.Errorf("%w: %q is %s minor units of %s", ErrOverflow, s, minor, cur.Code)
	}
	return minor.Int64(), nil
}

// ParseCents is ParseMinor for USD — the inverse of FromCents, and the spelling the
// dollar-denominated cost APIs want.
func ParseCents(s string) (int64, error) { return ParseMinor(s, USD) }

// ParseDecimal parses just the numeric part (separators handled) — no currency.
func ParseDecimal(s string) (decimal.Decimal, error) {
	num, err := cleanNumber(strings.TrimSpace(s), false)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return decimal.Parse(num)
}

// cleanNumber normalizes thousands/decimal separators to a plain decimal string.
func cleanNumber(s string, commaDecimal bool) (string, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, " ", ""))
	if s == "" {
		return "", ErrInvalidAmount
	}
	dot, comma := strings.LastIndexByte(s, '.'), strings.LastIndexByte(s, ',')
	switch {
	case dot >= 0 && comma >= 0:
		// the later separator is the decimal point; the other is thousands.
		if dot > comma {
			s = strings.ReplaceAll(s, ",", "") // US: 1,234.56
		} else {
			s = strings.ReplaceAll(s, ".", "") // EU: 1.234,56
			s = strings.Replace(s, ",", ".", 1)
		}
	case comma >= 0:
		if commaDecimal {
			s = strings.Replace(s, ",", ".", 1)
		} else {
			s = strings.ReplaceAll(s, ",", "") // thousands
		}
	}
	return s, nil
}
