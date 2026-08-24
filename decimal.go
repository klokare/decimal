// Package decimal implements a 96-bit base-10 fixed-point number, ported from
// .NET's System.Decimal.
//
// A Decimal holds a 96-bit integer scaled by a power of ten between 0 and 28,
// giving 28 significant digits over the range
//
//	-79228162514264337593543950335 .. 79228162514264337593543950335
//
// It is a 16-byte value type. Copying one costs nothing and arithmetic never
// allocates, which is what distinguishes this package from arbitrary-precision
// decimals. Values such as 0.1 are represented exactly.
//
// # Scale is part of the value
//
// Like .NET, this package preserves trailing zeros: 1.10 + 2.20 is 3.30, not
// 3.3, and the difference is visible in String and in [Decimal.Bits]. Two
// Decimals can therefore be numerically equal while holding different bits.
//
// # Comparing
//
// Because Go's == compares all 16 bytes, it distinguishes 1.0 from 1.00. Use
// [Decimal.Equal] or [Decimal.Cmp] for value comparison. For the same reason a
// Decimal used directly as a map key keys on the representation; call
// [Decimal.Normalize] first if that is not what you want.
//
// # Errors
//
// The arithmetic and conversion methods panic on overflow and division by zero,
// so expressions can be chained. Each has a Try twin that returns an error
// instead. Both wrap the same sentinels, so [errors.Is] works either way.
package decimal

import (
	"math/bits"
	"math/rand"
	"sort"
)

// Decimal is a 96-bit base-10 fixed-point number. Its zero value is 0.
//
// The field order matches the .NET layout and must not be changed; the
// arithmetic core reinterprets these words directly.
type Decimal struct {
	flags uint32
	high  uint32
	low   uint32
	mid   uint32
}

// RoundingMode selects how [Decimal.Round] resolves a value that is not exactly
// representable at the requested number of places.
type RoundingMode byte

// Rounding modes. ToEven and AwayFromZero correspond to .NET's
// MidpointRounding; the other three have no MidpointRounding equivalent and are
// an extension.
const (
	// ToEven rounds a midpoint to the nearest even digit (banker's rounding).
	// This is the default in .NET and in IEEE 754.
	ToEven RoundingMode = iota
	// AwayFromZero rounds a midpoint away from zero.
	AwayFromZero
	// Truncate discards the extra digits.
	Truncate
	// Floor rounds towards negative infinity.
	Floor
	// Ceiling rounds towards positive infinity.
	Ceiling

	maxRoundingMode = Ceiling
)

// Frequently used values. These are copies; assigning to one has no effect on
// any other.
var (
	// Zero is 0.
	Zero = Decimal{}
	// One is 1.
	One = Decimal{low: 1}
	// Two is 2.
	Two = Decimal{low: 2}
	// MinusOne is -1.
	MinusOne = Decimal{low: 1, flags: signMask}
	// MaxValue is 79228162514264337593543950335.
	MaxValue = Decimal{low: 0xFFFFFFFF, mid: 0xFFFFFFFF, high: 0xFFFFFFFF}
	// MinValue is -79228162514264337593543950335.
	MinValue = Decimal{low: 0xFFFFFFFF, mid: 0xFFFFFFFF, high: 0xFFFFFFFF, flags: signMask}
	// SmallestNonZero is 0.0000000000000000000000000001, the smallest positive
	// value a Decimal can represent.
	SmallestNonZero = Decimal{low: 1, flags: decimalPrecision << scaleShift}
)

// low64 returns the low and mid words as a single 64-bit value.
func (d Decimal) low64() uint64 { return uint64(d.mid)<<32 | uint64(d.low) }

// setLow64 splits a 64-bit value across the low and mid words.
func (d *Decimal) setLow64(value uint64) {
	d.mid = uint32(value >> 32)
	d.low = uint32(value)
}

// scale returns the power of ten the 96-bit integer is divided by.
func (d Decimal) scale() int { return int(byte(d.flags >> scaleShift)) }

// -- construction ------------------------------------------------------------

