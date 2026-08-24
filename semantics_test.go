package decimal

import (
	"errors"
	"math"
	"testing"

	"github.com/klokare/decimal/v2/internal/assert"
)

// Scale is part of a Decimal's value. These pin the behaviour that makes this
// package different from an arbitrary-precision one, and that the v1 suite could
// not see because it compared with Equal.
func TestScalePreservation(t *testing.T) {
	cases := []struct {
		expr string
		got  Decimal
		want string
	}{
		{"1.10 + 2.20", MustParse("1.10").Add(MustParse("2.20")), "3.30"},
		{"1.10 - 0.10", MustParse("1.10").Sub(MustParse("0.10")), "1.00"},
		{"1.1 * 2.2", MustParse("1.1").Mul(MustParse("2.2")), "2.42"},
		{"1.10 * 2.20", MustParse("1.10").Mul(MustParse("2.20")), "2.4200"},
		{"1 / 1.00", One.Div(MustParse("1.00")), "1"},
		{"2.50 + 2.50", MustParse("2.50").Add(MustParse("2.50")), "5.00"},
		{"0.1 + 0.2", MustParse("0.1").Add(MustParse("0.2")), "0.3"},
		{"parse 1.100", MustParse("1.100"), "1.100"},
		{"parse 100.00", MustParse("100.00"), "100.00"},
	}
	for _, c := range cases {
		if got := c.got.String(); got != c.want {
			t.Errorf("%s = %q, want %q (bits %s)", c.expr, got, c.want, formatBits(c.got))
		}
	}

	// 0.1 + 0.2 is exactly 0.3 here, which is the entire point of the type.
	assert.True(t, MustParse("0.1").Add(MustParse("0.2")).Equal(MustParse("0.3")),
		"0.1 + 0.2 should be exactly 0.3")

	// For contrast. These have to be variables: Go evaluates untyped float
	// constants exactly, so the constant expression 0.1+0.2 == 0.3 is true.
	f1, f2, f3 := 0.1, 0.2, 0.3
	assert.False(t, f1+f2 == f3, "float64 0.1+0.2 is not 0.3")
}

// Two Decimals can be numerically equal while holding different bits. Go's ==
// sees the bits; Equal and Cmp see the value. This is the sharpest edge in the
// package, so it is pinned rather than left to documentation.
func TestStructEqualityIsNotValueEquality(t *testing.T) {
	a, b := MustParse("1.0"), MustParse("1.00")

	assert.False(t, a == b, "== compares all 16 bytes, so 1.0 and 1.00 differ")
	assert.True(t, a.Equal(b), "Equal compares by value")
	assert.Equal(t, 0, a.Cmp(b), "Cmp compares by value")

	// Normalize is the bridge: it makes equal values share a representation.
	assert.True(t, a.Normalize() == b.Normalize(), "Normalize should make them identical")

	// Which is what makes a Decimal usable as a map key.
	m := map[Decimal]string{}
	m[a] = "first"
	m[b] = "second"
	assert.Equal(t, 2, len(m), "un-normalized values are distinct map keys")

	m2 := map[Decimal]string{}
	m2[a.Normalize()] = "first"
	m2[b.Normalize()] = "second"
	assert.Equal(t, 1, len(m2), "normalized values collapse to one key")
}

func TestNormalize(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"1.100", "1.1"},
		{"1.000", "1"},
		{"100.00", "100"},
		{"0.000", "0"},
		{"-0.000", "0"},
		{"1.101", "1.101"},
		{"0.0000000000000000000000000001", "0.0000000000000000000000000001"},
		{"79228162514264337593543950335", "79228162514264337593543950335"},
	} {
		got := MustParse(c.in).Normalize()
		assert.Equal(t, c.want, got.String(), "Normalize(%s)", c.in)
		assert.True(t, got.Equal(MustParse(c.in)), "Normalize(%s) changed the value", c.in)
	}

	// -0.000 normalizes to a zero that still carries its sign bit but is still
	// numerically zero.
	n := MustParse("-0.000").Normalize()
	assert.Equal(t, uint8(0), n.Scale(), "scale after normalizing -0.000")
	assert.Equal(t, 0, n.Sign(), "Sign of a normalized negative zero")
}

