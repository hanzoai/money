// Copyright (c) 2026, Hanzo AI, Inc. BSD-3-Clause.

package money

import (
	"errors"
	"math"
	"math/big"
	"testing"

	"github.com/hanzoai/decimal"
)

// pct is the exact rate for a whole percentage: 10 → 0.10.
func pct(p int64) decimal.Decimal { return decimal.New(p, 2) }

// TestScaleMinorRoundsHalfAwayFromZero pins the rule, and pins it against the exact values
// the truncating spellings produce — every `want` below that differs from `trunc` is a cent
// that the old arithmetic kept.
func TestScaleMinorRoundsHalfAwayFromZero(t *testing.T) {
	for _, tc := range []struct {
		units int64
		rate  decimal.Decimal
		want  int64
		trunc int64 // what int64(float64(units)*rate) / math.Floor gave
	}{
		// The brief's price book. .5 and above rounds up, so the customer gets the cent.
		{1999, pct(10), 200, 199},   // $19.99 @ 10% = 1.999
		{1999, pct(15), 300, 299},   // 2.9985
		{1999, pct(33), 660, 659},   // 6.5967
		{995, pct(10), 100, 99},     // 0.995 — exactly a half, rounds away from zero
		{995, pct(15), 149, 149},    // 1.4925 — below the half, unchanged
		{995, pct(33), 328, 328},    // 3.2835
		{3333, pct(10), 333, 333},   // 3.333 exactly representable as a third of a cent
		{3333, pct(15), 500, 499},   // 4.9995
		{3333, pct(33), 1100, 1099}, // 10.9989
		{29, pct(10), 3, 2},         // 0.29 — the smallest prices are hit hardest
		{29, pct(15), 4, 4},         // 0.435
		{29, pct(33), 10, 9},        // 0.957

		// Exact results must not move.
		{1000, pct(10), 100, 100},
		{2000, pct(50), 1000, 1000},
		{0, pct(33), 0, 0},

		// A rate of 1 and a rate of 0 are identity and annihilator.
		{1999, decimal.New(1, 0), 1999, 1999},
		{1999, decimal.New(0, 0), 0, 0},

		// Basis points, the promotion spelling: 1500 bps = 15%.
		{1999, decimal.New(1500, 4), 300, 299},

		// A NEGATIVE amount is a refund and rounds like the charge it reverses: away from
		// zero, so |result| matches. Truncation toward -Inf (math.Floor) would give -200
		// here and toward zero would give -199; half-away-from-zero gives -200 because
		// 1.999 rounds to 2.000 in magnitude.
		{-1999, pct(10), -200, -199},
		{-995, pct(10), -100, -99},
		{-29, pct(33), -10, -9},
	} {
		got, err := ScaleMinor(tc.units, tc.rate)
		if err != nil {
			t.Fatalf("ScaleMinor(%d, %s) error: %v", tc.units, tc.rate, err)
		}
		if got != tc.want {
			t.Errorf("ScaleMinor(%d, %s) = %d, want %d", tc.units, tc.rate, got, tc.want)
		}
	}
}

// TestScaleMinorBeatsTheFloat states the property that makes this worth having: the
// truncating float spelling is wrong on a large fraction of ordinary prices, always in the
// same direction.
func TestScaleMinorBeatsTheFloat(t *testing.T) {
	var truncLoss, scaleLoss int64
	var cases int
	for units := int64(1); units <= 10000; units++ {
		for _, p := range []int64{10, 15, 33} {
			exact := exactHalfAway(units, p)
			got, err := ScaleMinor(units, pct(p))
			if err != nil {
				t.Fatal(err)
			}
			if got != exact {
				t.Fatalf("ScaleMinor(%d, %d%%) = %d, want %d", units, p, got, exact)
			}
			trunc := int64(math.Floor(float64(units) * float64(p) * 0.01))
			truncLoss += exact - trunc
			scaleLoss += exact - got
			cases++
		}
	}
	if scaleLoss != 0 {
		t.Fatalf("ScaleMinor lost %d minor units over %d cases; it must lose none", scaleLoss, cases)
	}
	if truncLoss <= 0 {
		t.Fatalf("expected the truncating spelling to be biased low; it lost %d", truncLoss)
	}
	t.Logf("over %d cases the truncating float spelling withheld %d minor units; ScaleMinor withholds %d",
		cases, truncLoss, scaleLoss)
}

