package decimal

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEquals(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:24
	assert.True(t, Zero.Equal(Zero))
	assert.False(t, Zero.Equal(One))
	assert.True(t, MaxValue.Equal(MaxValue))
	assert.True(t, MinValue.Equal(MinValue))
	assert.False(t, MaxValue.Equal(MinValue))
	assert.False(t, MinValue.Equal(MaxValue))
}

func TestGreaterThan(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:24
	assert.False(t, Zero.GreaterThan(Zero))
	assert.False(t, Zero.GreaterThan(One))
	assert.True(t, One.GreaterThan(Zero))
	assert.False(t, MaxValue.GreaterThan(MaxValue))
	assert.False(t, MinValue.GreaterThan(MinValue))
	assert.False(t, MinValue.GreaterThan(MaxValue))
	assert.True(t, MaxValue.GreaterThan(MinValue))
}

func TestGreaterThanOrEqual(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:89
	assert.True(t, Zero.GreaterThanOrEqual(Zero))
	assert.False(t, Zero.GreaterThanOrEqual(One))
	assert.True(t, One.GreaterThanOrEqual(Zero))
	assert.True(t, MaxValue.GreaterThanOrEqual(MaxValue))
	assert.True(t, MinValue.GreaterThanOrEqual(MinValue))
	assert.False(t, MinValue.GreaterThanOrEqual(MaxValue))
	assert.True(t, MaxValue.GreaterThanOrEqual(MinValue))
}

func TestNotEquals(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:101
	assert.False(t, !Zero.Equal(Zero))
	assert.True(t, !Zero.Equal(One))
	assert.True(t, !One.Equal(Zero))
	assert.False(t, !MaxValue.Equal(MaxValue))
	assert.False(t, !MinValue.Equal(MinValue))
	assert.True(t, !MaxValue.Equal(MinValue))
	assert.True(t, !MinValue.Equal(MaxValue))
}

func TestLessThan(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:114
	assert.False(t, Zero.LessThan(Zero))
	assert.True(t, Zero.LessThan(One))
	assert.False(t, One.LessThan(Zero))
	d5 := NewFromInt32(5)
	d3 := NewFromInt32(3)
	assert.False(t, d5.LessThan(d3))
	assert.False(t, MaxValue.LessThan(MaxValue))
	assert.False(t, MinValue.LessThan(MinValue))
	assert.True(t, MinValue.LessThan(MaxValue))
	assert.False(t, MaxValue.LessThan(MinValue))
}

func TestLessThanOrEqual(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:131
	assert.True(t, Zero.LessThanOrEqual(Zero))
	assert.True(t, Zero.LessThanOrEqual(One))
	assert.False(t, One.LessThanOrEqual(Zero))
	assert.True(t, MaxValue.LessThanOrEqual(MaxValue))
	assert.True(t, MinValue.LessThanOrEqual(MinValue))
	assert.True(t, MinValue.LessThanOrEqual(MaxValue))
	assert.False(t, MaxValue.LessThanOrEqual(MinValue))
}

func verifyAdd(t *testing.T, d1, d2, expected Decimal, isPanic bool) {
	if isPanic {
		assert.Panics(t, func() { d1.Add(d2) })
	} else {
		if assert.NotPanics(t, func() { d1.Add(d2) }) {
			result1 := d1.Add(d2)
			assert.True(t, result1.Equal(expected))
		}
	}
}
func TestAdd(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:173
	verifyAdd(t, NewFromInt32(1), NewFromInt32(1), NewFromInt32(2), false)
	verifyAdd(t, NewFromInt32(-1), NewFromInt32(1), Zero, false) // TODO: fails with = but not .Equal because sign is kept as negative, resulting in 0 ≠ -0
	verifyAdd(t, NewFromInt32(1), NewFromInt32(-1), Zero, false)
	verifyAdd(t, MaxValue, Zero, MaxValue, false)
	verifyAdd(t, MinValue, Zero, MinValue, false)
	verifyAdd(t, NewFromString("79228162514264337593543950330"), NewFromString("5"), MaxValue, false)
	verifyAdd(t, NewFromString("79228162514264337593543950330"), NewFromString("-5"), NewFromString("79228162514264337593543950325"), false)
	verifyAdd(t, NewFromString("-79228162514264337593543950330"), NewFromString("-5"), MinValue, false)
	verifyAdd(t, NewFromString("-79228162514264337593543950330"), NewFromString("5"), NewFromString("-79228162514264337593543950325"), false)
	verifyAdd(t, NewFromString("1234.5678"), NewFromString("0.00009"), NewFromString("1234.56789"), false)
	verifyAdd(t, NewFromString("-1234.5678"), NewFromString("0.00009"), NewFromString("-1234.56771"), false)
	verifyAdd(t, NewFromString("0.1111111111111111111111111111"), NewFromString("0.1111111111111111111111111111"), NewFromString("0.2222222222222222222222222222"), false)
	verifyAdd(t, NewFromString("0.5555555555555555555555555555"), NewFromString("0.5555555555555555555555555555"), NewFromString("1.1111111111111111111111111110"), false)

	verifyAdd(t, MaxValue, MaxValue, Zero, true)
	verifyAdd(t, NewFromString("79228162514264337593543950330"), NewFromString("6"), Zero, true)
	verifyAdd(t, NewFromString("-79228162514264337593543950330"), NewFromString("-6"), Zero, true)
}

