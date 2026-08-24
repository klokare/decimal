package decimal

import (
	"testing"

	"github.com/klokare/decimal/v2/internal/assert"
)

// Cases ported from Mono's own System.Decimal suite:
//
//	mcs/class/corlib/Test/System/DecimalTest.cs
//	mcs/class/corlib/Test/System/DecimalTest2.cs
//	mcs/class/corlib/Test/System/DecimalFormatterTest.cs
//
// The generated tables in testdata/ cover far more ground, but these carry
// something the generator cannot: each one was added because a specific bug was
// found. Several are labelled with the bug number upstream.

// From DecimalTest.TestToString. It runs against InvariantInfo, so the currency
// symbol is the generic sign rather than a national one.
func TestMonoToString(t *testing.T) {
	cases := []struct{ format, value, want string }{
		{"F", "12.345678", "12.35"},
		{"F3", "12.345678", "12.346"},
		{"F0", "12.345678", "12"},
		{"F7", "12.345678", "12.3456780"},
		{"g", "12.345678", "12.345678"},
		{"E", "12.345678", "1.234568E+001"},
		{"E3", "12.345678", "1.235E+001"},
		{"E0", "12.345678", "1E+001"},
		{"e8", "12.345678", "1.23456780e+001"},
		{"F", "0.0012", "0.00"},
		{"F3", "0.0012", "0.001"},
		{"F0", "0.0012", "0"},
		{"F6", "0.0012", "0.001200"},
		{"e", "0.0012", "1.200000e-003"},
		{"E3", "0.0012", "1.200E-003"},
		{"E0", "0.0012", "1E-003"},
		{"E6", "0.0012", "1.200000E-003"},
		{"F4", "-0.001234", "-0.0012"},
		{"E3", "-0.001234", "-1.234E-003"},

		// The general format switches to scientific once the precision runs out.
		{"g", "-0.000012", "-0.000012"},
		{"g0", "-0.000012", "-1.2e-05"},
		{"g2", "-0.000012", "-1.2e-05"},
		{"g20", "-0.000012", "-1.2e-05"},
		{"g", "-0.00012", "-0.00012"},
		{"g4", "-0.00012", "-0.00012"},
		{"g7", "-0.00012", "-0.00012"},
		{"g", "-0.0001234", "-0.0001234"},
		{"g", "-0.0012", "-0.0012"},
		{"g", "-0.001234", "-0.001234"},
		{"g", "-0.012", "-0.012"},
		{"g4", "-0.012", "-0.012"},
		{"g", "-0.12", "-0.12"},
		{"g", "-1.2", "-1.2"},
		{"g4", "-120", "-120"},
		{"g", "-12.000", "-12.000"},
		{"g0", "-12.000", "-12"},
		{"g6", "-12.000", "-12"},
		{"g", "-12", "-12"},
		{"g", "-120", "-120"},
		{"g", "-1200", "-1200"},
		{"g4", "-1200", "-1200"},
		{"g", "-1234", "-1234"},
		{"g", "-12000", "-12000"},
		{"g4", "-12000", "-1.2e+04"},
		{"g5", "-12000", "-12000"},
		{"g", "-12345", "-12345"},
		{"g", "-120000", "-120000"},
		{"g4", "-120000", "-1.2e+05"},
		{"g5", "-120000", "-1.2e+05"},
		{"g6", "-120000", "-120000"},
		{"g", "-123456.1", "-123456.1"},
		{"g5", "-123456.1", "-1.2346e+05"},
		{"g6", "-123456.1", "-123456"},
		{"g", "-1200000", "-1200000"},
		{"g", "-1234567.1", "-1234567.1"},
		{"g", "-12000000", "-12000000"},
		{"g", "-12345678.1", "-12345678.1"},
		{"g", "-12000000000000000000", "-12000000000000000000"},

		{"F", "-123", "-123.00"},
		{"F3", "-123", "-123.000"},
		{"F0", "-123", "-123"},
		{"E3", "-123", "-1.230E+002"},
		{"E0", "-123", "-1E+002"},
		{"E", "-123", "-1.230000E+002"},

		{"F3", "-79228162514264337593543950335", "-79228162514264337593543950335.000"},
		{"F", "-79228162514264337593543950335", "-79228162514264337593543950335.00"},
		{"F0", "-79228162514264337593543950335", "-79228162514264337593543950335"},
		{"E", "-79228162514264337593543950335", "-7.922816E+028"},
		{"E3", "-79228162514264337593543950335", "-7.923E+028"},
		{"E28", "-79228162514264337593543950335", "-7.9228162514264337593543950335E+028"},
		{"E30", "-79228162514264337593543950335", "-7.922816251426433759354395033500E+028"},
		{"E0", "-79228162514264337593543950335", "-8E+028"},

		{"N3", "-79228162514264337593543950335", "-79,228,162,514,264,337,593,543,950,335.000"},
		{"N0", "-79228162514264337593543950335", "-79,228,162,514,264,337,593,543,950,335"},
		{"N", "-79228162514264337593543950335", "-79,228,162,514,264,337,593,543,950,335.00"},
		{"n3", "-79228162514264337593543950335", "-79,228,162,514,264,337,593,543,950,335.000"},
		{"n0", "-79228162514264337593543950335", "-79,228,162,514,264,337,593,543,950,335"},
		{"n", "-79228162514264337593543950335", "-79,228,162,514,264,337,593,543,950,335.00"},

		// InvariantInfo's currency symbol is U+00A4 and its negative pattern
		// is 0, which parenthesises.
		{"C", "123456.7890", "¤123,456.79"},
		{"C", "-123456.7890", "(¤123,456.79)"},
		{"C3", "1123456.7890", "¤1,123,456.789"},

		{"P", "123456.7891", "12,345,678.91 %"},
		{"P", "-123456.7892", "-12,345,678.92 %"},
		{"P3", "1234.56789", "123,456.789 %"},
	}

	for _, c := range cases {
		got, err := FormatWith(MustParse(c.value), c.format, Invariant)
		assert.NoError(t, err, "Format(%s, %q)", c.value, c.format)
		assert.Equal(t, c.want, got, "Format(%s, %q)", c.value, c.format)
	}
}