// Negative zero is representable and must behave as zero everywhere that
// matters, while still round-tripping through the representation.
func TestNegativeZero(t *testing.T) {
	negZero := Zero.Neg()

	assert.True(t, negZero.IsNegative(), "the sign bit is set")
	assert.True(t, negZero.IsZero(), "IsZero is true")
	assert.Equal(t, 0, negZero.Sign(), "Sign reports 0, not -1")
	assert.True(t, negZero.Equal(Zero), "equal to positive zero")
	assert.Equal(t, 0, negZero.Cmp(Zero), "compares equal to positive zero")
	assert.False(t, negZero == Zero, "but the bits differ")

	assert.Equal(t, "0", negZero.String(), "formats without a sign, as .NET does")
	assert.Equal(t, "0.00", MustFormat(negZero, "F2"), "and in fixed-point")

	// Arithmetic with negative zero keeps the other operand intact. This is the
	// decAddSub sign bug the golden tables caught.
	for _, s := range []string{"0.1", "-0.1", "1", "79228162514264337593543950335"} {
		d := MustParse(s)
		assert.Equal(t, d, d.Add(negZero), "%s + (-0)", s)
		assert.Equal(t, d, negZero.Add(d), "(-0) + %s", s)
		assert.Equal(t, d, d.Sub(negZero), "%s - (-0)", s)
	}

	// Abs clears the sign; Neg toggles it back.
	assert.Equal(t, Zero, negZero.Abs(), "Abs of negative zero")
	assert.Equal(t, negZero, Zero.Neg(), "Neg of positive zero")
}

func TestOverflowBoundaries(t *testing.T) {
	overflows := func(name string, fn func() Decimal) {
		t.Helper()
		_, panicked := wantPanic(func() { fn() })
		assert.True(t, panicked, "%s should overflow", name)
	}

	overflows("MaxValue + 1", func() Decimal { return MaxValue.Add(One) })
	overflows("MinValue - 1", func() Decimal { return MinValue.Sub(One) })
	overflows("MaxValue * 2", func() Decimal { return MaxValue.Mul(Two) })
	overflows("MaxValue / 0.5", func() Decimal { return MaxValue.Div(MustParse("0.5")) })
	overflows("MaxValue.Neg().Sub(1)", func() Decimal { return MaxValue.Neg().Sub(One) })

	// And the ones that must not overflow. Adding SmallestNonZero to MaxValue
	// needs 29 significant digits, so the addend rounds away entirely rather
	// than overflowing -- which is what .NET does.
	assert.Equal(t, MaxValue, MaxValue.Add(SmallestNonZero), "MaxValue + SmallestNonZero")
	assert.Equal(t, MaxValue, MaxValue.Add(Zero), "MaxValue + 0")
	assert.Equal(t, MinValue, MaxValue.Neg(), "-MaxValue is MinValue")
	assert.Equal(t, MaxValue, MinValue.Abs(), "|MinValue| is MaxValue")
	assert.True(t, MaxValue.Sub(MaxValue).IsZero(), "MaxValue - MaxValue")

	// The Try forms report instead of panicking, and leave a zero result.
	got, err := MaxValue.TryAdd(One)
	assert.True(t, errors.Is(err, ErrOverflow), "TryAdd should report ErrOverflow, got %v", err)
	assert.Equal(t, Decimal{}, got, "the result is zeroed on failure")

	_, err = One.TryDiv(Zero)
	assert.True(t, errors.Is(err, ErrDivideByZero), "TryDiv by zero, got %v", err)
}

func TestDivideByZero(t *testing.T) {
	for _, num := range []Decimal{Zero, One, MinusOne, MaxValue, MinValue} {
		_, panicked := wantPanic(func() { num.Div(Zero) })
		assert.True(t, panicked, "%s / 0 should panic", num)
		_, panicked = wantPanic(func() { num.Mod(Zero) })
		assert.True(t, panicked, "%s %% 0 should panic", num)
	}
}

