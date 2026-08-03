// Copyright (c) 2026, Hanzo AI, Inc. BSD-3-Clause.

package money

import "testing"

// TestMajorStringIsFixedScale states the property that makes MajorString a
// different thing from String: the rendering is padded to the currency's
// decimals, always.
//
// String() renders the value's OWN scale and therefore drops trailing zeros —
// FromCents(100).String() is "1". Handing "1" to a payment API that documents
// its amount field as a decimal of major units is how a $1.00 charge becomes a
// support ticket, so the wire needs a rendering that cannot do that.
func TestMajorStringIsFixedScale(t *testing.T) {
	for _, tc := range []struct {
		minor int64
		cur   Currency
		want  string
	}{
		{1099, USD, "10.99"},
		{100, USD, "1.00"},   // String() gives "1"
		{1000, USD, "10.00"}, // String() gives "10"
		{0, USD, "0.00"},     // String() gives "0"
		{1, USD, "0.01"},
		{5, USD, "0.05"},
		{123456789, USD, "1234567.89"}, // no thousands separators on the wire
		// A zero-decimal currency must NOT gain two: ¥500 is 500 yen.
		{500, JPY, "500"},
		{0, JPY, "0"},
		// A six-decimal token pads to six.
		{1000000, USDC, "1.000000"},
		{1, USDC, "0.000001"},
	} {
		if got := FromMinor(tc.minor, tc.cur).MajorString(); got != tc.want {
			t.Errorf("FromMinor(%d, %s).MajorString() = %q, want %q", tc.minor, tc.cur.Code, got, tc.want)
		}
	}
}

// TestMajorStringNegativesAreWellFormed pins the case every hand-rolled
// cents→string helper gets wrong. The obvious spelling — print cents/100, a
// dot, then cents%100 — emits a SECOND minus sign for the fraction, because Go's
// % keeps the sign of the dividend: -1999 renders as "-19.-99". A refund is the
// most common negative amount in a payment system, so the bug ships straight to
// the credit path.
func TestMajorStringNegativesAreWellFormed(t *testing.T) {
	for _, tc := range []struct {
		minor int64
		cur   Currency
		want  string
	}{
		{-1999, USD, "-19.99"}, // the naive helper gives "-19.-99"
		{-1, USD, "-0.01"},     // the naive helper gives "0.-1"
		{-5, USD, "-0.05"},     // the naive helper gives "0.-5"
		{-100, USD, "-1.00"},
		{-10000, USD, "-100.00"},
		{-500, JPY, "-500"},
	} {
		if got := FromMinor(tc.minor, tc.cur).MajorString(); got != tc.want {
			t.Errorf("FromMinor(%d, %s).MajorString() = %q, want %q", tc.minor, tc.cur.Code, got, tc.want)
		}
	}
}

// TestMajorStringInvertsParseMinor states the round trip as a law: MajorString
// is the exact reverse of ParseMinor, in both directions, with no float64
// anywhere between them — including past 2^53 minor units, where a float64 has
// no bit left to tell neighbouring cents apart.
func TestMajorStringInvertsParseMinor(t *testing.T) {
	for _, cur := range []Currency{USD, JPY, USDC} {
		for _, minor := range []int64{
			0, 1, -1, 99, -99, 100, -100, 1999, -1999, 123456, -123456,
			1 << 40, -(1 << 40), 1 << 53, (1 << 53) + 1,
		} {
			s := FromMinor(minor, cur).MajorString()
			got, err := ParseMinor(s, cur)
			if err != nil {
				t.Fatalf("ParseMinor(%q, %s) errored: %v", s, cur.Code, err)
			}
			if got != minor {
				t.Fatalf("ParseMinor(FromMinor(%d, %s).MajorString() = %q) = %d, want %d",
					minor, cur.Code, s, got, minor)
			}
		}
	}
}

// TestDisplayAgreesWithMajorString keeps the two renderings from drifting:
// Display is MajorString plus thousands separators and the symbol, so stripping
// those must give MajorString back. They share one scale-and-pad step for
// exactly this reason.
func TestDisplayAgreesWithMajorString(t *testing.T) {
	for _, minor := range []int64{0, 1, -1, 1099, -1099, 123456789, -123456789} {
		a := FromCents(minor)
		plain, disp := a.MajorString(), a.Display()
		want := "$" + plain
		if minor < 0 {
			want = "-$" + plain[1:] // the sign leads, then the symbol
		}
		// Display groups thousands; compare with the separators removed.
		stripped := ""
		for _, r := range disp {
			if r != ',' {
				stripped += string(r)
			}
		}
		if stripped != want {
			t.Errorf("FromCents(%d): Display()=%q strips to %q, want %q (MajorString %q)",
				minor, disp, stripped, want, plain)
		}
	}
}