func TestCeil(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:202
	assert.True(t, NewFromString("123").Equal(NewFromString("123").Ceil()))
	assert.True(t, NewFromString("124").Equal(NewFromString("123.123").Ceil()))
	assert.True(t, NewFromString("-123").Equal(NewFromString("-123.123").Ceil()))
	assert.True(t, NewFromString("124").Equal(NewFromString("123.567").Ceil()))
	assert.True(t, NewFromString("-123").Equal(NewFromString("-123.567").Ceil()))
}

func verifyDivide(t *testing.T, d1, d2, expected Decimal, isPanic bool) {
	if isPanic {
		assert.Panics(t, func() { d1.Div(d2) })
	} else {
		if assert.NotPanics(t, func() { d1.Div(d2) }) {
			result1 := d1.Div(d2)
			assert.True(t, result1.Equal(expected))
		}
	}
}

func TestDiv(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:231

	// Vanilla cases
	verifyDivide(t, One, One, One, false)
	verifyDivide(t, MaxValue, MinValue, MinusOne, false)
	verifyDivide(t, NewFromString("0.9214206543486529434634231456"), MaxValue, Zero, false)
	verifyDivide(t, NewFromString("38214206543486529434634231456"), NewFromString("0.49214206543486529434634231456"), NewFromString("77648730371625094566866001277"), false)
	verifyDivide(t, NewFromString("-78228162514264337593543950335"), MaxValue, NewFromString("-0.987378225516463811113412343"), false)
	verifyDivide(t, NewFromString("5").Add(NewFromString("10")), NewFromString("2"), NewFromString("7.5"), false)
	verifyDivide(t, NewFromString("10"), NewFromString("2"), NewFromString("5"), false)

	// Tests near MaxValue (VSWhidbey #389382 <- mono reference)
	verifyDivide(t, NewFromString("792281625142643375935439503.4"), NewFromString("0.1"), NewFromString("7922816251426433759354395034"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950.34"), NewFromString("0.1"), NewFromString("792281625142643375935439503.4"), false)
	verifyDivide(t, NewFromString("7922816251426433759354395.034"), NewFromString("0.1"), NewFromString("79228162514264337593543950.34"), false)
	verifyDivide(t, NewFromString("792281625142643375935439.5034"), NewFromString("0.1"), NewFromString("7922816251426433759354395.034"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("10"), NewFromString("7922816251426433759354395033.5"), false)
	verifyDivide(t, NewFromString("79228162514264337567774146561"), NewFromString("10"), NewFromString("7922816251426433756777414656.1"), false)
	verifyDivide(t, NewFromString("79228162514264337567774146560"), NewFromString("10"), NewFromString("7922816251426433756777414656"), false)
	verifyDivide(t, NewFromString("79228162514264337567774146559"), NewFromString("10"), NewFromString("7922816251426433756777414655.9"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.1"), NewFromString("72025602285694852357767227577"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.01"), NewFromString("78443725261647859003508861718"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.001"), NewFromString("79149013500763574019524425909.091"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0001"), NewFromString("79220240490215316061937756559.344"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00001"), NewFromString("79227370240561931974224208092.919"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000001"), NewFromString("79228083286181051412492537842.462"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0000001"), NewFromString("79228154591448878448656105469.389"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00000001"), NewFromString("79228161721982720373716746597.833"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000000001"), NewFromString("79228162435036175158507775176.492"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0000000001"), NewFromString("79228162506341521342909798200.709"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00000000001"), NewFromString("79228162513472055968409229775.316"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000000000001"), NewFromString("79228162514185109431029765225.569"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0000000000001"), NewFromString("79228162514256414777292524693.522"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00000000000001"), NewFromString("79228162514263545311918807699.547"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000000000000001"), NewFromString("79228162514264258365381436070.742"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0000000000000001"), NewFromString("79228162514264329670727698908.567"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00000000000000001"), NewFromString("79228162514264336801262325192.357"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000000000000000001"), NewFromString("79228162514264337514315787820.736"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0000000000000000001"), NewFromString("79228162514264337585621134083.574"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00000000000000000001"), NewFromString("79228162514264337592751668709.857"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000000000000000000001"), NewFromString("79228162514264337593464722172.486"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0000000000000000000001"), NewFromString("79228162514264337593536027518.749"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00000000000000000000001"), NewFromString("79228162514264337593543158053.375"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000000000000000000000001"), NewFromString("79228162514264337593543871106.837"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0000000000000000000000001"), NewFromString("79228162514264337593543942412.184"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00000000000000000000000001"), NewFromString("79228162514264337593543949542.718"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000000000000000000000000001"), NewFromString("79228162514264337593543950255.772"), false)
	verifyDivide(t, NewFromString("7922816251426433759354395033.5"), NewFromString("0.9999999999999999999999999999"), NewFromString("7922816251426433759354395034"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("10000000"), NewFromString("7922816251426433759354.3950335"), false)
	verifyDivide(t, NewFromString("7922816251426433759354395033.5"), NewFromString("1.000001"), NewFromString("7922808328618105141249253784.2"), false)
	verifyDivide(t, NewFromString("7922816251426433759354395033.5"), NewFromString("1.0000000000000000000000000001"), NewFromString("7922816251426433759354395032.7"), false)
	verifyDivide(t, NewFromString("7922816251426433759354395033.5"), NewFromString("1.0000000000000000000000000002"), NewFromString("7922816251426433759354395031.9"), false)
	verifyDivide(t, NewFromString("7922816251426433759354395033.5"), NewFromString("0.9999999999999999999999999999"), NewFromString("7922816251426433759354395034"), false)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0000000000000000000000000001"), NewFromString("79228162514264337593543950327"), false)
	boundary7 := Decimal{low: 429, mid: 2133437386, high: 0, flags: 0}
	boundary71 := Decimal{low: 429, mid: 2133437387, high: 0, flags: 0}
	maxValueBy7 := MaxValue.Mul(NewFromString("0.0000001"))
	verifyDivide(t, maxValueBy7, One, maxValueBy7, false)
	verifyDivide(t, maxValueBy7, One, maxValueBy7, false)
	verifyDivide(t, maxValueBy7, NewFromString("0.0000001"), MaxValue, false)
	verifyDivide(t, boundary7, One, boundary7, false)
	verifyDivide(t, boundary7, NewFromString("0.000000100000000000000000001"), NewFromString("91630438009337286849083695.62"), false)
	verifyDivide(t, boundary71, NewFromString("0.000000100000000000000000001"), NewFromString("91630438052286959809083695.62"), false)
	verifyDivide(t, NewFromString("7922816251426433759354.3950335"), NewFromString("1"), NewFromString("7922816251426433759354.3950335"), false)
	verifyDivide(t, NewFromString("7922816251426433759354.3950335"), NewFromString("0.0000001"), NewFromString("79228162514264337593543950335"), false)

	//[] DivideByZero exceptions
	verifyDivide(t, One, Zero, Zero, true)
	verifyDivide(t, Zero, Zero, Zero, true)
	verifyDivide(t, NewFromFloat32(-5.00), NewFromInt32(-1).Mul(Zero), Zero, true)
	verifyDivide(t, NewFromFloat32(0.0), NewFromFloat32(-0.00), Zero, true)

	//[] Overflow exceptions
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("-0.9999999999999999999999999"), Zero, true)
	verifyDivide(t, NewFromString("792281625142643.37593543950335"), NewFromString("0.0000000000000079228162514264337593543950335"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.1"), Zero, true)
	verifyDivide(t, NewFromString("7922816251426433759354395034"), NewFromString("0.1"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.9"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.99"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.9999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.99999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.9999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.99999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.9999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.99999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.9999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.99999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.999999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.9999999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.99999999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.999999999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.9999999999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.99999999999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.999999999999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.9999999999999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.99999999999999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.999999999999999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.9999999999999999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.99999999999999999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.999999999999999999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("0.9999999999999999999999999999"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("-0.1"), Zero, true)
	verifyDivide(t, NewFromString("79228162514264337593543950335"), NewFromString("-0.9999999999999999999999999"), Zero, true)
	verifyDivide(t, MaxValue.Div(NewFromInt32(2)), NewFromString("0.5"), Zero, true)

}

func TestFloor(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:345
	assert.True(t, NewFromString("123").Equal(NewFromString("123").Floor()))
	assert.True(t, NewFromString("123").Equal(NewFromString("123.123").Floor()))
	assert.True(t, NewFromString("-124").Equal(NewFromString("-123.123").Floor()))
	assert.True(t, NewFromString("123").Equal(NewFromString("123.567").Floor()))
	assert.True(t, NewFromString("-124").Equal(NewFromString("-123.567").Floor()))
}

func TestMaxValue(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:356
	assert.True(t, NewFromString("79228162514264337593543950335").Equal(MaxValue))
}

func TestMinusOne(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:364
	assert.True(t, NewFromString("-1").Equal(MinusOne))
}

func TestZero(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:370
	assert.True(t, NewFromString("0").Equal(Zero))
}

func TestOne(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:377
	assert.True(t, NewFromString("1").Equal(One))
}

func TestMinValue(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:384
	assert.True(t, NewFromString("-79228162514264337593543950335").Equal(MinValue))
}

func verifyMultiply(t *testing.T, d1, d2, expected Decimal, isPanic bool) {
	if isPanic {
		assert.Panics(t, func() { d1.Mul(d2) })
	} else {
		if assert.NotPanics(t, func() { d1.Mul(d2) }) {
			result1 := d1.Mul(d2)
			assert.True(t, result1.Equal(expected))
		}
	}
}

func TestMul(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:410
	verifyMultiply(t, One, One, One, false)
	verifyMultiply(t, NewFromString("7922816251426433759354395033.5"), NewFromString("10"), MaxValue, false)
	verifyMultiply(t, NewFromString("0.2352523523423422342354395033"), NewFromString("56033525474612414574574757495"), NewFromString("13182018677937129120135020796"), false)
	verifyMultiply(t, NewFromString("46161363632634613634.093453337"), NewFromString("461613636.32634613634083453337"), NewFromString("21308714924243214928823669051"), false)
	verifyMultiply(t, NewFromString("0.0000000000000345435353453563"), NewFromString("0.0000000000000023525235234234"), NewFromString("0.0000000000000000000000000001"), false)

	// Near MaxValue
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("0.9"), NewFromString("71305346262837903834189555302"), false)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("0.99"), NewFromString("78435880889121694217608510832"), false)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("0.9999999999999999999999999999"), NewFromString("79228162514264337593543950327"), false)
	verifyMultiply(t, NewFromString("-79228162514264337593543950335"), NewFromString("0.9"), NewFromString("-71305346262837903834189555302"), false)
	verifyMultiply(t, NewFromString("-79228162514264337593543950335"), NewFromString("0.99"), NewFromString("-78435880889121694217608510832"), false)
	verifyMultiply(t, NewFromString("-79228162514264337593543950335"), NewFromString("0.9999999999999999999999999999"), NewFromString("-79228162514264337593543950327"), false)

	// Exceptions
	verifyMultiply(t, MaxValue, MinValue, Zero, true)
	verifyMultiply(t, MinValue, NewFromString("1.1"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.1"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.01"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0000000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00000000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000000000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0000000000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00000000000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000000000000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0000000000000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00000000000000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000000000000000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0000000000000000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00000000000000000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000000000000000000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.0000000000000000000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.00000000000000000000000001"), Zero, true)
	verifyMultiply(t, NewFromString("79228162514264337593543950335"), NewFromString("1.000000000000000000000000001"), Zero, true)
	verifyMultiply(t, MaxValue.Div(NewFromInt32(2)), NewFromInt32(2), Zero, true)
}

func TestNegate(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:463
	assert.True(t, Zero.Neg().Equal(Zero))
	assert.True(t, MinusOne.Neg().Equal(One))
	assert.True(t, One.Neg().Equal(MinusOne))
}

func verifyRemainder(t *testing.T, d1, d2, expected Decimal, isPanic bool) {
	if isPanic {
		assert.Panics(t, func() { d1.Rem(d2) })
	} else {
		if assert.NotPanics(t, func() { d1.Rem(d2) }) {
			result1 := d1.Rem(d2)
			assert.True(t, result1.Equal(expected))
		}
	}
}

func TestRemainder(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:533
	negZero := Zero.Neg()
	verifyRemainder(t, NewFromString("5"), NewFromString("3"), NewFromString("2"), false)
	verifyRemainder(t, NewFromString("5"), NewFromString("-3"), NewFromString("2"), false)
	verifyRemainder(t, NewFromString("-5"), NewFromString("3"), NewFromString("-2"), false)
	verifyRemainder(t, NewFromString("-5"), NewFromString("-3"), NewFromString("-2"), false)
	verifyRemainder(t, NewFromString("3"), NewFromString("5"), NewFromString("3"), false)
	verifyRemainder(t, NewFromString("3"), NewFromString("-5"), NewFromString("3"), false)
	verifyRemainder(t, NewFromString("-3"), NewFromString("5"), NewFromString("-3"), false)
	verifyRemainder(t, NewFromString("-3"), NewFromString("-5"), NewFromString("-3"), false)
	verifyRemainder(t, NewFromString("10"), NewFromString("-3"), NewFromString("1"), false)
	verifyRemainder(t, NewFromString("-10"), NewFromString("3"), NewFromString("-1"), false)
	verifyRemainder(t, NewFromString("-2.0"), NewFromString("0.5"), NewFromString("-0.0"), false)
	verifyRemainder(t, NewFromString("2.3"), NewFromString("0.531"), NewFromString("0.176"), false)
	verifyRemainder(t, NewFromString("0.00123"), NewFromString("3242"), NewFromString("0.00123"), false)
	verifyRemainder(t, NewFromString("3242"), NewFromString("0.00123"), NewFromString("0.00044"), false)
	verifyRemainder(t, NewFromString("17.3"), NewFromString("3"), NewFromString("2.3"), false)
	verifyRemainder(t, NewFromString("8.55"), NewFromString("2.25"), NewFromString("1.8"), false)
	verifyRemainder(t, NewFromString("0.00"), NewFromString("3"), NewFromString("0.00"), false)
	verifyRemainder(t, negZero, NewFromString("2.2"), negZero, false)

	// Max / Min
	verifyRemainder(t, MaxValue, MaxValue, Zero, false)
	verifyRemainder(t, MaxValue, MinValue, Zero, false)
	verifyRemainder(t, MaxValue, One, Zero, false)
	verifyRemainder(t, MaxValue, NewFromString("2394713"), NewFromString("1494647"), false)
	verifyRemainder(t, MaxValue, NewFromString("-32768"), NewFromString("32767"), false)
	verifyRemainder(t, NewFromString("-0.00"), MaxValue, NewFromString("-0.00"), false)
	verifyRemainder(t, NewFromString("1.23984"), MaxValue, NewFromString("1.23984"), false)
	verifyRemainder(t, NewFromString("2398412.12983"), MaxValue, NewFromString("2398412.12983"), false)
	verifyRemainder(t, NewFromString("-0.12938"), MaxValue, NewFromString("-0.12938"), false)

	verifyRemainder(t, MinValue, MinValue, negZero, false)
	verifyRemainder(t, MinValue, MaxValue, negZero, false)
	verifyRemainder(t, MinValue, One, negZero, false)
	verifyRemainder(t, MinValue, NewFromString("2394713"), NewFromString("-1494647"), false)
	verifyRemainder(t, MinValue, NewFromString("-32768"), NewFromString("-32767"), false)
	verifyRemainder(t, NewFromString("0.0"), MinValue, NewFromString("0.0"), false)
	verifyRemainder(t, NewFromString("1.23984"), MinValue, NewFromString("1.23984"), false)
	verifyRemainder(t, NewFromString("2398412.12983"), MinValue, NewFromString("2398412.12983"), false)
	verifyRemainder(t, NewFromString("-0.12938"), MinValue, NewFromString("-0.12938"), false)

	verifyRemainder(t, NewFromString("57675350989891243676868034225"), NewFromString("7"), NewFromString("5"), false)
	verifyRemainder(t, NewFromString("-57675350989891243676868034225"), NewFromString("7"), NewFromString("-5"), false)
	verifyRemainder(t, NewFromString("57675350989891243676868034225"), NewFromString("-7"), NewFromString("5"), false)
	verifyRemainder(t, NewFromString("-57675350989891243676868034225"), NewFromString("-7"), NewFromString("-5"), false)

	verifyRemainder(t, NewFromString("792281625142643375935439503.4"), NewFromString("0.1"), NewFromString("0.0"), false)
	verifyRemainder(t, NewFromString("79228162514264337593543950.34"), NewFromString("0.1"), NewFromString("0.04"), false)
	verifyRemainder(t, NewFromString("7922816251426433759354395.034"), NewFromString("0.1"), NewFromString("0.034"), false)
	verifyRemainder(t, NewFromString("792281625142643375935439.5034"), NewFromString("0.1"), NewFromString("0.0034"), false)
	verifyRemainder(t, NewFromString("79228162514264337593543950335"), NewFromString("10"), NewFromString("5"), false)
	verifyRemainder(t, NewFromString("79228162514264337567774146561"), NewFromString("10"), NewFromString("1"), false)
	verifyRemainder(t, NewFromString("79228162514264337567774146560"), NewFromString("10"), NewFromString("0"), false)
	verifyRemainder(t, NewFromString("79228162514264337567774146559"), NewFromString("10"), NewFromString("9"), false)

}

func verifySubstract(t *testing.T, d1, d2, expected Decimal, isPanic bool) {
	if isPanic {
		assert.Panics(t, func() { d1.Sub(d2) })
	} else {
		if assert.NotPanics(t, func() { d1.Sub(d2) }) {
			result1 := d1.Sub(d2)
			assert.True(t, result1.Equal(expected))
		}
	}
}

func TestSub(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:614
	verifySubstract(t, One, One, Zero, false)
	verifySubstract(t, MinusOne, One, NewFromString("-2"), false)
	verifySubstract(t, One, MinusOne, NewFromString("2"), false)
	verifySubstract(t, MaxValue, Zero, MaxValue, false)
	verifySubstract(t, MinValue, Zero, MinValue, false)
	verifySubstract(t, NewFromString("79228162514264337593543950330"), NewFromString("-5"), MaxValue, false)
	verifySubstract(t, NewFromString("79228162514264337593543950330"), NewFromString("5"), NewFromString("79228162514264337593543950325"), false)
	verifySubstract(t, NewFromString("-79228162514264337593543950330"), NewFromString("5"), MinValue, false)
	verifySubstract(t, NewFromString("-79228162514264337593543950330"), NewFromString("-5"), NewFromString("-79228162514264337593543950325"), false)
	verifySubstract(t, NewFromString("1234.5678"), NewFromString("0.00009"), NewFromString("1234.56771"), false)
	verifySubstract(t, NewFromString("-1234.5678"), NewFromString("0.00009"), NewFromString("-1234.56789"), false)
	verifySubstract(t, NewFromString("0.1111111111111111111111111111"), NewFromString("0.1111111111111111111111111111"), NewFromString("0"), false)
	verifySubstract(t, NewFromString("0.2222222222222222222222222222"), NewFromString("0.1111111111111111111111111111"), NewFromString("0.1111111111111111111111111111"), false)
	verifySubstract(t, NewFromString("1.1111111111111111111111111110"), NewFromString("0.5555555555555555555555555555"), NewFromString("0.5555555555555555555555555555"), false)
}

func TestTruncate(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:640
	assert.True(t, NewFromString("123").Equal(NewFromString("123").Truncate()))
	assert.True(t, NewFromString("123").Equal(NewFromString("123.123").Truncate()))
	assert.True(t, NewFromString("-123").Equal(NewFromString("-123.123").Truncate()))
	assert.True(t, NewFromString("123").Equal(NewFromString("123.567").Truncate()))
	assert.True(t, NewFromString("-123").Equal(NewFromString("-123.567").Truncate()))
}

func TestRound(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:651
	// Not implement in DecimalTest-Microsoft.cs. Only commented out copy of TestTruncate function
}

func TestCompare(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:662
	assert.Equal(t, Zero.Cmp(Zero), 0)
	assert.Less(t, Zero.Cmp(One), 0)
	assert.Greater(t, One.Cmp(Zero), 0)
	assert.Less(t, MinusOne.Cmp(Zero), 0)
	assert.Greater(t, Zero.Cmp(MinusOne), 0)
	assert.Greater(t, NewFromString("5").Cmp(NewFromString("3")), 0)
	assert.Equal(t, NewFromString("5").Cmp(NewFromString("5")), 0)
	assert.Less(t, NewFromString("5").Cmp(NewFromString("9")), 0)
	assert.Less(t, NewFromString("-123.123").Cmp(NewFromString("123.123")), 0)
	assert.Equal(t, MaxValue.Cmp(MaxValue), 0)
	assert.Equal(t, MinValue.Cmp(MinValue), 0)
	assert.Less(t, MinValue.Cmp(MaxValue), 0)
	assert.Greater(t, MaxValue.Cmp(MinValue), 0)
}

func TestToFloat32(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:711
	var s float32 = 12345.12
	assert.Equal(t, s, NewFromFloat32(s).ToFloat32())
	assert.Equal(t, -s, NewFromFloat32(-s).ToFloat32())

	s = 1e20
	assert.Equal(t, s, NewFromFloat32(s).ToFloat32())
	assert.Equal(t, -s, NewFromFloat32(-s).ToFloat32())

	s = 1e27
	assert.Equal(t, s, NewFromFloat32(s).ToFloat32())
	assert.Equal(t, -s, NewFromFloat32(-s).ToFloat32())
}

func TestToFloat64(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:728
	var dbl float64 = 123456789.123456
	assert.Equal(t, dbl, NewFromFloat64(dbl).ToFloat64())
	assert.Equal(t, -dbl, NewFromFloat64(-dbl).ToFloat64())

	dbl = 1e20
	assert.Equal(t, dbl, NewFromFloat64(dbl).ToFloat64())
	assert.Equal(t, -dbl, NewFromFloat64(-dbl).ToFloat64())

	dbl = 1e27
	assert.Equal(t, dbl, NewFromFloat64(dbl).ToFloat64())
	assert.Equal(t, -dbl, NewFromFloat64(-dbl).ToFloat64())

	dbl = float64(math.MaxInt64)
	// C# note: Need to pass in the Int64.MaxValue to ToDouble and not dbl because the conversion to double is a little lossy and we want precision
	assert.Equal(t, dbl, NewFromInt64(math.MaxInt64).ToFloat64())
	assert.Equal(t, -dbl, NewFromInt64(-math.MaxInt64).ToFloat64())
}

func TestToInt32(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:760
	assert.Equal(t, int32(math.MaxInt32), NewFromInt32(math.MaxInt32).ToInt32())
	assert.Equal(t, int32(math.MinInt32), NewFromInt32(math.MinInt32).ToInt32())
}

func TestToInt64(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:774
	assert.Equal(t, int64(math.MaxInt64), NewFromInt64(math.MaxInt64).ToInt64())
	assert.Equal(t, int64(math.MinInt64), NewFromInt64(math.MinInt64).ToInt64())
}

func TestString(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:774
	d1 := NewFromString("6410.23")
	assert.Equal(t, "6410.23", d1.String())

	d2 := NewFromString("-8249.000003")
	assert.Equal(t, "-8249.000003", d2.String())

	assert.Equal(t, "79228162514264337593543950335", MaxValue.String())
	assert.Equal(t, "-79228162514264337593543950335", MinValue.String())
}

func TestNumberBufferLimit(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs.txt:870
	dE := NewFromString("1234567890123456789012345.6785")
	s1 := "1234567890123456789012345.678456"
	d1 := NewFromString(s1)
	assert.Equal(t, d1, dE)
}
