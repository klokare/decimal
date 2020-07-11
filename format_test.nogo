package decimal

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func SkipTestParse(t *testing.T) {
	// FROM: DecimalTest-Microsoft.cs:501

	var cases = []struct {
		S        string
		Expected Decimal
		IsError  bool
	}{
		{S: "0", Expected: Decimal{flags: 0, high: 0, low: 0, mid: 0}, IsError: false},
		{S: "123", Expected: Decimal{flags: 0, high: 0, low: 123, mid: 0}, IsError: false},
		{S: "-123", Expected: Decimal{flags: 2147483648, high: 0, low: 123, mid: 0}, IsError: false},
		{S: "123.123", Expected: Decimal{flags: 196608, high: 0, low: 123123, mid: 0}, IsError: false},
		{S: "-123.123", Expected: Decimal{flags: 2147680256, high: 0, low: 123123, mid: 0}, IsError: false},

		{S: "79228162514264337593543950335", Expected: Decimal{flags: 0, high: 4294967295, low: 4294967295, mid: 4294967295}, IsError: false},
		{S: "-79228162514264337593543950335", Expected: Decimal{flags: 2147483648, high: 4294967295, low: 4294967295, mid: 4294967295}, IsError: false},

		{S: "79,228,162,514,264,337,593,543,950,335", Expected: Decimal{flags: 0, high: 4294967295, low: 4294967295, mid: 4294967295}, IsError: false},

		{S: "ysaidufljasdf", Expected: Decimal{flags: 0, high: 0, low: 0, mid: 0}, IsError: true},
		{S: "79228162514264337593543950336", Expected: Decimal{flags: 0, high: 0, low: 0, mid: 0}, IsError: true},
	}

	for _, z := range cases {
		t.Run(z.S, func(t *testing.T) {
			actual, err := Parse(z.S)

			if !z.IsError {
				if err != nil {
					t.Errorf("parse should have succeeded: %s", err)
				}

				if z.Expected.flags != actual.flags {
					t.Errorf("incorrect flags: expected %d, actual %d", z.Expected.flags, actual.flags)
				}
				if z.Expected.high != actual.high {
					t.Errorf("incorrect high: expected %d, actual %d", z.Expected.high, actual.high)
				}
				if z.Expected.low != actual.low {
					t.Errorf("incorrect low: expected %d, actual %d", z.Expected.low, actual.low)
				}
				if z.Expected.mid != actual.mid {
					t.Errorf("incorrect mid: expected %d, actual %d", z.Expected.mid, actual.mid)
				}
			} else {
				if err == nil {
					t.Errorf("parse should have failed")
				}
			}
		})
	}
}

func SkipTestFormat(t *testing.T) {
	//d := NewFromString("12345.6789")
	rng := rand.New(rand.NewSource(42))
	d := Random(rng)
	// d := MaxValue
	// s1, _ := Format(d, "0.00")
	s1 := fmt.Sprintf("%e", d)
	//d = MaxValue
	s2 := fmt.Sprintf("%+v", d)
	if s1 != s2 {
		t.Errorf("incorrect format: expected %s, actual %s", s1, s2)
	}
}

func SkipTestFormatE(t *testing.T) {
	s := "5e2" // 500
	d := NewFromString(s)
	s2 := fmt.Sprintf("%e", d)
	assert.Equal(t, s, s2)
}
