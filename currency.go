// Copyright (c) 2026, Hanzo AI, Inc. BSD-3-Clause.

package money

import "strings"

// Currency is a currency's identity: its ISO-4217 (or custom) code, the number of
// decimal places in its smallest unit (Decimals — USD 2, JPY 0, an 18-decimal ERC-20 18),
// a display Symbol, and the ISO numeric code where applicable. Custom currencies (crypto,
// loyalty points) are first-class — construct one directly or Register it.
type Currency struct {
	Code     string // ISO-4217 alpha code or custom ("USD", "HUSD")
	Decimals int32  // fractional digits of the smallest unit
	Symbol   string // display grapheme ("$", "€", "₿")
	Numeric  string // ISO-4217 numeric code ("840"); "" for custom
	Name     string
}

// registry holds all known currencies by code; seeded with the tables below and
// extendable via Register.
var registry = map[string]Currency{}

func reg(c Currency) Currency {
	registry[strings.ToUpper(c.Code)] = c
	return c
}

// Fiat (ISO 4217) — the common set; extend via Register for the long tail.
var (
	USD = reg(Currency{"USD", 2, "$", "840", "US Dollar"})
	EUR = reg(Currency{"EUR", 2, "€", "978", "Euro"})
	GBP = reg(Currency{"GBP", 2, "£", "826", "Pound Sterling"})
	JPY = reg(Currency{"JPY", 0, "¥", "392", "Yen"})
	CNY = reg(Currency{"CNY", 2, "¥", "156", "Yuan Renminbi"})
	CHF = reg(Currency{"CHF", 2, "Fr", "756", "Swiss Franc"})
	CAD = reg(Currency{"CAD", 2, "$", "124", "Canadian Dollar"})
	AUD = reg(Currency{"AUD", 2, "$", "036", "Australian Dollar"})
	NZD = reg(Currency{"NZD", 2, "$", "554", "New Zealand Dollar"})
	HKD = reg(Currency{"HKD", 2, "$", "344", "Hong Kong Dollar"})
	SGD = reg(Currency{"SGD", 2, "$", "702", "Singapore Dollar"})
	INR = reg(Currency{"INR", 2, "₹", "356", "Indian Rupee"})
	KRW = reg(Currency{"KRW", 0, "₩", "410", "Won"})
	BRL = reg(Currency{"BRL", 2, "R$", "986", "Brazilian Real"})
	MXN = reg(Currency{"MXN", 2, "$", "484", "Mexican Peso"})
	ZAR = reg(Currency{"ZAR", 2, "R", "710", "Rand"})
	SEK = reg(Currency{"SEK", 2, "kr", "752", "Swedish Krona"})
	NOK = reg(Currency{"NOK", 2, "kr", "578", "Norwegian Krone"})
	DKK = reg(Currency{"DKK", 2, "kr", "208", "Danish Krone"})
	PLN = reg(Currency{"PLN", 2, "zł", "985", "Zloty"})
	AED = reg(Currency{"AED", 2, "د.إ", "784", "UAE Dirham"})
	SAR = reg(Currency{"SAR", 2, "﷼", "682", "Saudi Riyal"})
	TRY = reg(Currency{"TRY", 2, "₺", "949", "Turkish Lira"})
	RUB = reg(Currency{"RUB", 2, "₽", "643", "Russian Ruble"})
)

// Crypto / stablecoins — decimals are the token's on-chain decimals so an off-chain
// Amount and an on-chain balance share one integer scale.
var (
	USDC = reg(Currency{"USDC", 6, "$", "", "USD Coin"})
	USDT = reg(Currency{"USDT", 6, "$", "", "Tether"})
	BTC  = reg(Currency{"BTC", 8, "₿", "", "Bitcoin"})
	ETH  = reg(Currency{"ETH", 18, "Ξ", "", "Ether"})
	HUSD = reg(Currency{"HUSD", 18, "$", "", "Hanzo USD"}) // Hanzo USD stablecoin, ERC-20
)

// Register adds (or overrides) a currency in the registry and returns it, so custom
// currencies resolve by code the same as the built-ins.
func Register(c Currency) Currency { return reg(c) }

// ByCode looks a currency up by alpha code (case-insensitive).
func ByCode(code string) (Currency, error) {
	if c, ok := registry[strings.ToUpper(strings.TrimSpace(code))]; ok {
		return c, nil
	}
	return Currency{}, ErrUnknownCurrency
}

// ByNumeric looks a currency up by ISO-4217 numeric code ("840").
func ByNumeric(num string) (Currency, error) {
	num = strings.TrimSpace(num)
	for _, c := range registry {
		if c.Numeric == num && num != "" {
			return c, nil
		}
	}
	return Currency{}, ErrUnknownCurrency
}