// New assembles a Decimal from its parts: a 96-bit integer given as three
// 32-bit words least-significant first, a sign, and a scale between 0 and 28.
// The value is (low + mid*2^32 + high*2^64) / 10^scale, negated when neg.
//
// It reports [ErrScaleRange] if scale exceeds 28.
func New(low, mid, high uint32, neg bool, scale uint8) (Decimal, error) {
	if scale > decimalPrecision {
		return Decimal{}, wrapf(ErrScaleRange, "scale %d exceeds %d", scale, decimalPrecision)
	}
	flags := uint32(scale) << scaleShift
	if neg {
		flags |= signMask
	}
	return Decimal{low: low, mid: mid, high: high, flags: flags}, nil
}

// FromBits rebuilds a Decimal from the representation returned by
// [Decimal.Bits]: low, mid, high and the flags word. It reports [ErrScaleRange]
// if the flags word has a scale above 28 or sets any reserved bit.
func FromBits(b [4]uint32) (Decimal, error) {
	flags := b[3]
	if flags & ^(signMask|scaleMask) != 0 {
		return Decimal{}, wrapf(ErrScaleRange, "flags %#08x sets reserved bits", flags)
	}
	if scale := byte(flags >> scaleShift); scale > decimalPrecision {
		return Decimal{}, wrapf(ErrScaleRange, "scale %d exceeds %d", scale, decimalPrecision)
	}
	return Decimal{low: b[0], mid: b[1], high: b[2], flags: flags}, nil
}

// Signed and Unsigned constrain the generic integer constructors.
type (
	// Signed is any signed integer type.
	Signed interface {
		~int | ~int8 | ~int16 | ~int32 | ~int64
	}
	// Unsigned is any unsigned integer type.
	Unsigned interface {
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
	}
)

// FromInt converts any signed integer to a Decimal. Every value of every width
// is representable, so this cannot fail.
func FromInt[T Signed](value T) Decimal {
	v := int64(value)
	var d Decimal
	if v < 0 {
		d.flags = signMask
		// Negating math.MinInt64 overflows int64, so negate in unsigned space.
		d.setLow64(-uint64(v))
		return d
	}
	d.setLow64(uint64(v))
	return d
}

// FromUint converts any unsigned integer to a Decimal. Every value of every
// width is representable, so this cannot fail.
func FromUint[T Unsigned](value T) Decimal {
	var d Decimal
	d.setLow64(uint64(value))
	return d
}

// FromFloat32 converts a float32, keeping the 7 significant digits a float32
// carries. It reports [ErrOverflow] for NaN, an infinity, or a value outside the
// Decimal range.
//
// Converting binary floating point to decimal is lossy in general: float32(0.1)
// is not exactly 0.1. Parse the text instead when the digits matter.
func FromFloat32(value float32) (d Decimal, err error) {
	defer func() { err = recovered(recover(), &d) }()
	varDecFromR4(value, &d)
	return d, nil
}

// FromFloat64 converts a float64, keeping the 15 significant digits a float64
// carries. It reports [ErrOverflow] for NaN, an infinity, or a value outside the
// Decimal range.
func FromFloat64(value float64) (d Decimal, err error) {
	defer func() { err = recovered(recover(), &d) }()
	varDecFromR8(value, &d)
	return d, nil
}

// FromOACurrency converts an OLE Automation currency value, which is an amount
// scaled by 10000.
func FromOACurrency(cy int64) Decimal {
	d := FromInt(cy)
	// Dividing by 10000 exactly is a scale adjustment, not a division.
	d.flags |= 4 << scaleShift
	return d
}

// -- inspection --------------------------------------------------------------

// Bits returns the representation as low, mid, high and flags, matching .NET's
// decimal.GetBits. Bits 16 to 23 of the flags word hold the scale and bit 31
// holds the sign; the rest are zero.
func (d Decimal) Bits() [4]uint32 { return [4]uint32{d.low, d.mid, d.high, d.flags} }

// Scale returns the power of ten the 96-bit integer is divided by, 0 to 28.
// It reflects trailing zeros: MustParse("1.100").Scale() is 3.
func (d Decimal) Scale() uint8 { return uint8(d.flags >> scaleShift) }

// Coefficient returns the unscaled 96-bit integer as three words,
// least-significant first, without its sign.
func (d Decimal) Coefficient() [3]uint32 { return [3]uint32{d.low, d.mid, d.high} }

