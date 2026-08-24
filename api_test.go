package decimal

import (
	"errors"
	"math"
	"testing"

	"github.com/klokare/decimal/v2/internal/assert"
)

// Covers the parts of the exported surface the golden tables do not reach, so
// that every exported function is exercised at least once.

func TestTryVariants(t *testing.T) {
	a, b := MustParse("6"), MustParse("4")

	for _, c := range []struct {
		name   string
		try    func() (Decimal, error)
		direct func() Decimal
	}{
		{"TryAdd", func() (Decimal, error) { return a.TryAdd(b) }, func() Decimal { return a.Add(b) }},
		{"TrySub", func() (Decimal, error) { return a.TrySub(b) }, func() Decimal { return a.Sub(b) }},
		{"TryMul", func() (Decimal, error) { return a.TryMul(b) }, func() Decimal { return a.Mul(b) }},
		{"TryDiv", func() (Decimal, error) { return a.TryDiv(b) }, func() Decimal { return a.Div(b) }},
		{"TryMod", func() (Decimal, error) { return a.TryMod(b) }, func() Decimal { return a.Mod(b) }},
	} {
		got, err := c.try()
		assert.NoError(t, err, c.name)
		assert.Equal(t, c.direct(), got, "%s should agree with the panicking form", c.name)
	}

	// And the failing paths.
	for _, c := range []struct {
		name string
		try  func() (Decimal, error)
		want error
	}{
		{"TryAdd overflow", func() (Decimal, error) { return MaxValue.TryAdd(One) }, ErrOverflow},
		{"TrySub overflow", func() (Decimal, error) { return MinValue.TrySub(One) }, ErrOverflow},
		{"TryMul overflow", func() (Decimal, error) { return MaxValue.TryMul(Two) }, ErrOverflow},
		{"TryDiv by zero", func() (Decimal, error) { return One.TryDiv(Zero) }, ErrDivideByZero},
		{"TryMod by zero", func() (Decimal, error) { return One.TryMod(Zero) }, ErrDivideByZero},
		{"TryRound range", func() (Decimal, error) { return One.TryRound(99, ToEven) }, ErrScaleRange},
	} {
		got, err := c.try()
		assert.ErrorIs(t, err, c.want, c.name)
		assert.Equal(t, Decimal{}, got, "%s should zero the result", c.name)
	}
}

func TestIntAndUint(t *testing.T) {
	assert.Equal(t, 42, MustParse("42.9").Int(), "Int truncates")
	assert.Equal(t, -42, MustParse("-42.9").Int(), "Int truncates towards zero")
	assert.Equal(t, uint(42), MustParse("42.9").Uint(), "Uint")

	v, err := MustParse("42.9").IntE()
	assert.NoError(t, err, "IntE")
	assert.Equal(t, 42, v, "IntE")

	u, err := MustParse("42.9").UintE()
	assert.NoError(t, err, "UintE")
	assert.Equal(t, uint(42), u, "UintE")

	_, err = MinusOne.UintE()
	assert.ErrorIs(t, err, ErrOverflow, "UintE of a negative")

	// Int and Uint follow the platform width.
	if bits := ^uint(0) >> 63; bits == 1 {
		assert.Equal(t, int(math.MinInt64), FromInt(math.MinInt64).Int(), "64-bit Int")
	}
}

func TestOACurrency(t *testing.T) {
	for _, c := range []struct {
		in   string
		want int64
	}{
		{"0", 0},
		{"1", 10000},
		{"-1", -10000},
		{"1.5", 15000},
		{"0.0001", 1},
		{"1.00005", 10000}, // rounds to even at the fourth place
		{"123.4567", 1234567},
	} {
		assert.Equal(t, c.want, MustParse(c.in).ToOACurrency(), "ToOACurrency(%s)", c.in)
	}

	// And back.
	for _, cy := range []int64{0, 1, -1, 10000, -10000, 1234567} {
		d := FromOACurrency(cy)
		assert.Equal(t, cy, d.ToOACurrency(), "OA currency round-trip for %d", cy)
	}
	assert.Equal(t, "1.0000", FromOACurrency(10000).String(), "FromOACurrency keeps four places")
}

