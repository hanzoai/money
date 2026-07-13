# money

The one exact money type for Go — an `Amount` in a `Currency`, backed by
[`hanzoai/decimal`](https://github.com/hanzoai/decimal) (**big.Int** fixed-point).

Martin Fowler's Money pattern (never floats, integer-exact, `Add`/`Sub`/`Split`/`Allocate`,
same-currency safety) **without the precision ceiling**. int64/uint64 money libraries cap
around 19 digits (~`$9.20` at 18 decimals); this holds a fiat cent **and** an 18-decimal
on-chain balance exactly.

```go
price, _ := money.Parse("6.60", money.USD)
cost := price.Mul(decimal.New(200, 0)).Mul(decimal.MustParse("0.000001")) // "0.00132", never 0
bal, _ := money.FromCents(100).Sub(cost)                                  // "0.99868"

money.FromCents(123456).Display()             // "$1,234.56"
parts, _ := money.FromCents(10000).Allocate(50, 30, 20) // $50 / $30 / $20, nothing lost
a, _ := money.ParseAmount("$1,234.56")        // symbol/code/separator aware
money.FromHUSD(wei)                            // 18-dec on-chain stablecoin
```

## Features
- **Exact, uncapped**: big.Int core — fiat cents to 18-decimal crypto wei, no float, no ceiling
- **Arithmetic**: `Add` `Sub` `Mul` `Neg` `Cmp` `Sign`, same-currency safety (zero-value agnostic)
- **Allocation**: `Split(n)` and `Allocate(ratios…)` — no sub-unit ever lost
- **Aggregation**: `Sum` `Min` `Max` `Avg`
- **Exchange**: rate table + `Convert` / `ConvertWithRate`
- **Parsing**: `ParseAmount` (`$1,234.56`, `EUR 250,75`, `£99.99`), European decimal-comma, `ParseDecimal`
- **Formatting**: `Display` (symbol + thousands separators), `AsMajorUnits`
- **Currencies**: ISO-4217 set (symbol, numeric code) + crypto (USDC/USDT/BTC/ETH/HUSD) + `Register` custom
- **Integration**: `json.Marshaler`/`Unmarshaler`, `sql.Scanner`/`driver.Valuer`

## Packages
Single package `github.com/hanzoai/money` (`Amount`, `Currency`, `Exchange`, parse/format/aggregate),
on `github.com/hanzoai/decimal`.

Original work (big.Int core written from scratch, not a fork). BSD-3-Clause.