// From DecimalFormatterTest, which runs against en-US. D and X are integral-only
// and must be rejected.
func TestMonoFormatterRejectsIntegralFormats(t *testing.T) {
	d := MustParse("1.0000001")
	for _, format := range []string{"D2", "X2", "D", "X", "d", "x"} {
		_, err := FormatWith(d, format, EnUS)
		assert.ErrorIs(t, err, ErrFormat, "Format(%q) should be rejected", format)
	}
}

// From DecimalFormatterTest.FormatTest, spot-checking each specifier against
// en-US, where the currency symbol is $ and the negative pattern is 1.
func TestMonoFormatterEnUS(t *testing.T) {
	cases := []struct{ value, format, want string }{
		{"1.0034", "C", "$1.00"},
		{"1.0034", "C0", "$1"},
		{"1.0034", "C1", "$1.0"},
		{"1.0034", "C4", "$1.0034"},
		{"1.0034", "C6", "$1.003400"},
		{"1.0034", "E", "1.003400E+000"},
		{"1.0034", "E0", "1E+000"},
		{"1.0034", "E4", "1.0034E+000"},
		{"1.0034", "F", "1.003"},
		{"1.0034", "F0", "1"},
		{"1.0034", "F2", "1.00"},
		{"1.0034", "F4", "1.0034"},
		{"1.0034", "G", "1.0034"},
		{"1.0034", "N", "1.003"},
		{"1.0034", "N0", "1"},
		{"1.0034", "N4", "1.0034"},
		// en-US has PercentPositivePattern 1, which is "#%" with no space;
		// the invariant culture uses pattern 0, "# %". The Mono fixture
		// reconfigured its culture, so its expectations differ here.
		{"1.0034", "P0", "100%"},
		{"1.0034", "P2", "100.34%"},
	}
	for _, c := range cases {
		got, err := FormatWith(MustParse(c.value), c.format, EnUS)
		assert.NoError(t, err, "Format(%s, %q)", c.value, c.format)
		assert.Equal(t, c.want, got, "Format(%s, %q) under en-US", c.value, c.format)
	}
}