// Sign returns -1 if d is negative, 0 if it is zero, and +1 if it is positive.
// Negative zero reports 0.
func (d Decimal) Sign() int {
	if d.low|d.mid|d.high == 0 {
		return 0
	}
	if d.flags&signMask != 0 {
		return -1
	}
	return 1
}

// IsZero reports whether d is zero, including negative zero.
func (d Decimal) IsZero() bool { return d.low|d.mid|d.high == 0 }

// IsNegative reports whether d carries a sign bit. Negative zero reports true;
// use [Decimal.Sign] to treat it as zero.
func (d Decimal) IsNegative() bool { return d.flags&signMask != 0 }

// IsInteger reports whether d has no fractional part.
func (d Decimal) IsInteger() bool { return d.Truncate().Equal(d) }

// -- arithmetic --------------------------------------------------------------

// Add returns d + value. It panics with [ErrOverflow] if the result does not
// fit; see [Decimal.TryAdd] for the error-returning form.
func (d Decimal) Add(value Decimal) Decimal {
	decAddSub(&d, &value, false)
	return d
}

// Sub returns d - value. It panics with [ErrOverflow] if the result does not fit.
func (d Decimal) Sub(value Decimal) Decimal {
	decAddSub(&d, &value, true)
	return d
}

// Mul returns d * value. It panics with [ErrOverflow] if the result does not fit.
func (d Decimal) Mul(value Decimal) Decimal {
	varDecMul(&d, &value)
	return d
}

// Div returns d / value. It panics with [ErrDivideByZero] if value is zero, and
// with [ErrOverflow] if the result does not fit.
func (d Decimal) Div(value Decimal) Decimal {
	varDecDiv(&d, &value)
	return d
}

// Mod returns the remainder of d / value. The result takes the sign of d, as in
// .NET and in Go's % operator. It panics with [ErrDivideByZero] if value is zero.
func (d Decimal) Mod(value Decimal) Decimal {
	varDecMod(&d, &value)
	return d
}

// Neg returns -d. Negating zero yields negative zero, which compares equal to
// zero.
func (d Decimal) Neg() Decimal {
	d.flags ^= signMask
	return d
}

// Abs returns the absolute value of d.
func (d Decimal) Abs() Decimal {
	d.flags &^= signMask
	return d
}

// TryAdd returns d + value, reporting [ErrOverflow] instead of panicking.
func (d Decimal) TryAdd(value Decimal) (r Decimal, err error) {
	defer func() { err = recovered(recover(), &r) }()
	return d.Add(value), nil
}

// TrySub returns d - value, reporting [ErrOverflow] instead of panicking.
func (d Decimal) TrySub(value Decimal) (r Decimal, err error) {
	defer func() { err = recovered(recover(), &r) }()
	return d.Sub(value), nil
}

// TryMul returns d * value, reporting [ErrOverflow] instead of panicking.
func (d Decimal) TryMul(value Decimal) (r Decimal, err error) {
	defer func() { err = recovered(recover(), &r) }()
	return d.Mul(value), nil
}

// TryDiv returns d / value, reporting [ErrDivideByZero] or [ErrOverflow]
// instead of panicking.
func (d Decimal) TryDiv(value Decimal) (r Decimal, err error) {
	defer func() { err = recovered(recover(), &r) }()
	return d.Div(value), nil
}

// TryMod returns the remainder of d / value, reporting [ErrDivideByZero] or
// [ErrOverflow] instead of panicking.
func (d Decimal) TryMod(value Decimal) (r Decimal, err error) {
	defer func() { err = recovered(recover(), &r) }()
	return d.Mod(value), nil
}

// -- comparison --------------------------------------------------------------

// Cmp compares d and value by numeric value, ignoring differences in scale. It
// returns -1, 0 or +1.
func (d Decimal) Cmp(value Decimal) int { return int(varDecCmp(d, value)) }

// Equal reports whether d and value have the same numeric value. It ignores
// scale, so 1.0 and 1.00 are equal, and it treats negative zero as zero. Go's ==
// does neither.
func (d Decimal) Equal(value Decimal) bool { return varDecCmp(d, value) == 0 }

