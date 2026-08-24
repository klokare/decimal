# decimal

A Go port of .NET's `System.Decimal` — a 96-bit base-10 fixed-point number with a scale of 0–28
digits, exact representation of decimal fractions, and no heap allocation.

```
-79,228,162,514,264,337,593,543,950,335  ..  79,228,162,514,264,337,593,543,950,335
```

> **v2 is under active development.** The API is changing. v1 is frozen at
> [`v1.0.0`](https://github.com/klokare/decimal/tree/v1.0.0) and documented there, including its
> known defects. Do not depend on `master` until v2.0.0 is tagged.

## Why this and not another decimal package

| | `klokare/decimal` | `shopspring/decimal` | `math/big.Rat` |
|---|---|---|---|
| Representation | 96-bit integer + scale, 16 bytes | `big.Int` + exponent | two `big.Int` |
| Range | ±7.9×10²⁸, 28 significant digits | unbounded | unbounded |
| Allocation | none | per operation | per operation |
| Value semantics | yes, copyable struct | pointer-ish, shared mantissa | pointer |
| Scale preserved | yes — `1.10 + 2.20` is `3.30` | yes | no |
| .NET / SQL Server `DECIMAL` parity | exact | approximate | no |

Pick this one when you want bounded, allocation-free money arithmetic that agrees bit-for-bit with
.NET and SQL Server. Pick an arbitrary-precision package when you need unbounded range.

## Status

- **Arithmetic core** — complete and validated against golden data generated from .NET.
  Now runs on every Go architecture, including 32-bit and big-endian.
- **Parsing / formatting** — in progress.
- **API** — being reshaped for v2.

## Provenance

Ported from the Mono/Microsoft reference implementation:

- [`referencesource/mscorlib/system/decimal.cs`](https://github.com/mono/mono/blob/main/mcs/class/referencesource/mscorlib/system/decimal.cs) — the public `Decimal` type
- `System.Decimal.DecCalc` from [dotnet/runtime](https://github.com/dotnet/runtime) — the arithmetic core (`calc.go`)
- `Number.Parsing.cs` and `Number.Formatting.cs` — parsing and formatting

## License

MIT. See [LICENSE](LICENSE).