// DecimalTest.Round_OddValue and Round_EvenValue, at every scale.
func TestMonoRoundEveryScale(t *testing.T) {
	five := MustParse("5.5555555555555555555555555555")
	want := []string{
		"6", "5.6", "5.56", "5.556", "5.5556", "5.55556", "5.555556",
		"5.5555556", "5.55555556", "5.555555556", "5.5555555556",
		"5.55555555556", "5.555555555556", "5.5555555555556",
		"5.55555555555556", "5.555555555555556", "5.5555555555555556",
		"5.55555555555555556", "5.555555555555555556", "5.5555555555555555556",
		"5.55555555555555555556", "5.555555555555555555556",
		"5.5555555555555555555556", "5.55555555555555555555556",
		"5.555555555555555555555556", "5.5555555555555555555555556",
		"5.55555555555555555555555556", "5.555555555555555555555555556",
		"5.5555555555555555555555555555",
	}
	for places, w := range want {
		got := five.Round(places, ToEven)
		assert.True(t, got.Equal(MustParse(w)),
			"Round(5.5555..., %d) = %s, want %s", places, got, w)
	}

	// Banker's rounding: a midpoint goes to the nearest even digit.
	for _, c := range []struct {
		in     string
		places int
		want   string
	}{
		{"2.5", 0, "2"},
		{"2.25", 1, "2.2"},
		{"2.225", 2, "2.22"},
		{"2.2225", 3, "2.222"},
		{"2.22225", 4, "2.2222"},
		{"-2.5", 0, "-2"},
		{"-2.25", 1, "-2.2"},
		{"3.5", 0, "4"},
		{"-3.5", 0, "-4"},
	} {
		got := MustParse(c.in).Round(c.places, ToEven)
		assert.True(t, got.Equal(MustParse(c.want)),
			"Round(%s, %d) = %s, want %s", c.in, c.places, got, c.want)
	}
}

// DecimalTest.MidpointRoundingAwayFromZero.
func TestMonoMidpointAwayFromZero(t *testing.T) {
	for _, c := range []struct {
		in     string
		places int
		want   string
	}{
		{"1.5", 0, "2"}, {"2.5", 0, "3"}, {"-1.5", 0, "-2"}, {"-2.5", 0, "-3"},
		{"1.25", 1, "1.3"}, {"1.35", 1, "1.4"}, {"-1.25", 1, "-1.3"},
	} {
		got := MustParse(c.in).Round(c.places, AwayFromZero)
		assert.True(t, got.Equal(MustParse(c.want)),
			"Round(%s, %d, AwayFromZero) = %s, want %s", c.in, c.places, got, c.want)
	}
}

// DecimalTest2.TestRemainder, plus DecimalTest.Remainder.
func TestMonoRemainder(t *testing.T) {
	for _, c := range []struct{ a, b, want string }{
		{"3.6", "1.3", "1.0"},
		{"79228162514264337593543950335", "27703302467091960609331879.53200", "24420760848422211464106753.012"},
		{"45937986975432", "43987453", "42334506"},
		{"45937986975000", "5000", "0"},
		{"-54789548973.6234", "1.3356", "-0.1074"},
	} {
		got := MustParse(c.a).Mod(MustParse(c.b))
		assert.True(t, got.Equal(MustParse(c.want)),
			"%s %% %s = %s, want %s", c.a, c.b, got, c.want)
	}
}

// DecimalTest.ParseAndKeepPrecision, bug #59425: parsing must not discard the
// scale the text asked for.
func TestMonoParseKeepsPrecision(t *testing.T) {
	for _, s := range []string{
		"0", "0.0", "0.00", "0.000", "0.0000000000000000000000000000",
		"1", "1.0", "1.00", "1.000",
		"-0.0", "-0.00",
		"12345678901234567890123456789",
	} {
		d := MustParse(s)
		assert.Equal(t, s, canonicalZero(s, d), "MustParse(%q).String()", s)
	}
}

// canonicalZero accounts for String dropping the sign of a zero, which is what
// .NET does and which TestNegativeZeroEncoding pins.
func canonicalZero(in string, d Decimal) string {
	got := d.String()
	if d.IsZero() && len(in) > 0 && in[0] == '-' {
		return "-" + got
	}
	return got
}

