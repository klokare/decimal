# decimal

A Go port of .NET's `System.Decimal` — a 96-bit base-10 fixed-point number with a scale of 0–28
digits, exact representation of decimal fractions, and no heap allocation.

```
-79,228,162,514,264,337,593,543,950,335  ..  79,228,162,514,264,337,593,543,950,335
```

Unlike binary floating point, `0.1` is represented exactly. Unlike arbitrary-precision decimals,
the value is a flat 16-byte struct that costs nothing to copy and never touches the heap.

---

## ⚠️ This is v1, and v1 is frozen

**v1.0.0 is the final release of this API.** It is published as-is, unchanged from the code its
author ran in private production for several years. It is *not* recommended for new projects.

**New code should use v2:**

```
go get github.com/klokare/decimal/v2
```

v2 completes the port (formatting, parsing, the full conversion set), fixes everything listed
below, and adopts an idiomatic Go API. This document exists so that anyone pinned to v1 knows
exactly what they are pinned to.

---

## What v1 does well

The arithmetic core is a faithful, line-by-line port of .NET's `DecCalc` and is the part that saw
real production use. It is validated against ~6,000 golden test cases generated from .NET.

- `Add`, `Sub`, `Mul`, `Div`, `Rem`, `Neg`, `Abs`
- `Cmp`, `Equal`, `LessThan`, `GreaterThan`, and friends
- `Ceil`, `Floor`, `Truncate`
- Conversions to and from `int32`/`int64`/`uint32`/`uint64`/`float32`/`float64`
- **Scale preservation** — `1.10 + 2.20` is `3.30`, not `3.3`, matching .NET exactly
- `NullDecimal` plus `database/sql` `Scanner`/`Valuer` support

## What v1 does badly

These are known and will not be fixed in v1. Every one of them is confirmed by running the code.

### Broken

| Defect | Effect |
|---|---|
| `divByConst` has no 32-bit implementation | **Division panics on every 32-bit platform** (386, arm, mips, ppc, s390) |
| `roundNumber` never rounds up | `Format(12.345678, "G4")` returns `"12.34"`; .NET returns `"12.35"`. All `G`/`E` output is affected |
| `Parse` ignores unrecognised characters | `Parse("")`, `Parse("ysaidufljasdf")` and `Parse("12abc")` all return `0, nil` instead of an error |
| `Median` indexes past the end | `Median(a, b)` panics; every even-length input averages the wrong pair |
| `Round` rejects three of its five modes | `Round(d, n, Floor)`, `Truncate` and `Ceiling` panic with `"invalid rounding mode"`, despite being implemented |
| `Random` produces invalid decimals | ~88% of generated values carry a scale byte above the legal maximum of 28 |
| Negative-zero parse loses its scale | `flags = signMask` overwrites the scale instead of OR-ing into it |

### Missing

- **Format specifiers `C`, `F`, `N`, `P`, `R` are not implemented** and panic. Only `G` and `E`
  work. `Format(d, "F2")` panics with `"bad format specifier: F"`.
- No locale support of any kind. The decimal point, group separator and negative sign are
  hardcoded to `.`, `,` and `-`. There is no currency or percent symbol.
- No `NumberStyles`-equivalent parsing: no currency symbols, parenthesised negatives, trailing
  signs, or group-size validation.
- No `byte`/`int8`/`int16`/`uint16`/`int`/`uint` conversions.
- No way to read a value's representation — no `Scale()`, no `Bits()`.
- `Format` declares an `error` return but always returns `nil`; failures arrive as panics.

### Behavioural notes

- **Every error is a `panic`** with an untyped string. Overflow, divide-by-zero, bad format
  specifiers and out-of-range conversions all panic. There are no error-returning variants.
- **`==` does not mean equality.** Go compares the 16 raw bytes, so `1.0` and `1.00` are *not*
  `==` even though `Equal` reports them equal. This also makes `Decimal` unsafe as a map key.
  .NET's `GetHashCode` normalises; there is no equivalent here.
- `MarshalJSON` emits a **bare, unquoted number**. `NullDecimal` inherits it and therefore never
  produces or accepts JSON `null`.
- Only one of the four test files in the repository is compiled; the rest carry a `.nogo`
  extension. Parsing, formatting, marshalling, `NullDecimal` and the statistical helpers have no
  test coverage at all.

---

## Installing v1

```
go get github.com/klokare/decimal@v1.0.0
```

```go
import "github.com/klokare/decimal"

a := decimal.NewFromString("1.10")   // panics on bad input
b := decimal.NewFromString("2.20")
fmt.Println(a.Add(b))                // 3.30
```

---

## Provenance

Ported from the Mono/Microsoft reference implementation:

- [`mcs/class/referencesource/mscorlib/system/decimal.cs`](https://github.com/mono/mono/blob/main/mcs/class/referencesource/mscorlib/system/decimal.cs) — the public `Decimal` type
- `System.Decimal.DecCalc` from [dotnet/runtime](https://github.com/dotnet/runtime) — the arithmetic core (`calc.go`)
- `Number.Parsing.cs` and `Number.Formatting.cs` — parsing and formatting (`format.go`)

## License

MIT. See [LICENSE](LICENSE).