// The integer conversions rely on wrapping arithmetic at the extremes; these are
// the values where a sign test done the obvious way gets it wrong.
func TestConversionExtremes(t *testing.T) {
	assert.Equal(t, int32(math.MinInt32), FromInt(math.MinInt32).Int32(), "int32 minimum round-trips")
	assert.Equal(t, int64(math.MinInt64), FromInt(math.MinInt64).Int64(), "int64 minimum round-trips")
	assert.Equal(t, int32(math.MaxInt32), FromInt(math.MaxInt32).Int32(), "int32 maximum")
	assert.Equal(t, uint64(math.MaxUint64), FromUint(uint64(math.MaxUint64)).Uint64(), "uint64 maximum")

	// One past each boundary must fail.
	_, err := FromInt(int64(math.MaxInt32) + 1).Int32E()
	assert.True(t, errors.Is(err, ErrOverflow), "int32 max + 1")
	_, err = FromInt(int64(math.MinInt32) - 1).Int32E()
	assert.True(t, errors.Is(err, ErrOverflow), "int32 min - 1")
	_, err = MinusOne.Uint64E()
	assert.True(t, errors.Is(err, ErrOverflow), "-1 as uint64")

	// Negative zero converts to unsigned zero rather than failing, matching .NET.
	v, err := Zero.Neg().Uint64E()
	assert.NoError(t, err, "-0 as uint64")
	assert.Equal(t, uint64(0), v, "-0 as uint64")

	// Conversion truncates towards zero; it does not round.
	assert.Equal(t, int64(1), MustParse("1.9").Int64(), "1.9 truncates to 1")
	assert.Equal(t, int64(-1), MustParse("-1.9").Int64(), "-1.9 truncates to -1")
}

func TestFloatConversionEdges(t *testing.T) {
	for _, f := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), math.MaxFloat64, -math.MaxFloat64, 1e29, -1e29} {
		_, err := FromFloat64(f)
		assert.True(t, errors.Is(err, ErrOverflow), "FromFloat64(%v) should report ErrOverflow, got %v", f, err)
	}
	for _, f := range []float32{float32(math.NaN()), float32(math.Inf(1)), math.MaxFloat32} {
		_, err := FromFloat32(f)
		assert.True(t, errors.Is(err, ErrOverflow), "FromFloat32(%v) should report ErrOverflow, got %v", f, err)
	}

	// Subnormals and tiny values round to zero rather than failing.
	for _, f := range []float64{math.SmallestNonzeroFloat64, -math.SmallestNonzeroFloat64, 1e-30} {
		d, err := FromFloat64(f)
		assert.NoError(t, err, "FromFloat64(%v)", f)
		assert.True(t, d.IsZero(), "FromFloat64(%v) should be zero, got %s", f, d)
	}

	// Binary floating point cannot hold 0.1 exactly, so the conversion carries
	// the float's own error. Parsing the text is the exact route.
	d, err := FromFloat64(0.1)
	assert.NoError(t, err, "FromFloat64(0.1)")
	assert.Equal(t, "0.1", d.String(), "FromFloat64 keeps 15 significant digits")
	assert.True(t, MustParse("0.1").Equal(d), "and agrees with the parsed value here")
}

func TestRoundAllModes(t *testing.T) {
	cases := []struct {
		in                                    string
		toEven, awayFromZero, trunc, fl, ceil string
	}{
		{"2.5", "2", "3", "2", "2", "3"},
		{"3.5", "4", "4", "3", "3", "4"},
		{"-2.5", "-2", "-3", "-2", "-3", "-2"},
		{"-3.5", "-4", "-4", "-3", "-4", "-3"},
		{"2.4", "2", "2", "2", "2", "3"},
		{"-2.4", "-2", "-2", "-2", "-3", "-2"},
		{"0.5", "0", "1", "0", "0", "1"},
		{"-0.5", "0", "-1", "0", "-1", "0"},
	}
	for _, c := range cases {
		d := MustParse(c.in)
		for _, m := range []struct {
			name string
			mode RoundingMode
			want string
		}{
			{"ToEven", ToEven, c.toEven},
			{"AwayFromZero", AwayFromZero, c.awayFromZero},
			{"Truncate", Truncate, c.trunc},
			{"Floor", Floor, c.fl},
			{"Ceiling", Ceiling, c.ceil},
		} {
			got := d.Round(0, m.mode)
			assert.Equal(t, m.want, got.String(), "Round(%s, 0, %s)", c.in, m.name)
		}
	}

	// Ceil, Floor and Truncate agree with the corresponding mode at zero places.
	for _, s := range []string{"2.5", "-2.5", "2.4", "-2.4", "0", "-0.5"} {
		d := MustParse(s)
		assert.True(t, d.Ceil().Equal(d.Round(0, Ceiling)), "Ceil vs Round Ceiling for %s", s)
		assert.True(t, d.Floor().Equal(d.Round(0, Floor)), "Floor vs Round Floor for %s", s)
		assert.True(t, d.Truncate().Equal(d.Round(0, Truncate)), "Truncate vs Round Truncate for %s", s)
	}
}