// GreaterThan reports whether d > value.
func (d Decimal) GreaterThan(value Decimal) bool { return varDecCmp(d, value) > 0 }

// GreaterThanOrEqual reports whether d >= value.
func (d Decimal) GreaterThanOrEqual(value Decimal) bool { return varDecCmp(d, value) >= 0 }

// LessThan reports whether d < value.
func (d Decimal) LessThan(value Decimal) bool { return varDecCmp(d, value) < 0 }

// LessThanOrEqual reports whether d <= value.
func (d Decimal) LessThanOrEqual(value Decimal) bool { return varDecCmp(d, value) <= 0 }

// Clamp returns d limited to the inclusive range min to max.
func (d Decimal) Clamp(min, max Decimal) Decimal {
	if d.LessThan(min) {
		return min
	}
	if d.GreaterThan(max) {
		return max
	}
	return d
}

// -- rounding ----------------------------------------------------------------

// Round returns d rounded to places decimal places using mode. places must be 0
// to 28; it panics with [ErrScaleRange] otherwise.
//
// Unlike .NET, which accepts only ToEven and AwayFromZero here, every
// [RoundingMode] is supported.
func (d Decimal) Round(places int, mode RoundingMode) Decimal {
	if places < 0 || places > decimalPrecision {
		panic(wrapf(ErrScaleRange, "places %d is not in 0..%d", places, decimalPrecision))
	}
	if mode > maxRoundingMode {
		panic(wrapf(ErrScaleRange, "unknown rounding mode %d", mode))
	}
	if scale := int32(d.scale()) - int32(places); scale > 0 {
		internalRound(&d, uint32(scale), mode)
	}
	return d
}

// TryRound returns d rounded to places decimal places, reporting
// [ErrScaleRange] instead of panicking.
func (d Decimal) TryRound(places int, mode RoundingMode) (r Decimal, err error) {
	defer func() { err = recovered(recover(), &r) }()
	return d.Round(places, mode), nil
}

// Ceil returns the smallest integer value not less than d.
func (d Decimal) Ceil() Decimal { return d.roundToInteger(Ceiling) }

// Floor returns the largest integer value not greater than d.
func (d Decimal) Floor() Decimal { return d.roundToInteger(Floor) }

// Truncate returns d with its fractional part discarded, rounding towards zero.
func (d Decimal) Truncate() Decimal { return d.roundToInteger(Truncate) }

func (d Decimal) roundToInteger(mode RoundingMode) Decimal {
	if d.flags&scaleMask != 0 {
		internalRound(&d, uint32(byte(d.flags>>scaleShift)), mode)
	}
	return d
}

// Normalize returns d with trailing fractional zeros removed, so that equal
// values share a representation: MustParse("1.100").Normalize() equals
// MustParse("1").
//
// Use it before comparing with == or before using a Decimal as a map key.
func (d Decimal) Normalize() Decimal {
	if d.flags&scaleMask == 0 {
		return d
	}
	low := d.low
	high64 := uint64(d.high)<<32 | uint64(d.mid)
	scale := int32(d.scale())
	calcUnscale(&low, &high64, &scale)
	d.low = low
	d.mid = uint32(high64)
	d.high = uint32(high64 >> 32)
	d.flags = d.flags&signMask | uint32(scale)<<scaleShift
	return d
}

// -- conversion --------------------------------------------------------------

// Float32 returns d as a float32, rounding to the 7 significant digits a
// float32 carries. It never fails; values beyond float32 range become an
// infinity, as they do in .NET.
func (d Decimal) Float32() float32 { return varR4FromDec(d) }

// Float64 returns d as a float64, rounding to the 15 significant digits a
// float64 carries. It never fails.
func (d Decimal) Float64() float64 { return varR8FromDec(d) }

// ToOACurrency returns d as an OLE Automation currency value, an amount scaled
// by 10000 and rounded to four decimal places.
func (d Decimal) ToOACurrency() int64 {
	return d.Round(4, ToEven).Mul(FromInt(10000)).Int64()
}

// The integer conversions truncate towards zero and panic with [ErrOverflow] if
// the truncated value does not fit the destination. Each has an E twin that
// returns an error.

