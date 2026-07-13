# money

The one exact money type for Go: an `Amount` in a `Currency`, backed by
[`hanzoai/decimal`](https://github.com/hanzoai/decimal) (big.Int fixed-point).

The community money pattern — never floats, integer-exact, `Add`/`Sub`/`Split`/`Allocate`,
same-currency safety — **without the precision ceiling**. int64/uint64 money libraries cap
around 19 digits (~`$9.20` at 18 decimals); this holds a fiat cent *and* an 18-decimal
on-chain balance exactly.

```go
price, _ := money.Parse("6.60", money.USD)
cost := price.Mul(decimal.New(200, 0)).Mul(decimal.MustParse("0.000001")) // "0.00132", never 0
bal, _ := money.FromCents(100).Sub(cost)                                  // "0.99868"

husd, _ := money.Parse("100003.01", money.HUSD) // 18-dec on-chain unit
husd.MinorString()                              // "100003010000000000000000"

parts, _ := money.FromCents(100).Split(3)       // 34 + 33 + 33 = 100, nothing lost
```

Currencies carry their smallest-unit decimals (USD 2, USDC 6, HUSD/ETH 18). BSD-3-Clause.