func TestRoundArgumentChecks(t *testing.T) {
	for _, places := range []int{-1, 29, 100} {
		_, panicked := wantPanic(func() { One.Round(places, ToEven) })
		assert.True(t, panicked, "Round with places=%d should panic", places)

		_, err := One.TryRound(places, ToEven)
		assert.True(t, errors.Is(err, ErrScaleRange), "TryRound places=%d, got %v", places, err)
	}

	_, panicked := wantPanic(func() { One.Round(0, RoundingMode(99)) })
	assert.True(t, panicked, "an unknown rounding mode should panic")

	// All five documented modes are accepted; v1 rejected three of them.
	for _, m := range []RoundingMode{ToEven, AwayFromZero, Truncate, Floor, Ceiling} {
		assert.NotPanics(t, func() { MustParse("1.5").Round(0, m) }, "mode %d should be accepted", m)
	}
}

func TestPrecisionLoss(t *testing.T) {
	// A third, to the full 28 digits.
	third := One.Div(MustParse("3"))
	assert.Equal(t, "0.3333333333333333333333333333", third.String(), "1/3")

	// Multiplying back does not recover 1, and that is correct.
	assert.False(t, third.Mul(MustParse("3")).Equal(One), "3 * (1/3) is not 1")

	// The case the C# Remainder comment warns about: dividing a 28-digit value
	// rounds rather than truncating, so multiplying back can exceed the range.
	// MaxValue is odd, so MaxValue/2 rounds up and doubling it overflows. This
	// is .NET's behaviour, not a defect, and Remainder works around it.
	half := MaxValue.Div(Two)
	assert.Equal(t, "39614081257132168796771975168", half.String(), "MaxValue/2 rounds up")
	_, panicked := wantPanic(func() { half.Mul(Two) })
	assert.True(t, panicked, "MaxValue/2 * 2 overflows, because the division rounded up")

	// A product needing more than 96 bits loses scale, not magnitude.
	big := MustParse("79228162514264337593543950335")
	assert.NotPanics(t, func() { big.Mul(MustParse("0.5")) }, "MaxValue * 0.5")

	// Remainder of the largest values, the case DecimalTest2 pins.
	got := MustParse("79228162514264337593543950335").Mod(MustParse("27703302467091960609331879.53200"))
	assert.Equal(t, "24420760848422211464106753.012", got.String(), "the Mono remainder case")
}

func TestIsInteger(t *testing.T) {
	for _, c := range []struct {
		in   string
		want bool
	}{
		{"0", true}, {"-0", true}, {"1", true}, {"-1", true},
		{"1.0", true}, {"1.00", true}, {"100.000", true},
		{"1.1", false}, {"0.1", false}, {"-0.0000000000000000000000000001", false},
		{"79228162514264337593543950335", true},
	} {
		assert.Equal(t, c.want, MustParse(c.in).IsInteger(), "IsInteger(%s)", c.in)
	}
}

