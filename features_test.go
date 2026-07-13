package money

import (
	"encoding/json"
	"testing"
)

func TestDisplay(t *testing.T) {
	if got := FromCents(123456).Display(); got != "$1,234.56" {
		t.Errorf("USD display = %q", got)
	}
	if got := FromMinor(10000, JPY).Display(); got != "¥10,000" {
		t.Errorf("JPY display = %q", got)
	}
	if got := FromCents(-500).Display(); got != "-$5.00" {
		t.Errorf("neg display = %q", got)
	}
}

func TestAllocate(t *testing.T) {
	parts, err := FromCents(10000).Allocate(50, 30, 20) // $100 by 50/30/20
	if err != nil {
		t.Fatal(err)
	}
	if parts[0].Display() != "$50.00" || parts[1].Display() != "$30.00" || parts[2].Display() != "$20.00" {
		t.Fatalf("allocate = %s %s %s", parts[0].Display(), parts[1].Display(), parts[2].Display())
	}
	// remainder distribution: $100/3 by equal ratios → 34,33,33
	p2, _ := FromCents(10000).Allocate(1, 1, 1)
	sum, _ := Sum(p2...)
	if sum.MinorString() != FromCents(10000).MinorString() {
		t.Fatal("allocate lost pennies")
	}
}

func TestAggregate(t *testing.T) {
	xs := []Amount{FromCents(10000), FromCents(20000), FromCents(30000)}
	if s, _ := Sum(xs...); s.Display() != "$600.00" {
		t.Errorf("sum")
	}
	if m, _ := Min(xs...); m.Display() != "$100.00" {
		t.Errorf("min")
	}
	if m, _ := Max(xs...); m.Display() != "$300.00" {
		t.Errorf("max")
	}
	if a, _ := Avg(xs...); a.Display() != "$200.00" {
		t.Errorf("avg")
	}
}

func TestExchange(t *testing.T) {
	ex := NewExchange()
	_ = ex.AddRate(USD, EUR, "0.85")
	eur, err := ex.Convert(FromCents(10000), EUR) // $100 → €85
	if err != nil || eur.Display() != "€85.00" {
		t.Fatalf("convert = %v %s", err, eur.Display())
	}
}

func TestParse(t *testing.T) {
	for _, tc := range []struct{ in, want, cur string }{
		{"$1,234.56", "1234.56", "USD"},
		{"EUR 250.75", "250.75", "EUR"},
		{"100.50 USD", "100.5", "USD"},
		{"£99.99", "99.99", "GBP"},
	} {
		a, err := ParseAmount(tc.in)
		if err != nil {
			t.Fatalf("ParseAmount(%q): %v", tc.in, err)
		}
		if a.String() != tc.want || a.Currency().Code != tc.cur {
			t.Errorf("ParseAmount(%q) = %s %s", tc.in, a.String(), a.Currency().Code)
		}
	}
	eu, _ := ParseAmountEuropean("€10,50")
	if eu.String() != "10.5" {
		t.Errorf("european = %s", eu.String())
	}
}

func TestJSONAndSQL(t *testing.T) {
	a := FromCents(2999)
	b, _ := json.Marshal(a)
	if string(b) != `{"amount":"29.99","currency":"USD"}` {
		t.Fatalf("marshal = %s", b)
	}
	var back Amount
	if err := json.Unmarshal(b, &back); err != nil || back.Cmp(a) != 0 || back.Currency().Code != "USD" {
		t.Fatalf("json round-trip: %v %s", err, back.String())
	}
	v, _ := a.Value()
	if v.(string) != "29.99|USD" {
		t.Fatalf("sql value = %v", v)
	}
	var s Amount
	if err := s.Scan("29.99|USD"); err != nil || s.Cmp(a) != 0 {
		t.Fatalf("sql scan: %v", err)
	}
}
