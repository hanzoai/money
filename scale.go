// Copyright (c) 2026, Hanzo AI, Inc. BSD-3-Clause.

package money

import (
	"fmt"
	"math"
	"strconv"

	"github.com/hanzoai/decimal"
)

// ScaleMinor scales an integer count of minor units by an exact rate and returns whole
// minor units — a percentage off a price, a commission on a charge, a tax on a subtotal.
// It is the third member of the ledger-edge set alongside ParseMinor (string → minor) and
// MajorString (minor → string): minor × rate → minor, the only one of the three that has
// to decide what to do with a fraction of a cent.
//
// It decides by ROUNDING HALF AWAY FROM ZERO, the same rule ParseMinor and Rescale use, so
// one amount cannot depend on which conversion it went through. The obvious spellings
// instead TRUNCATE — int64(float64(cents)*rate) drops the fraction toward zero, and
// math.Floor drops it toward negative infinity — and truncation is not a small error but a
// biased one: 10% off $19.99 is $1.999, which truncates to $1.99 and rounds to $2.00, and
// the missing cent always falls on the same side of the trade. Across a price book that is
// a systematic half-cent per line, never random, never self-cancelling.
//
// A rate is dimensionless, so no currency is needed: one minor unit scaled by 0.1 is a
// tenth of a minor unit whatever the currency's decimals are. Build the rate exactly —
// decimal.New(10, 2) for 10%, decimal.New(1500, 4) for 1500 basis points, RateFromFloat
// for a float64 that a legacy schema handed you.
//
// The result is computed in big.Int, so the rate is applied exactly at any magnitude; only
// the final answer has to fit an int64, and if it does not that is REPORTED. A scaled
// amount that silently wraps is a balance nobody holds.
func ScaleMinor(units int64, rate decimal.Decimal) (int64, error) {
	scaled := decimal.New(units, 0).Mul(rate).Rescale(0).Coef()
	if !scaled.IsInt64() {
		return 0, fmt.Errorf("%w: %d minor units scaled by %s is %s", ErrOverflow, units, rate, scaled)
	}
	return scaled.Int64(), nil
}

// RateFromFloat recovers the exact decimal that a float64 rate was meant to be: the
// shortest decimal which round-trips to that float, so the 0.1 a human typed comes back as
// 0.1 and not as 0.1000000000000000055511151231257827.
//
// It exists because schemas predating this package store rates as float64 — a percent
// column, a commission, a tax rate — and the value has to become exact somewhere before it
// touches money. That conversion has exactly one correct spelling. Rendering with a fixed
// precision instead (%.2f and friends) silently truncates any rate finer than the format:
// a 0.375% fee becomes 0.38%, or with two decimals of a fraction, 0.00.
//
// This does not make the stored float exact — nothing can, the information is already
// gone. It recovers the decimal the float was the nearest representation OF, which for a
// human-entered rate is the number that was meant. NaN and ±Inf are not rates and are
// reported, never mapped to zero: a zero rate is a legal rate, and a caller that cannot
// tell "no discount" from "unreadable discount" charges full price and calls it a sale.
func RateFromFloat(f float64) (decimal.Decimal, error) {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return decimal.Decimal{}, fmt.Errorf("%w: %v is not a rate", ErrInvalidAmount, f)
	}
	return decimal.Parse(strconv.FormatFloat(f, 'f', -1, 64))
}
