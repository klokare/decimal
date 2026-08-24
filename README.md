# decimal

[![Go Reference](https://pkg.go.dev/badge/github.com/klokare/decimal/v2.svg)](https://pkg.go.dev/github.com/klokare/decimal/v2)
[![CI](https://github.com/klokare/decimal/actions/workflows/ci.yml/badge.svg)](https://github.com/klokare/decimal/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/klokare/decimal/v2)](https://goreportcard.com/report/github.com/klokare/decimal/v2)
[![Coverage Status](https://coveralls.io/repos/github/klokare/decimal/badge.svg?branch=master)](https://coveralls.io/github/klokare/decimal?branch=master)

A Go port of .NET's `System.Decimal`: a 96-bit base-10 fixed-point number with a scale of 0 to 28
digits.

```
-79,228,162,514,264,337,593,543,950,335  ..  79,228,162,514,264,337,593,543,950,335
```

```go
import "github.com/klokare/decimal/v2"

price    := decimal.MustParse("19.99")
quantity := decimal.FromInt(3)
taxRate  := decimal.MustParse("0.0825")

subtotal := price.Mul(quantity)                              // 59.97
tax      := subtotal.Mul(taxRate).Round(2, decimal.AwayFromZero) // 4.95
total    := subtotal.Add(tax)                                // 64.92
```

```
go get github.com/klokare/decimal/v2
```

## Why this one

It is a 16-byte value. Copying costs nothing, arithmetic never allocates, and `0.1` is stored
exactly. That combination is the whole point — it is a different trade from arbitrary precision,
not a worse one.

| | `klokare/decimal` | `shopspring/decimal` | `math/big.Rat` |
|---|---|---|---|
| Representation | 96-bit integer + scale, 16 bytes | `big.Int` + exponent | two `big.Int` |
| Range | ±7.9×10²⁸, 28 significant digits | unbounded | unbounded |
| Allocations per operation | none | several | several |
| Copy semantics | plain value | shares a mantissa pointer | pointer |
| Trailing zeros preserved | yes | yes | no |
| Agrees with .NET / SQL Server `DECIMAL` | bit for bit | approximately | no |
| Dependencies | none | none | stdlib |

Reach for this when you want bounded, allocation-free money arithmetic that agrees exactly with a
.NET service or a SQL Server `DECIMAL` column. Reach for an arbitrary-precision package when you
need unbounded range or more than 28 digits.

```
BenchmarkAdd       2.3 ns/op    0 allocs/op
BenchmarkSub       2.1 ns/op    0 allocs/op
BenchmarkMul      11.6 ns/op    0 allocs/op
BenchmarkDiv      33.3 ns/op    0 allocs/op
BenchmarkCmp       2.6 ns/op    0 allocs/op
BenchmarkRound     7.0 ns/op    0 allocs/op
```

<sub>Apple M3 Pro, Go 1.25.5, darwin/arm64. Run `make bench` for your own figures — the
absolute numbers move with the machine, the zero allocations do not.</sub>

## Two things to know up front

**Scale is part of the value.** Trailing zeros survive, exactly as in .NET:

```go
decimal.MustParse("1.10").Add(decimal.MustParse("2.20"))  // 3.30, not 3.3
decimal.MustParse("0.1").Add(decimal.MustParse("0.2"))    // 0.3, exactly
0.1 + 0.2                                                 // 0.30000000000000004
```

**`==` is not value equality.** Go compares all 16 bytes, so it distinguishes `1.0` from `1.00`:

```go
a, b := decimal.MustParse("1.0"), decimal.MustParse("1.00")

a == b                            // false — different representations
a.Equal(b)                        // true  — same value
a.Normalize() == b.Normalize()    // true
```

The same applies to map keys: call `Normalize` first if you want `1.0` and `1.00` to collide.

## Errors

Arithmetic panics so that expressions chain. Every method has a `Try` twin that returns an error
instead, and both raise the same sentinels, so `errors.Is` works either way.

```go
total := a.Mul(b).Add(c)              // panics on overflow

sum, err := a.TryMul(b)               // (Decimal, error)
n, err := d.Int64E()                  // (int64, error)

errors.Is(err, decimal.ErrOverflow)   // also ErrDivideByZero, ErrScaleRange,
                                      // ErrSyntax, ErrFormat
```

A panic that is not one of this package's errors is never swallowed by the `Try` variants.

## Formatting

All the .NET standard specifiers are supported — `C`, `E`, `F`, `G`, `N`, `P`, `R`, each with an
optional precision — along with custom picture formats. `D` and `X` are integral-only in .NET and
report `ErrFormat`.

```go
d := decimal.MustParse("-1234567.891")

decimal.Format(d, "N2")                    // -1,234,567.89
decimal.Format(d, "E3")                    // -1.235E+006
decimal.Format(d, "#,##0.00;(#,##0.00)")   // (1,234,567.89)
decimal.FormatWith(d, "C2", decimal.EnUS)  // -$1,234,567.89
```

Go has no ambient locale, so symbols and layout come from a `*NumberFormat` you pass in. `Invariant`
is the default and `EnUS` ships too; both are transcribed from the runtime and checked against it in
CI. Build your own with `Clone`:

```go
de := decimal.Invariant.Clone()
de.CurrencySymbol = "€"
de.CurrencyDecimalSeparator = ","
de.CurrencyGroupSeparator = "."
de.CurrencyPositivePattern = 3            // "n $", using .NET's pattern numbering

decimal.FormatWith(d, "C2", de)           // 1.234.567,89 €
```

The pattern fields keep .NET's numbering, so a value read out of a `NumberFormatInfo` can be copied
straight across.

## Parsing

`Parse` accepts what .NET's `Parse` accepts by default: a sign, digits, group separators and a
decimal point. Scientific notation, currency symbols and parenthesised negatives need an explicit
style.

```go
decimal.Parse("1,234.56")                                            // 1234.56
decimal.Parse("1.5e3")                                               // ErrSyntax
decimal.ParseStyle("1.5e3", decimal.StyleFloat, decimal.Invariant)   // 1500
decimal.ParseStyle("($1,234.56)", decimal.StyleAny, decimal.EnUS)    // -1234.56
```

## JSON, SQL and the wire

A `Decimal` marshals to a JSON **string**. A bare JSON number is read back by most consumers —
JavaScript above all — as an IEEE 754 double, which throws away the precision this package exists to
keep. Unmarshalling accepts either form, and `JSONNumber` opts into the bare-number form where a wire
format demands it.

```go
type invoice struct {
    Total decimal.Decimal    `json:"total"`
    Fee   decimal.JSONNumber `json:"fee"`
}
// {"total":"1234.50","fee":2.99}
```

`Decimal` implements `driver.Valuer` and `sql.Scanner`, sending values as text so nothing is lost on
the way to a `NUMERIC` column. Use `NullDecimal` for a nullable one; it handles SQL `NULL` and JSON
`null` in both directions.

`MarshalBinary` gives a stable 16-byte big-endian encoding, which is also what `encoding/gob` uses.
`DotNetBytes` and `FromDotNetBytes` give .NET's own little-endian layout, for exchanging values with
a .NET process.

## Correctness

The suite checks against **79,193 golden records generated from .NET 9**, covering arithmetic,
rounding, comparison, conversion, parsing and every format specifier under two cultures.

Each record carries the four representation words rather than a rendered number, so a result that is
numerically right but carries the wrong scale — or a zero with the wrong sign — fails. Regenerate
them yourself with `make testdata`; the output is deterministic and CI checks it has not drifted.

On top of that: four property-based fuzz targets, the bug-specific cases from Mono's own suite, and
94.7% statement coverage with no uncovered function.

The package builds and is tested on every Go architecture, including 32-bit and big-endian.

```
make test race checkptr cross bench    # and `make testdata` with a .NET SDK
```

## Differences from .NET

Some things do not survive the crossing, and a few are improvements.

| .NET | Here |
|---|---|
| Operators `+ - * / % == < >` | Methods `Add`, `Sub`, `Mul`, `Div`, `Mod`, `Equal`, `Cmp`, `LessThan`… |
| Implicit conversion from every integer type | `FromInt` / `FromUint`, generic over all widths |
| Explicit casts `(int)d` | `d.Int()`, or `d.IntE()` for the error form |
| Literal suffix `1.0034m` | `decimal.MustParse("1.0034")` |
| Exceptions | Panics with sentinel errors, plus `Try` twins |
| `CultureInfo`, ambient thread culture | An explicit `*NumberFormat`; there is no ambient locale in Go |
| `GetHashCode`, which normalises | `Normalize` first — see [equality](#two-things-to-know-up-front) |
| `MidpointRounding`, two modes | Five modes: `ToEven`, `AwayFromZero`, `Truncate`, `Floor`, `Ceiling` |
| `decimal.GetBits`, `new decimal(int[])` | `Bits`, `FromBits`, and `New` from parts |

Per-mille (U+2030) is supported in custom formats; other `NumberFormatInfo` fields this package does
not read, such as `NaNSymbol`, have no meaning for a decimal.

## Versions

**v2** is current. **v1** is frozen at [`v1.0.0`](https://github.com/klokare/decimal/tree/v1.0.0)
and documented there, including its known defects — several of which were serious. See
[CHANGELOG.md](CHANGELOG.md) for what changed and how to migrate.

## Provenance

Ported from the Mono and Microsoft reference implementations:

- [`referencesource/mscorlib/system/decimal.cs`](https://github.com/mono/mono/blob/main/mcs/class/referencesource/mscorlib/system/decimal.cs) — the public type
- [`System.Decimal.DecCalc`](https://github.com/dotnet/runtime/blob/main/src/libraries/System.Private.CoreLib/src/System/Decimal.DecCalc.cs) — the arithmetic core
- `Number.Parsing.cs` and `Number.Formatting.cs` — parsing and formatting

Test cases are drawn from Mono's `DecimalTest.cs`, `DecimalTest2.cs` and `DecimalFormatterTest.cs`.

## License

MIT. See [LICENSE](LICENSE).