// DecimalTest.TrailingZerosBug, SUSE #655780, and RoundToString, bug #21764.
func TestMonoTrailingZeros(t *testing.T) {
	assert.Equal(t, "1.0", MustParse("1.0").String(), "1.0 keeps its zero")
	assert.Equal(t, "1.000", MustParse("1.000").String(), "1.000 keeps its zeros")
	assert.Equal(t, "0.000", MustParse("0.000").String(), "0.000 keeps its zeros")

	// Rounding reduces the scale to the requested places but never increases it,
	// so 1.1 rounded to two places stays 1.1 rather than becoming 1.10.
	assert.Equal(t, "1.00", MustParse("1.004").Round(2, ToEven).String(), "Round reduces scale")
	assert.Equal(t, "1.1", MustParse("1.1").Round(2, ToEven).String(), "Round does not add scale")
}

// DecimalTest.DecimalDivision_24411, Xamarin bug #24411.
func TestMonoDivision24411(t *testing.T) {
	a := MustParse("1.0000000000000000000000000000")
	b := MustParse("3")
	got := a.Div(b)
	assert.Equal(t, "0.3333333333333333333333333333", got.String(), "1.0000... / 3")
}

// DecimalTest.TestConstructDouble and TestConstructSingle.
func TestMonoConstructFromFloat(t *testing.T) {
	for _, c := range []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{1, "1"},
		{-1, "-1"},
		{1.5, "1.5"},
		{-1.5, "-1.5"},
		{0.1, "0.1"},
		{1e10, "10000000000"},
		{1e-10, "0.0000000001"},
		{123456789012345, "123456789012345"},
	} {
		d, err := FromFloat64(c.in)
		assert.NoError(t, err, "FromFloat64(%v)", c.in)
		assert.Equal(t, c.want, d.String(), "FromFloat64(%v)", c.in)
	}

	for _, c := range []struct {
		in   float32
		want string
	}{
		{0, "0"}, {1, "1"}, {-1, "-1"}, {1.5, "1.5"}, {100000, "100000"},
	} {
		d, err := FromFloat32(c.in)
		assert.NoError(t, err, "FromFloat32(%v)", c.in)
		assert.Equal(t, c.want, d.String(), "FromFloat32(%v)", c.in)
	}
}

// DecimalTest.TestConstants.
func TestMonoConstants(t *testing.T) {
	assert.Equal(t, "0", Zero.String(), "Zero")
	assert.Equal(t, "1", One.String(), "One")
	assert.Equal(t, "-1", MinusOne.String(), "MinusOne")
	assert.Equal(t, "79228162514264337593543950335", MaxValue.String(), "MaxValue")
	assert.Equal(t, "-79228162514264337593543950335", MinValue.String(), "MinValue")
	assert.Equal(t, "0.0000000000000000000000000001", SmallestNonZero.String(), "SmallestNonZero")

	// They are values, not pointers: modifying a copy leaves the original alone.
	z := Zero
	z = z.Add(One)
	assert.Equal(t, "0", Zero.String(), "Zero must not be mutable through a copy")
	assert.Equal(t, "1", z.String(), "the copy")
}

// DecimalTest.TestPartConstruct: the four-part constructor.
func TestMonoPartConstruct(t *testing.T) {
	d, err := New(0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, false, 0)
	assert.NoError(t, err, "New at the maximum")
	assert.Equal(t, MaxValue, d, "New should build MaxValue")

	d, err = New(1, 0, 0, true, 28)
	assert.NoError(t, err, "New at the smallest")
	assert.Equal(t, "-0.0000000000000000000000000001", d.String(), "the smallest negative")

	_, err = New(1, 0, 0, false, 29)
	assert.ErrorIs(t, err, ErrScaleRange, "scale 29 must be rejected")
}