// Int64 returns d truncated to an int64.
func (d Decimal) Int64() int64 {
	t := d.Truncate()
	if t.high == 0 {
		l := int64(t.low64())
		if !t.IsNegative() {
			if l >= 0 {
				return l
			}
		} else {
			// Relies on wrapping: math.MinInt64 negated is itself, and is <= 0.
			l = -l
			if l <= 0 {
				return l
			}
		}
	}
	panic(wrapf(ErrOverflow, "%s does not fit in an int64", d))
}

// Uint64 returns d truncated to a uint64.
func (d Decimal) Uint64() uint64 {
	t := d.Truncate()
	if t.high == 0 {
		l := t.low64()
		if !t.IsNegative() || l == 0 {
			return l
		}
	}
	panic(wrapf(ErrOverflow, "%s does not fit in a uint64", d))
}

// Int32 returns d truncated to an int32.
func (d Decimal) Int32() int32 {
	t := d.Truncate()
	if t.high|t.mid == 0 {
		i := int32(t.low)
		if !t.IsNegative() {
			if i >= 0 {
				return i
			}
		} else {
			i = -i
			if i <= 0 {
				return i
			}
		}
	}
	panic(wrapf(ErrOverflow, "%s does not fit in an int32", d))
}

// Uint32 returns d truncated to a uint32.
func (d Decimal) Uint32() uint32 {
	t := d.Truncate()
	if t.high|t.mid == 0 {
		i := t.low
		if !t.IsNegative() || i == 0 {
			return i
		}
	}
	panic(wrapf(ErrOverflow, "%s does not fit in a uint32", d))
}

// Int16 returns d truncated to an int16.
func (d Decimal) Int16() int16 { return int16(narrowSigned(d, -1<<15, 1<<15-1, "int16")) }

// Uint16 returns d truncated to a uint16.
func (d Decimal) Uint16() uint16 { return uint16(narrowUnsigned(d, 1<<16-1, "uint16")) }

// Int8 returns d truncated to an int8.
func (d Decimal) Int8() int8 { return int8(narrowSigned(d, -1<<7, 1<<7-1, "int8")) }

// Uint8 returns d truncated to a uint8.
func (d Decimal) Uint8() uint8 { return uint8(narrowUnsigned(d, 1<<8-1, "uint8")) }

// Int returns d truncated to an int.
func (d Decimal) Int() int {
	if bits.UintSize == 64 {
		return int(d.Int64())
	}
	return int(d.Int32())
}

// Uint returns d truncated to a uint.
func (d Decimal) Uint() uint {
	if bits.UintSize == 64 {
		return uint(d.Uint64())
	}
	return uint(d.Uint32())
}

// narrowSigned truncates through int64 and range-checks against a narrower type.
func narrowSigned(d Decimal, min, max int64, name string) int64 {
	v := d.Int64()
	if v < min || v > max {
		panic(wrapf(ErrOverflow, "%s does not fit in an %s", d, name))
	}
	return v
}

// narrowUnsigned truncates through uint64 and range-checks against a narrower type.
func narrowUnsigned(d Decimal, max uint64, name string) uint64 {
	v := d.Uint64()
	if v > max {
		panic(wrapf(ErrOverflow, "%s does not fit in a %s", d, name))
	}
	return v
}

// Int64E returns d truncated to an int64, reporting [ErrOverflow] instead of
// panicking.
func (d Decimal) Int64E() (v int64, err error) {
	defer func() { err = recovered(recover(), &v) }()
	return d.Int64(), nil
}

// Uint64E returns d truncated to a uint64, reporting [ErrOverflow] instead of
// panicking.
func (d Decimal) Uint64E() (v uint64, err error) {
	defer func() { err = recovered(recover(), &v) }()
	return d.Uint64(), nil
}

// Int32E returns d truncated to an int32, reporting [ErrOverflow] instead of
// panicking.
func (d Decimal) Int32E() (v int32, err error) {
	defer func() { err = recovered(recover(), &v) }()
	return d.Int32(), nil
}

// Uint32E returns d truncated to a uint32, reporting [ErrOverflow] instead of
// panicking.
func (d Decimal) Uint32E() (v uint32, err error) {
	defer func() { err = recovered(recover(), &v) }()
	return d.Uint32(), nil
}