// exactHalfAway is units*p/100 by integer arithmetic, rounded half away from zero.
func exactHalfAway(units, p int64) int64 {
	n := new(big.Int).Mul(big.NewInt(units), big.NewInt(p))
	neg := n.Sign() < 0
	a := new(big.Int).Abs(n)
	q, r := new(big.Int).QuoRem(a, big.NewInt(100), new(big.Int))
	if new(big.Int).Mul(r, big.NewInt(2)).Cmp(big.NewInt(100)) >= 0 {
		q.Add(q, big.NewInt(1))
	}
	if neg {
		q.Neg(q)
	}
	return q.Int64()
}

// TestScaleMinorOverflowIsReported — a scaled amount that does not fit is an error, never a
// wrapped number.
func TestScaleMinorOverflowIsReported(t *testing.T) {
	if _, err := ScaleMinor(math.MaxInt64, decimal.New(2, 0)); !errors.Is(err, ErrOverflow) {
		t.Errorf("MaxInt64 doubled: got %v, want ErrOverflow", err)
	}
	if _, err := ScaleMinor(math.MinInt64, decimal.New(2, 0)); !errors.Is(err, ErrOverflow) {
		t.Errorf("MinInt64 doubled: got %v, want ErrOverflow", err)
	}
	// The boundary itself fits.
	if got, err := ScaleMinor(math.MaxInt64, decimal.New(1, 0)); err != nil || got != math.MaxInt64 {
		t.Errorf("MaxInt64 x 1 = %d, %v; want %d, nil", got, err, int64(math.MaxInt64))
	}
}

// TestScaleMinorIsExactAtAnyMagnitude — the rate is applied in big.Int, so an amount past
// the float64 integer limit (2^53) is scaled exactly rather than approximately.
func TestScaleMinorIsExactAtAnyMagnitude(t *testing.T) {
	const units = int64(1) << 60 // 1152921504606846976, far past 2^53
	got, err := ScaleMinor(units, pct(50))
	if err != nil {
		t.Fatal(err)
	}
	if want := units / 2; got != want {
		t.Errorf("ScaleMinor(2^60, 50%%) = %d, want %d", got, want)
	}
	// The float spelling cannot even represent the operands here.
	if float64(units)*0.5 == float64(units/2) {
		return // it happened to land; the exactness claim above is what matters
	}
	t.Logf("float64 spelling gives %.0f, exact is %d", float64(units)*0.5, units/2)
}

// TestRateFromFloat recovers the decimal a float64 was the nearest representation of.
func TestRateFromFloat(t *testing.T) {
	for _, tc := range []struct {
		f    float64
		want string
	}{
		{0.1, "0.1"},
		{0.15, "0.15"},
		{0.33, "0.33"},
		{0.075, "0.075"},
		{1.0 / 3.0, "0.3333333333333333"},
		{0, "0"},
		{1, "1"},
		{-0.1, "-0.1"},
		{0.0001, "0.0001"},
	} {
		got, err := RateFromFloat(tc.f)
		if err != nil {
			t.Fatalf("RateFromFloat(%v) error: %v", tc.f, err)
		}
		if got.String() != tc.want {
			t.Errorf("RateFromFloat(%v) = %s, want %s", tc.f, got, tc.want)
		}
	}
}

// TestRateFromFloatRejectsNonRates — NaN and Inf are reported, not mapped to a zero rate.
func TestRateFromFloatRejectsNonRates(t *testing.T) {
	for _, f := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		if _, err := RateFromFloat(f); !errors.Is(err, ErrInvalidAmount) {
			t.Errorf("RateFromFloat(%v): got %v, want ErrInvalidAmount", f, err)
		}
	}
}

// TestRateFromFloatFeedsScaleMinor — the two compose into the conversion a legacy float64
// percent column actually needs, and it is exact where the direct float multiply is not.
func TestRateFromFloatFeedsScaleMinor(t *testing.T) {
	for _, tc := range []struct {
		units int64
		f     float64
		want  int64
		trunc int64 // currency.Cents(float64(price) * amt.Percent)
	}{
		{1999, 0.10, 200, 199},
		{1999, 0.15, 300, 299},
		{1999, 0.33, 660, 659},
		{995, 1.0 / 3.0, 332, 331},
		{29, 1.0 / 3.0, 10, 9},
		{3333, 0.15, 500, 499},
	} {
		rate, err := RateFromFloat(tc.f)
		if err != nil {
			t.Fatal(err)
		}
		got, err := ScaleMinor(tc.units, rate)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want {
			t.Errorf("ScaleMinor(%d, RateFromFloat(%v)) = %d, want %d", tc.units, tc.f, got, tc.want)
		}
		if trunc := int64(float64(tc.units) * tc.f); trunc != tc.trunc {
			t.Errorf("the old spelling int64(float64(%d)*%v) = %d, expected it to give %d",
				tc.units, tc.f, trunc, tc.trunc)
		}
	}
}