func TestFromIntAllWidths(t *testing.T) {
	assert.Equal(t, "127", FromInt(int8(127)).String(), "int8")
	assert.Equal(t, "-128", FromInt(int8(-128)).String(), "int8 minimum")
	assert.Equal(t, "32767", FromInt(int16(32767)).String(), "int16")
	assert.Equal(t, "2147483647", FromInt(int32(math.MaxInt32)).String(), "int32")
	assert.Equal(t, "-9223372036854775808", FromInt(int64(math.MinInt64)).String(), "int64 minimum")
	assert.Equal(t, "255", FromUint(uint8(255)).String(), "uint8")
	assert.Equal(t, "65535", FromUint(uint16(65535)).String(), "uint16")
	assert.Equal(t, "4294967295", FromUint(uint32(math.MaxUint32)).String(), "uint32")
	assert.Equal(t, "18446744073709551615", FromUint(uint64(math.MaxUint64)).String(), "uint64")

	// Named types satisfy the constraints too.
	type cents int64
	assert.Equal(t, "500", FromInt(cents(500)).String(), "a named integer type")
}

func TestMustPanics(t *testing.T) {
	_, panicked := wantPanic(func() { MustParse("not a number") })
	assert.True(t, panicked, "MustParse should panic on bad input")

	_, panicked = wantPanic(func() { MustFormat(One, "D") })
	assert.True(t, panicked, "MustFormat should panic on a bad specifier")

	assert.NotPanics(t, func() { MustParse("1.5") }, "MustParse on good input")
	assert.NotPanics(t, func() { MustFormat(One, "G") }, "MustFormat on a good specifier")
}

// A panic that is not one of this package's errors must not be swallowed by the
// Try variants' recover.
func TestForeignPanicsPropagate(t *testing.T) {
	var d Decimal
	err := recovered(nil, &d)
	assert.NoError(t, err, "no panic gives no error")

	rec, panicked := wantPanic(func() {
		var result Decimal
		defer func() { _ = recovered(recover(), &result) }()
		panic("something else entirely")
	})
	assert.True(t, panicked, "a foreign panic should be re-raised")
	assert.Equal(t, "something else entirely", rec, "and unchanged")
}

func TestErrorsAreDistinct(t *testing.T) {
	all := []error{ErrOverflow, ErrDivideByZero, ErrScaleRange, ErrSyntax, ErrFormat}
	for i, a := range all {
		for j, b := range all {
			if i == j {
				continue
			}
			assert.False(t, errors.Is(a, b), "%v should not match %v", a, b)
		}
	}
}

// The division carry path, where rounding overflows 96 bits and the result has
// to be scaled back down. It is only reachable with operands that fill the
// range, so it is exercised explicitly.
func TestDivisionCarryPath(t *testing.T) {
	cases := []struct{ a, b string }{
		{"79228162514264337593543950335", "79228162514264337593543950335"},
		{"79228162514264337593543950334", "79228162514264337593543950335"},
		{"79228162514264337593543950335", "0.9999999999999999999999999999"},
		{"7922816251426433759354395033.5", "0.7922816251426433759354395033"},
		{"79228162514264337593543950335", "1.0000000000000000000000000001"},
	}
	for _, c := range cases {
		a, b := MustParse(c.a), MustParse(c.b)
		got, err := a.TryDiv(b)
		if err != nil {
			assert.ErrorIs(t, err, ErrOverflow, "%s / %s", c.a, c.b)
			continue
		}
		// Whatever comes back must be a legal Decimal.
		assert.True(t, got.Scale() <= 28, "%s / %s produced scale %d", c.a, c.b, got.Scale())
		assert.Equal(t, uint32(0), got.flags&^(signMask|scaleMask), "%s / %s set reserved bits", c.a, c.b)
	}
}

func TestNumberFormatNilIsInvariant(t *testing.T) {
	got, err := FormatWith(MustParse("1234.5"), "N2", nil)
	assert.NoError(t, err, "FormatWith(nil)")
	assert.Equal(t, "1,234.50", got, "a nil NumberFormat should behave as Invariant")

	d, err := ParseStyle("1,234.50", StyleNumber, nil)
	assert.NoError(t, err, "ParseStyle(nil)")
	assert.Equal(t, "1234.50", d.String(), "a nil NumberFormat should behave as Invariant")
}

