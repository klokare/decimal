# Changelog

## v2.0.0

The port is complete: formatting and parsing work, the API is idiomatic Go, and the whole thing is
checked against golden data generated from .NET 9.

v2 is a new module path. v1 stays reachable at its tag.

```
go get github.com/klokare/decimal/v2
```

### Bugs fixed

Fifteen defects, most of which produced silently wrong answers rather than failing.

**Arithmetic**

- `divByConst` had no 32-bit implementation and panicked, so **division crashed on every 32-bit
  platform** — 386, arm, mips, ppc, s390. All five 32/64-bit branches were performance
  specialisations carried over from C#; the 64-bit form is correct everywhere and is now used
  unconditionally.
- `div128By96` translated C#'s `if (remainder++ < den) break` as `remainder++; if remainder < den`.
  Post-increment became pre-increment, so the "add the divisor back" loop **never terminated**.
  `79228162514264337593543950334 / 79228162514264337593543950335` hung forever; the path is
  reachable from any division with a 96-bit divisor.
- The 64-bit-divisor loop in `varDecDiv` was missing a `break`, which is the same hang by another
  route.
- `decAddSub` masked the result sign with `scaleMask` where the reference uses `SignMask`, so
  `x + (-0)` and `x - (-0)` came back **with the sign flipped**.
- `Unscale` guarded three of its four divisions with `>` where the reference uses `>=`, so the last
  power of ten could never be divided out and **every division result carried a spurious trailing
  zero**: `1 / 1.00` returned `1.0`.

**Formatting**

- `roundNumber` passed a variable that was declared and never assigned where the reference passes
  the digit at the rounding position, so **nothing was ever rounded up**. `Format(12.345678, "G4")`
  returned `12.34` instead of `12.35`, and every `G` and `E` result was affected.
- The custom-format scanner had no case for backslash, so the escaped character in `\#0` was counted
  as a digit placeholder.
- Its quoted-literal scan stopped on the closing quote instead of consuming it, so the quote was
  re-read as the start of a second quoted run and swallowed what followed: `'lit'0` lost its
  placeholder.
- Scanner and emitter both compared the character after `E` against NUL rather than against `'0'`,
  so scientific notation in a custom format was never recognised and `0E0` rendered as a literal.
- Prepending a negative sign read `sb.Bytes()` while writing back into the same buffer, so the sign
  overwrote the bytes still being read: `-0.5` with format `0.0` produced `----`.

**Parsing**

- `break` inside a `switch` breaks the switch, not the loop, so unrecognised characters were skipped
  rather than rejected. **`Parse("")`, `Parse("ysaidufljasdf")` and `Parse("12abc")` all returned 0
  with a nil error.**
- The all-zeros path assigned the sign instead of OR-ing it, dropping the scale, so `-0.00` lost its
  trailing zeros.

**Elsewhere**

- `Median` indexed past the end: it panicked for two values and averaged the wrong pair for every
  even count.
- `Round` rejected `Truncate`, `Floor` and `Ceiling` despite implementing them.
- `Random` produced an illegal scale above 28 about 88% of the time.

### New

- **Formatting**: `C`, `F`, `N`, `P` and `R` specifiers, which previously panicked. `FormatWith`
  and `MustFormat`. `Format` returns errors rather than panicking.
- **`NumberFormat`**: injectable symbols and layout patterns, with `Invariant` and `EnUS`
  transcribed from the runtime and checked against it in CI.
- **`Styles` and `ParseStyle`**: the `NumberStyles` equivalent — currency symbols, parenthesised
  negatives, trailing signs, exponents.
- **Inspection**: `Bits`, `Scale`, `Coefficient`, `IsInteger`, `Normalize`.
- **Conversions**: the full `Int8`…`Int64`/`Int` and `Uint8`…`Uint64`/`Uint` set, each with an `E`
  twin. `ToOACurrency` and `FromOACurrency`.
- **Errors**: `ErrOverflow`, `ErrDivideByZero`, `ErrScaleRange`, `ErrSyntax`, `ErrFormat`, raised by
  both the panicking and the error-returning forms.
- **Encoding**: `AppendText`, `AppendBinary`, `DotNetBytes`, `FromDotNetBytes`, `JSONNumber`.
  `UnmarshalBinary` now validates the flags word.
- **Zero dependencies.** `testify` is gone.
- Builds and is tested on every Go architecture, including 32-bit and big-endian.

### Breaking changes

The module path changes to `github.com/klokare/decimal/v2`. Then:

| v1 | v2 |
|---|---|
| `NewFromString(s)` | `MustParse(s)`, or `Parse(s)` for the error |
| `NewFromInt32`, `NewFromInt64` | `FromInt(v)` — generic over every signed width |
| `NewFromUint32`, `NewFromUint64` | `FromUint(v)` |
| `NewFromFloat32`, `NewFromFloat64` | `FromFloat32`, `FromFloat64` — now return an error |
| `d.Rem(x)` | `d.Mod(x)` |
| `d.ToInt32()`, `d.ToInt64()`, … | `d.Int32()`, `d.Int64()`, … |
| `d.ToFloat32()`, `d.ToFloat64()` | `d.Float32()`, `d.Float64()` |
| `d.Round(n int32, mode)` | `d.Round(n int, mode)` — and all five modes now work |
| `Format(d, f)` returned `(string, nil)` and panicked | returns a real error |
| `NullDecimal` embedded `Decimal` | named field: `n.Decimal`, `n.Valid` |

Two behaviour changes to check for:

- **JSON is now a quoted string**, not a bare number. Unmarshalling accepts both, so readers are
  unaffected; writers that must emit a number should use `JSONNumber`. See the README for why.
- **`Parse` rejects malformed input** that v1 silently accepted as `0`. Code that relied on
  `Parse("")` returning zero without an error needs to handle the error.

Everything else keeps its name: `Add`, `Sub`, `Mul`, `Div`, `Neg`, `Abs`, `Cmp`, `Equal`,
`LessThan`, `GreaterThan`, `Ceil`, `Floor`, `Truncate`, `Clamp`, `Sign`, `IsZero`, `IsNegative`,
`Max`, `Min`, `MaxAny`, `MinAny`, `Sum`, `Mean`, `Median`, `Random`, `NullDecimal`, and the
`Zero`/`One`/`Two`/`MinusOne`/`MaxValue`/`MinValue`/`SmallestNonZero` constants.

---

## v1.0.0

The final release of the v1 API, published unchanged from several years of private production use.

The arithmetic core is a faithful port of .NET's `DecCalc` and is what saw that use. Formatting,
parsing and the conversion set are incomplete, and the defects listed above are all present. It is
frozen; new code should use v2.