// Int16E returns d truncated to an int16, reporting [ErrOverflow] instead of
// panicking.
func (d Decimal) Int16E() (v int16, err error) {
	defer func() { err = recovered(recover(), &v) }()
	return d.Int16(), nil
}

// Uint16E returns d truncated to a uint16, reporting [ErrOverflow] instead of
// panicking.
func (d Decimal) Uint16E() (v uint16, err error) {
	defer func() { err = recovered(recover(), &v) }()
	return d.Uint16(), nil
}

// Int8E returns d truncated to an int8, reporting [ErrOverflow] instead of
// panicking.
func (d Decimal) Int8E() (v int8, err error) {
	defer func() { err = recovered(recover(), &v) }()
	return d.Int8(), nil
}

// Uint8E returns d truncated to a uint8, reporting [ErrOverflow] instead of
// panicking.
func (d Decimal) Uint8E() (v uint8, err error) {
	defer func() { err = recovered(recover(), &v) }()
	return d.Uint8(), nil
}

// IntE returns d truncated to an int, reporting [ErrOverflow] instead of
// panicking.
func (d Decimal) IntE() (v int, err error) {
	defer func() { err = recovered(recover(), &v) }()
	return d.Int(), nil
}

// UintE returns d truncated to a uint, reporting [ErrOverflow] instead of
// panicking.
func (d Decimal) UintE() (v uint, err error) {
	defer func() { err = recovered(recover(), &v) }()
	return d.Uint(), nil
}

// -- aggregates --------------------------------------------------------------

// Max returns the greater of a and b.
func Max(a, b Decimal) Decimal {
	if a.LessThan(b) {
		return b
	}
	return a
}

// Min returns the lesser of a and b.
func Min(a, b Decimal) Decimal {
	if a.GreaterThan(b) {
		return b
	}
	return a
}

// MaxAny returns the greatest of values, or [Zero] if there are none.
func MaxAny(values ...Decimal) Decimal {
	if len(values) == 0 {
		return Zero
	}
	max := values[0]
	for _, v := range values[1:] {
		if max.LessThan(v) {
			max = v
		}
	}
	return max
}

// MinAny returns the least of values, or [Zero] if there are none.
func MinAny(values ...Decimal) Decimal {
	if len(values) == 0 {
		return Zero
	}
	min := values[0]
	for _, v := range values[1:] {
		if min.GreaterThan(v) {
			min = v
		}
	}
	return min
}

// Sum returns the total of values, or [Zero] if there are none. It panics with
// [ErrOverflow] if any intermediate total does not fit.
func Sum(values ...Decimal) Decimal {
	var total Decimal
	for _, v := range values {
		total = total.Add(v)
	}
	return total
}

// Mean returns the arithmetic mean of values, or [Zero] if there are none.
func Mean(values ...Decimal) Decimal {
	if len(values) == 0 {
		return Zero
	}
	return Sum(values...).Div(FromInt(len(values)))
}

// Median returns the median of values, or [Zero] if there are none. For an even
// count it averages the two middle values. The input is not modified.
func Median(values ...Decimal) Decimal {
	n := len(values)
	if n == 0 {
		return Zero
	}
	sorted := make([]Decimal, n)
	copy(sorted, values)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].LessThan(sorted[j]) })
	if n%2 == 1 {
		return sorted[n/2]
	}
	// The two middle elements of an even-length slice are at n/2-1 and n/2.
	return sorted[n/2-1].Add(sorted[n/2]).Div(Two)
}

// Random returns a uniformly distributed random Decimal covering the full range
// of representations: every 96-bit coefficient, every legal scale, either sign.
//
// It is meant for tests and fuzzing, not for anything requiring unpredictability.
func Random(rng *rand.Rand) Decimal {
	// The scale byte must land in 0..28; masking the raw word would produce the
	// illegal scales 29..255 for most draws.
	flags := uint32(rng.Intn(decimalPrecision+1)) << scaleShift
	if rng.Intn(2) == 1 {
		flags |= signMask
	}
	return Decimal{
		flags: flags,
		low:   rng.Uint32(),
		mid:   rng.Uint32(),
		high:  rng.Uint32(),
	}
}
