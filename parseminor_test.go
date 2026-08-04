// Copyright (c) 2026, Hanzo AI, Inc. MIT OR Apache-2.0.

package money

import (
	"errors"
	"math"
	"math/big"
	"strconv"
	"testing"
)

// TestParseCents is the table three separate hand-rolled converters used to answer
// differently. Each case below is a real shape the DigitalOcean/AWS/GCP billing APIs
// emit, and the three copies this function replaces disagreed on half of them:
// "1,234.56" was 123456 in one and 0 in the other two, "-5.00" was 0, 0 and -500.
func TestParseCents(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int64
	}{
		// ── the ordinary case, and the one a float64 gets wrong ──────────────
		{"cent-precise price", "19.99", 1999},
		{"whole dollars", "12", 1200},
		{"trailing zero cent", "9.90", 990},
		{"sub-dollar", "0.29", 29},
		{"zero", "0", 0},
		{"zero with cents", "0.00", 0},

		// ── the thousands separator DO actually emits on large invoices ──────
		{"thousands separator", "1,234.56", 123456},
		{"multiple groups", "1,234,567.89", 123456789},
		{"grouped, no fraction", "12,345", 1234500},

		// ── a negative is a CREDIT. Swallowing it overstates spend. ──────────
		{"credit", "-5.00", -500},
		{"credit, cent-precise", "-19.99", -1999},
		{"grouped credit", "-1,234.56", -123456},

		// ── sign and space handling ──────────────────────────────────────────
		{"explicit plus", "+7.50", 750},
		{"surrounding space", "  42.42  ", 4242},

		// ── sub-cent input rounds half-away-from-zero, both directions ───────
		{"rounds up at the half", "0.005", 1},
		{"rounds down below the half", "0.004", 0},
		{"credit rounds away from zero", "-0.005", -1},
		{"third decimal", "1.234", 123},
		{"third decimal, up", "1.235", 124},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseCents(tc.in)
			if err != nil {
				t.Fatalf("ParseCents(%q) errored: %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("ParseCents(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

// TestParseCentsRejects pins the half that matters most: an amount that cannot be
// read is an ERROR, not a zero. Zero is a legal amount, so a converter that maps
// garbage to it hands the caller a confident "$0.00 spent" for a row it could not
// read at all — the spend disappears from the ledger and nothing reports it.
func TestParseCentsRejects(t *testing.T) {
	for _, in := range []string{
		"",            // absent field — a fact to handle, not a zero
		"   ",         // whitespace only
		"abc",         // not a number
		"12.34.56",    // not a decimal
		"$12.34",      // symbol: ParseAmount's job, not this one
		"12.34 USD",   // ditto for a code
		"1.2e3",       // exponent notation is not a money literal
		"--5",         // malformed sign
		"12,34,5.6.7", // salad
		"NaN",
		"Inf",
	} {
		t.Run(strconv.Quote(in), func(t *testing.T) {
			got, err := ParseCents(in)
			if err == nil {
				t.Fatalf("ParseCents(%q) = %d with no error — an unreadable amount became a legal one", in, got)
			}
			if !errors.Is(err, ErrInvalidAmount) {
				t.Fatalf("ParseCents(%q) error = %v, want it to match ErrInvalidAmount", in, err)
			}
			if got != 0 {
				t.Fatalf("ParseCents(%q) = %d alongside an error, want 0", in, got)
			}
		})
	}
}

// TestParseMinorHonorsCurrencyScale proves the general form is not a USD function in
// disguise: the scale comes from the currency, so a zero-decimal currency does not
// silently gain two.
func TestParseMinorHonorsCurrencyScale(t *testing.T) {
	cases := []struct {
		in   string
		cur  Currency
		want int64
	}{
		{"1234", JPY, 1234},               // 0 decimals: yen are already minor units
		{"1,234", JPY, 1234},              // separators still handled
		{"1234.6", JPY, 1235},             // rounds to the currency's precision
		{"12.34", USD, 1234},              // 2 decimals
		{"12.34", EUR, 1234},              // ditto
		{"-1234", JPY, -1234},             // credits survive at any scale
		{"0.000000000000000001", HUSD, 1}, // 18 decimals: one wei-equivalent
	}
	for _, tc := range cases {
		t.Run(tc.cur.Code+"/"+tc.in, func(t *testing.T) {
			got, err := ParseMinor(tc.in, tc.cur)
			if err != nil {
				t.Fatalf("ParseMinor(%q, %s) errored: %v", tc.in, tc.cur.Code, err)
			}
			if got != tc.want {
				t.Fatalf("ParseMinor(%q, %s) = %d, want %d", tc.in, tc.cur.Code, got, tc.want)
			}
		})
	}
}

// TestParseMinorOverflows guards the boundary an int64 return imposes. Wrapping is
// the one failure mode worse than refusing: it reports a balance of the wrong SIGN.
func TestParseMinorOverflows(t *testing.T) {
	// One minor unit past math.MaxInt64, written as major units.
	tooBig := new(big.Int).Add(big.NewInt(math.MaxInt64), big.NewInt(1))
	s := new(big.Int).Quo(tooBig, big.NewInt(100)).String() + "." +
		tooBig.String()[len(tooBig.String())-2:]

	if got, err := ParseMinor(s, USD); err == nil {
		t.Fatalf("ParseMinor(%q, USD) = %d with no error — an int64 cannot hold it", s, got)
	} else if !errors.Is(err, ErrOverflow) {
		t.Fatalf("ParseMinor(%q, USD) error = %v, want it to match ErrOverflow", s, err)
	}

	// The largest amount that DOES fit must still convert, so the guard is a
	// boundary and not a cap that rejects legitimate money.
	ok := strconv.FormatInt(math.MaxInt64/100, 10) + ".00"
	if _, err := ParseMinor(ok, USD); err != nil {
		t.Fatalf("ParseMinor(%q, USD) errored on an amount that fits: %v", ok, err)
	}
}

// TestParseMinorIsExactNotFloat is the property the three replaced converters could
// not hold. Every cent value in a wide sweep must round-trip through its decimal
// string EXACTLY; the float spelling (int64(f*100) / math.Round(f*100)) drifts,
// which is how prices like 19.99 became 1998.
func TestParseMinorIsExactNotFloat(t *testing.T) {
	for cents := int64(-20000); cents <= 20000; cents++ {
		s := FromCents(cents).String()
		got, err := ParseCents(s)
		if err != nil {
			t.Fatalf("ParseCents(%q) errored: %v", s, err)
		}
		if got != cents {
			t.Fatalf("ParseCents(%q) = %d, want %d — the conversion is not exact", s, got, cents)
		}
	}
}

// TestParseCentsInvertsFromCents states the relationship as a law: ParseCents undoes
// FromCents for every representable amount, including the extremes where a float64
// has no bit left to distinguish neighbouring cents.
func TestParseCentsInvertsFromCents(t *testing.T) {
	for _, cents := range []int64{
		0, 1, -1, 99, 100, -100, 1999, -1999, 123456, -123456,
		1 << 40, -(1 << 40), 1 << 53, (1 << 53) + 1, math.MaxInt64 / 100,
	} {
		s := FromCents(cents).String()
		got, err := ParseCents(s)
		if err != nil {
			t.Fatalf("ParseCents(%q) errored: %v", s, err)
		}
		if got != cents {
			t.Fatalf("ParseCents(FromCents(%d).String() = %q) = %d, want %d", cents, s, got, cents)
		}
	}
}