func TestAggregates(t *testing.T) {
	d := func(s string) Decimal { return MustParse(s) }

	assert.Equal(t, Zero, Sum(), "Sum of nothing")
	assert.Equal(t, Zero, Mean(), "Mean of nothing")
	assert.Equal(t, Zero, Median(), "Median of nothing")
	assert.Equal(t, Zero, MaxAny(), "MaxAny of nothing")
	assert.Equal(t, Zero, MinAny(), "MinAny of nothing")

	vals := []Decimal{d("3"), d("1"), d("4"), d("1"), d("5")}
	assert.Equal(t, "14", Sum(vals...).String(), "Sum")
	assert.Equal(t, "2.8", Mean(vals...).String(), "Mean")
	assert.Equal(t, "3", Median(vals...).String(), "Median of 5")
	assert.Equal(t, "5", MaxAny(vals...).String(), "MaxAny")
	assert.Equal(t, "1", MinAny(vals...).String(), "MinAny")

	// Median with an even count averages the middle two. v1 panicked here.
	assert.Equal(t, "1.5", Median(d("1"), d("2")).String(), "Median of 2")
	assert.Equal(t, "2.5", Median(d("1"), d("2"), d("3"), d("4")).String(), "Median of 4")
	assert.Equal(t, "3", Median(d("4"), d("1"), d("2"), d("100")).String(), "Median of 4, unsorted")
	assert.Equal(t, "1", Median(d("1")).String(), "Median of 1")

	// Median must not reorder the caller's slice.
	input := []Decimal{d("3"), d("1"), d("2")}
	_ = Median(input...)
	assert.Equal(t, "3", input[0].String(), "Median should not sort the input in place")

	assert.Equal(t, d("5"), Max(d("5"), d("3")), "Max")
	assert.Equal(t, d("3"), Min(d("5"), d("3")), "Min")
	assert.Equal(t, d("5"), d("9").Clamp(d("1"), d("5")), "Clamp above")
	assert.Equal(t, d("1"), d("0").Clamp(d("1"), d("5")), "Clamp below")
	assert.Equal(t, d("3"), d("3").Clamp(d("1"), d("5")), "Clamp inside")
}

func TestNewAndBits(t *testing.T) {
	d, err := New(1, 0, 0, false, 1)
	assert.NoError(t, err, "New")
	assert.Equal(t, "0.1", d.String(), "New(1,0,0,false,1)")

	_, err = New(1, 0, 0, false, 29)
	assert.True(t, errors.Is(err, ErrScaleRange), "New with scale 29, got %v", err)

	// Bits round-trips through FromBits.
	for _, s := range []string{"0", "-0", "1.100", "79228162514264337593543950335", "-0.0000000000000000000000000001"} {
		orig := MustParse(s)
		back, err := FromBits(orig.Bits())
		assert.NoError(t, err, "FromBits(%s)", s)
		assert.Equal(t, orig, back, "FromBits round-trip for %s", s)
	}

	// A flags word with reserved bits set is rejected.
	_, err = FromBits([4]uint32{1, 0, 0, 0x00000001})
	assert.True(t, errors.Is(err, ErrScaleRange), "reserved low bits, got %v", err)
	_, err = FromBits([4]uint32{1, 0, 0, 0x40000000})
	assert.True(t, errors.Is(err, ErrScaleRange), "reserved bit 30, got %v", err)
	_, err = FromBits([4]uint32{1, 0, 0, 29 << 16})
	assert.True(t, errors.Is(err, ErrScaleRange), "scale 29, got %v", err)

	assert.Equal(t, uint8(3), MustParse("1.100").Scale(), "Scale reflects trailing zeros")
	assert.Equal(t, [3]uint32{1100, 0, 0}, MustParse("1.100").Coefficient(), "Coefficient")
}

func TestRandomProducesValidDecimals(t *testing.T) {
	rng := newTestRand(1)
	for i := 0; i < 10000; i++ {
		d := Random(rng)
		assert.True(t, d.Scale() <= 28, "Random produced scale %d", d.Scale())
		assert.Equal(t, uint32(0), d.flags&^(signMask|scaleMask), "Random set reserved bits")
		// Every generated value must survive a round-trip through text.
		back, err := Parse(d.String())
		assert.NoError(t, err, "Parse(Random().String()) for %s", formatBits(d))
		assert.True(t, back.Equal(d), "round-trip changed the value: %s", formatBits(d))
	}
}