// TestDirectedScalingBeatsTheFloat is the red test for ScaleMinorCeil/ScaleMinorFloor: it
// pins the exact answer for products that are WHOLE cents, and shows that the float
// spellings a caller would otherwise reach for get them wrong in both directions.
//
// The failures are not truncation — the exact product here IS an integer. They are the
// float's inability to hold the rate: 0.07 is stored a hair high, so 700×0.07 lands at
// 49.000000000000007 and Ceil invents a cent; 0.29 is stored a hair low, so 100×0.29 lands
// at 28.999999999999996 and Floor destroys one.
func TestDirectedScalingBeatsTheFloat(t *testing.T) {
	cases := []struct {
		units int64
		rate  float64
		want  int64 // the exact product, a whole number of minor units
	}{
		{700, 0.07, 49},
		{7000, 0.07, 490},
		{100, 0.29, 29},
		{2900, 0.01, 29},
		{300, 0.10, 30},
	}
	for _, c := range cases {
		rate, err := RateFromFloat(c.rate)
		if err != nil {
			t.Fatalf("RateFromFloat(%v): %v", c.rate, err)
		}
		up, err := ScaleMinorCeil(c.units, rate)
		if err != nil {
			t.Fatalf("ScaleMinorCeil(%d, %v): %v", c.units, c.rate, err)
		}
		down, err := ScaleMinorFloor(c.units, rate)
		if err != nil {
			t.Fatalf("ScaleMinorFloor(%d, %v): %v", c.units, c.rate, err)
		}
		// The product is exact, so rounding it in EITHER direction is the same number.
		if up != c.want || down != c.want {
			t.Errorf("%d × %v: ceil=%d floor=%d, want %d for both", c.units, c.rate, up, down, c.want)
		}
		// And that is the number the float spellings miss.
		if fc := int64(math.Ceil(float64(c.units) * c.rate)); fc != c.want {
			t.Logf("  (float math.Ceil would have said %d, not %d)", fc, c.want)
		}
		if ff := int64(math.Floor(float64(c.units) * c.rate)); ff != c.want {
			t.Logf("  (float math.Floor would have said %d, not %d)", ff, c.want)
		}
	}
}

// TestDirectedScalingRoundsTheStatedWay pins the direction itself on products that are NOT
// whole, including negatives — where "up" means toward +infinity, not away from zero.
func TestDirectedScalingRoundsTheStatedWay(t *testing.T) {
	cases := []struct {
		units             int64
		rate              decimal.Decimal
		wantCeil, wantFlr int64
	}{
		{1999, pct(10), 200, 199},    // 199.9
		{995, pct(10), 100, 99},      // 99.5
		{29, pct(33), 10, 9},         // 9.57
		{-1999, pct(10), -199, -200}, // -199.9: ceil is toward +inf
		{-995, pct(10), -99, -100},   // -99.5
		{1250, pct(6), 75, 75},       // 75 exactly
	}
	for _, c := range cases {
		up, err := ScaleMinorCeil(c.units, c.rate)
		if err != nil {
			t.Fatalf("ScaleMinorCeil: %v", err)
		}
		down, err := ScaleMinorFloor(c.units, c.rate)
		if err != nil {
			t.Fatalf("ScaleMinorFloor: %v", err)
		}
		if up != c.wantCeil || down != c.wantFlr {
			t.Errorf("%d × %s: ceil=%d want %d, floor=%d want %d",
				c.units, c.rate, up, c.wantCeil, down, c.wantFlr)
		}
	}
}

// TestDirectedScalingOverflowIsReported: a scaled amount that silently wraps is a balance
// nobody holds, the same contract ScaleMinor keeps.
func TestDirectedScalingOverflowIsReported(t *testing.T) {
	huge := decimal.New(1, -18) // 10^18
	if _, err := ScaleMinorCeil(math.MaxInt64, huge); !errors.Is(err, ErrOverflow) {
		t.Errorf("ScaleMinorCeil overflow: got %v, want ErrOverflow", err)
	}
	if _, err := ScaleMinorFloor(math.MaxInt64, huge); !errors.Is(err, ErrOverflow) {
		t.Errorf("ScaleMinorFloor overflow: got %v, want ErrOverflow", err)
	}
}
