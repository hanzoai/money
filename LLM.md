# money

The one exact money type for Go — an `Amount` in a `Currency`, over
`hanzoai/decimal` (`big.Int` fixed-point). Fowler's Money pattern
(`Add`/`Sub`/`Split`/`Allocate`, same-currency safety, never floats) **without**
the ~19-digit ceiling that int64/uint64 money libraries hit, so a fiat cent and
an 18-decimal on-chain balance are both exact. Single package
`github.com/hanzoai/money`; API in `README.md`.

`hanzoai/decimal` is the only dependency. The big.Int core was written from
scratch — not a fork of `Rhymond/go-money`, `bojanz/currency`, or any other.

## Licensing

`MIT OR Apache-2.0`, at your option — per HIP-0137 (`hanzoai/hips`, `HIPs/hip-0137-one-license.md`). Relicensed from BSD-3-Clause,
which HIP-0137 puts out of scope for `hanzoai`. The only dependency is
`hanzoai/decimal`, relicensed identically, so there is no upstream licence to
inherit; the per-file `// Copyright (c) 2026, Hanzo AI, Inc.` headers moved with
it.