func TestFormatWithCustomNumberFormat(t *testing.T) {
	// A European-style culture, to prove the separators really are injectable.
	de := Invariant.Clone()
	de.NumberDecimalSeparator = ","
	de.NumberGroupSeparator = "."
	de.CurrencySymbol = "€"
	de.CurrencyDecimalSeparator = ","
	de.CurrencyGroupSeparator = "."
	de.CurrencyPositivePattern = 3 // "n $"

	got, err := FormatWith(MustParse("1234567.89"), "N2", de)
	assert.NoError(t, err, "N2")
	assert.Equal(t, "1.234.567,89", got, "German-style number")

	got, err = FormatWith(MustParse("1234567.89"), "C2", de)
	assert.NoError(t, err, "C2")
	assert.Equal(t, "1.234.567,89 €", got, "German-style currency")

	// And parsing with the same format.
	d, err := ParseStyle("1.234.567,89", StyleNumber, de)
	assert.NoError(t, err, "ParseStyle")
	assert.Equal(t, "1234567.89", d.String(), "German-style parse")
}

// Group sizes other than the default [3], including the Indian lakh/crore
// grouping, which uses a repeating final size.
func TestGroupSizes(t *testing.T) {
	indian := Invariant.Clone()
	indian.NumberGroupSizes = []int{3, 2}

	got, err := FormatWith(MustParse("12345678.9"), "N1", indian)
	assert.NoError(t, err, "N1")
	assert.Equal(t, "1,23,45,678.9", got, "Indian grouping")

	// A zero entry disables grouping from that point.
	none := Invariant.Clone()
	none.NumberGroupSizes = []int{0}
	got, err = FormatWith(MustParse("12345678.9"), "N1", none)
	assert.NoError(t, err, "N1 with grouping disabled")
	assert.Equal(t, "12345678.9", got, "no grouping")
}

// overflowUnscale handles the case where rounding during division carries the
// quotient past 96 bits: it reloads the high word with 2^32/10, divides the rest
// by ten and drops the scale by one. No operand pair in the golden tables
// reaches it, so it is checked directly against the reference algorithm.
func TestOverflowUnscale(t *testing.T) {
	// reference reimplements the C# in the most literal way available, using
	// 128-bit arithmetic via big-endian byte juggling, so that the optimised
	// version in calc.go is checked against something independent.
	reference := func(u0, u1 uint32, sticky bool) (r0, r1, r2 uint32) {
		// The value being unscaled is 2^96 + (u1<<32 | u0), divided by ten.
		const highBit = uint64(1) << 32
		r2 = uint32(highBit / 10)
		tmp := ((highBit % 10) << 32) + uint64(u1)
		r1 = uint32(tmp / 10)
		tmp = ((tmp - uint64(r1)*10) << 32) + uint64(u0)
		r0 = uint32(tmp / 10)
		rem := uint32(tmp - uint64(r0)*10)
		if rem > 5 || rem == 5 && (sticky || r0&1 != 0) {
			// propagate a carry through the three words
			if r0++; r0 == 0 {
				if r1++; r1 == 0 {
					r2++
				}
			}
		}
		return r0, r1, r2
	}

	for _, c := range []struct {
		u0, u1 uint32
		sticky bool
	}{
		{0, 0, false},
		{0, 0, true},
		{5, 0, false},
		{55, 0, true},
		{0xFFFFFFFF, 0xFFFFFFFF, false},
		{0xFFFFFFFF, 0xFFFFFFFF, true},
		{9, 0, false},
		{9, 0, true},
		{0x7FFFFFFF, 0x80000000, true},
	} {
		buf := &buf12{U0: c.u0, U1: c.u1}
		gotScale := overflowUnscale(buf, 5, c.sticky)
		assert.Equal(t, int32(4), gotScale, "the scale drops by one")

		w0, w1, w2 := reference(c.u0, c.u1, c.sticky)
		assert.Equal(t, w0, buf.U0, "U0 for %#x,%#x sticky=%v", c.u1, c.u0, c.sticky)
		assert.Equal(t, w1, buf.U1, "U1 for %#x,%#x sticky=%v", c.u1, c.u0, c.sticky)
		assert.Equal(t, w2, buf.U2, "U2 for %#x,%#x sticky=%v", c.u1, c.u0, c.sticky)
	}

	// A scale that cannot be reduced any further is an overflow.
	_, panicked := wantPanic(func() { overflowUnscale(&buf12{}, 0, false) })
	assert.True(t, panicked, "unscaling at scale 0 should overflow")
}

func TestJSONNumberString(t *testing.T) {
	assert.Equal(t, "1.100", JSONNumber(MustParse("1.100")).String(), "JSONNumber.String")
}
