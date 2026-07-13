// Copyright (c) 2026, Hanzo AI, Inc. BSD-3-Clause.

package money

import (
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
