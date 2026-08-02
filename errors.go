// Copyright (c) 2026, Hanzo AI, Inc. BSD-3-Clause.

package money

import "errors"

// Sentinel errors, matched with errors.Is.
var (
	// ErrCurrencyMismatch is returned when an operation combines two different
	// (non-zero) currencies.
	ErrCurrencyMismatch = errors.New("money: currencies do not match")
	// ErrUnknownCurrency is returned when a currency code/number is not in the registry.
	ErrUnknownCurrency = errors.New("money: unknown currency")
	// ErrInvalidAmount is returned when an amount string cannot be parsed.
	ErrInvalidAmount = errors.New("money: invalid amount")
	// ErrNoRate is returned when an exchange rate is missing.
	ErrNoRate = errors.New("money: no exchange rate")
	// ErrNonPositive is returned when a positive value was required (split parts, ratios).
	ErrNonPositive = errors.New("money: value must be positive")
	// ErrOverflow is returned when an exact value does not fit the requested
	// machine type. It is never a rounded or truncated answer: a money conversion
	// that silently wraps reports a balance nobody holds.
	ErrOverflow = errors.New("money: value does not fit an int64")
)