// DecimalTest2.TestCompare: a full comparison matrix over values chosen to
// stress the scale-alignment path.
func TestMonoCompareMatrix(t *testing.T) {
	values := []string{
		"0", "1", "-1", "2", "10", "0.1", "0.11",
		"79228162514264337593543950335",
		"-79228162514264337593543950335",
		"27703302467091960609331879.532",
		"-3203854.9559968181492513385018",
		"-3203854.9559968181492513385017",
		"-48466870444188873796420.0286",
		"-48466870444188873796420.02860",
	}

	// The two values that differ only in scale must compare equal.
	a := MustParse("-48466870444188873796420.0286")
	b := MustParse("-48466870444188873796420.02860")
	assert.Equal(t, 0, a.Cmp(b), "values differing only in scale")
	assert.False(t, a == b, "but not identical")

	// Cmp must be a consistent total order over the whole set.
	for _, x := range values {
		dx := MustParse(x)
		for _, y := range values {
			dy := MustParse(y)
			c := dx.Cmp(dy)
			assert.Equal(t, -c, dy.Cmp(dx), "Cmp(%s,%s) should be antisymmetric", x, y)
			assert.Equal(t, c < 0, dx.LessThan(dy), "LessThan(%s,%s)", x, y)
			assert.Equal(t, c > 0, dx.GreaterThan(dy), "GreaterThan(%s,%s)", x, y)
			assert.Equal(t, c == 0, dx.Equal(dy), "Equal(%s,%s)", x, y)
			assert.Equal(t, c <= 0, dx.LessThanOrEqual(dy), "LessThanOrEqual(%s,%s)", x, y)
			assert.Equal(t, c >= 0, dx.GreaterThanOrEqual(dy), "GreaterThanOrEqual(%s,%s)", x, y)
		}
	}
}

// DecimalTest.TestParse and its several bug-report companions.
func TestMonoParse(t *testing.T) {
	ok := []struct{ in, want string }{
		{"1", "1"},
		{"-1", "-1"},
		{"1.0", "1.0"},
		{"1,234", "1234"},
		{"1,234,567.89", "1234567.89"},
		{" 12 ", "12"},
		{"79228162514264337593543950335", "79228162514264337593543950335"},
		{"-79228162514264337593543950335", "-79228162514264337593543950335"},
	}
	for _, c := range ok {
		d, err := Parse(c.in)
		assert.NoError(t, err, "Parse(%q)", c.in)
		assert.Equal(t, c.want, d.String(), "Parse(%q)", c.in)
	}

	// DecimalTest.Parse_Int64_Overflow and the range checks.
	bad := []string{
		"", " ", ".", "-", "+", "abc", "1.2.3", "1e5",
		"79228162514264337593543950336",
		"-79228162514264337593543950336",
	}
	for _, in := range bad {
		_, err := Parse(in)
		assert.Error(t, err, "Parse(%q) should fail", in)
	}
}

// DecimalTest.CastTruncRounding: casting truncates rather than rounding.
func TestMonoCastTruncates(t *testing.T) {
	for _, c := range []struct {
		in   string
		want int64
	}{
		{"1.5", 1}, {"-1.5", -1}, {"1.9", 1}, {"-1.9", -1},
		{"0.9", 0}, {"-0.9", 0}, {"2.5", 2}, {"-2.5", -2},
	} {
		assert.Equal(t, c.want, MustParse(c.in).Int64(), "Int64(%s)", c.in)
	}
}

// DecimalTest.TestFloorTruncate.
func TestMonoFloorTruncate(t *testing.T) {
	for _, c := range []struct{ in, floor, ceil, trunc string }{
		{"1.5", "1", "2", "1"},
		{"-1.5", "-2", "-1", "-1"},
		{"0.5", "0", "1", "0"},
		{"-0.5", "-1", "0", "0"},
		{"1", "1", "1", "1"},
		{"-1", "-1", "-1", "-1"},
		{"0", "0", "0", "0"},
	} {
		d := MustParse(c.in)
		assert.True(t, d.Floor().Equal(MustParse(c.floor)), "Floor(%s) = %s", c.in, d.Floor())
		assert.True(t, d.Ceil().Equal(MustParse(c.ceil)), "Ceil(%s) = %s", c.in, d.Ceil())
		assert.True(t, d.Truncate().Equal(MustParse(c.trunc)), "Truncate(%s) = %s", c.in, d.Truncate())
	}
}
