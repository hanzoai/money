package money

import (
	"testing"

	"github.com/hanzoai/decimal"
)

func TestExactBilling(t *testing.T) {
	// 200 tokens × $6.60/1M = 0.00132 exactly — cents-flooring dropped this to 0.
	price, _ := Parse("6.60", USD)
	cost := price.Mul(decimal.New(200, 0)).Mul(decimal.MustParse("0.000001")) // ÷1e6
	if cost.String() != "0.00132" || cost.IsZero() {
		t.Fatalf("token cost = %q", cost.String())
	}
	bal := FromCents(100) // $1.00
	after, err := bal.Sub(cost)
	if err != nil {
		t.Fatal(err)
	}
	if after.String() != "0.99868" {
		t.Fatalf("$1 - 0.00132 = %q", after.String())
	}
}

func TestMinorAndCurrency(t *testing.T) {
	if FromCents(1).MinorString() != "1" {
		t.Fatal("1 cent minor")
	}
	// HUSD is 18-dec: $100,003.01 → minor units = 24-digit integer (int64 can't hold this)
	h, _ := Parse("100003.01", HUSD)
	if h.MinorString() != "100003010000000000000000" {
		t.Fatalf("HUSD minor = %s", h.MinorString())
	}
	// cross-currency add is refused
	if _, err := FromCents(100).Add(Zero(EUR)); err == nil {
		t.Fatal("expected currency-mismatch error")
	}
}

func TestSplitExact(t *testing.T) {
	parts, err := FromCents(100).Split(3) // $1.00 / 3
	if err != nil {
		t.Fatal(err)
	}
	sum := Zero(USD)
	for _, p := range parts {
		sum, _ = sum.Add(p)
	}
	if sum.MinorString() != "100" { // 34+33+33 = 100, nothing lost
		t.Fatalf("split sum = %s", sum.MinorString())
	}
	if parts[0].MinorString() != "34" || parts[2].MinorString() != "33" {
		t.Fatalf("split = %s,%s,%s", parts[0].MinorString(), parts[1].MinorString(), parts[2].MinorString())
	}
}
