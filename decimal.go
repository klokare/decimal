package decimal

import (
	"bytes"
	"database/sql/driver"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

// RoundingMode ...
type RoundingMode byte

// RoundingModes
const (
	ToEven RoundingMode = iota
	AwayFromZero
	Truncate
	Floor
	Ceiling
)

// Common Decimals
var (
	Zero            = Decimal{flags: 0, high: 0, low: 0, mid: 0}
	One             = Decimal{flags: 0, high: 0, low: 1, mid: 0}
	Two             = Decimal{flags: 0, high: 0, low: 2, mid: 0}
	MinusOne        = Decimal{flags: 2147483648, high: 0, low: 1, mid: 0}
	MaxValue        = Decimal{flags: 0, high: 4294967295, low: 4294967295, mid: 4294967295}
	MinValue        = Decimal{flags: 2147483648, high: 4294967295, low: 4294967295, mid: 4294967295}
	SmallestNonZero = Decimal{flags: 1835008, high: 0, low: 1, mid: 0}
)

// Decimal implements the public API of the internal decimal format
type Decimal struct {
	flags uint32
	high  uint32
	low   uint32
	mid   uint32
}

// low64 returns the low and mid words as a single 64-bit value.
func (d Decimal) low64() uint64 {
	return uint64(d.mid)<<32 | uint64(d.low)
}

// setLow64 splits a 64-bit value across the low and mid words.
func (d *Decimal) setLow64(value uint64) {
	d.mid = uint32(value >> 32)
	d.low = uint32(value)
}

// scale ...
func (d Decimal) scale() int { return int(byte(d.flags >> scaleShift)) }

// NewFromInt32 constructs a Decimal from an int32 value.
func NewFromInt32(value int32) Decimal {
	var d Decimal
	if value < 0 {
		d.flags = signMask
		value = -value
	}
	d.low = uint32(value)
	return d
}

// NewFromUint32 constructs a Decimal from an uint32 value.
func NewFromUint32(value uint32) Decimal {
	var d Decimal
	d.low = value
	return d
}

// NewFromInt64 constructs a Decimal from an int64 value.
func NewFromInt64(value int64) Decimal {
	var d Decimal
	if value < 0 {
		d.flags = signMask
		value = -value
	}
	d.low = uint32(value)
	d.mid = uint32(value >> 32)
	return d
}

// NewFromUint64 constructs a Decimal from an uint64 value.
func NewFromUint64(value uint64) Decimal {
	var d Decimal
	d.low = uint32(value)
	d.mid = uint32(value >> 32)
	return d
}

// NewFromFloat32 constructs a Decimal from an float32 value.
func NewFromFloat32(value float32) Decimal {
	var d Decimal
	varDecFromR4(value, &d)
	return d
}

// NewFromFloat64 constructs a Decimal from an float32 value.
func NewFromFloat64(value float64) Decimal {
	var d Decimal
	varDecFromR8(value, &d)
	return d
}

// NewFromString .... Panics if value is not a valid Decimal string
// To check for number format error, use ParseString
func NewFromString(value string) Decimal {
	d, err := Parse(value)
	if err != nil {
		panic(err)
	}
	return d
}

// Abs returns the absolute value of the given Decimal. If d is
// positive, the result is d. If d is negative, the result
// is -d.
func (d Decimal) Abs() Decimal {
	d.flags = d.flags & ^signMask
	return d
}

// Add the value to this Decimal.
func (d Decimal) Add(value Decimal) Decimal {
	decAddSub(&d, &value, false)
	return d
}

// Ceil rounds a Decimal to an integer value. The Decimal argument is rounded
// towards positive infinity.
// NOTE: c# method is Ceiling
func (d Decimal) Ceil() Decimal {
	if (d.flags & scaleMask) != 0 {
		internalRound(&d, uint32(byte(d.flags>>scaleShift)), Ceiling)
	}
	return d
}

// Clamp returns a value clamped to the inclusive range of min and max.
func (d Decimal) Clamp(min, max Decimal) Decimal {
	if d.LessThan(min) {
		d = min
	}
	if d.GreaterThan(max) {
		d = max
	}
	return d
}

// Cmp compares two Decimal values, return an intereger that indicates their relationship.
// NOTE: c# method is Compare
func (d Decimal) Cmp(value Decimal) int {
	return int(varDecCmp(d, value))
}

// Div divides two Decimal values.
// NOTE: c# method is Divide
func (d Decimal) Div(value Decimal) Decimal {
	varDecDiv(&d, &value)
	return d
}

// Equal returns true if both Decimals have equal value; othewise, false.
func (d Decimal) Equal(value Decimal) bool {
	return varDecCmp(d, value) == 0 // TODO: should this be a comparison of the two Decimals as structs. Go allows that.
}

// Floor rounds a Decimal to an integer value. The Decimal argument is rounded
// towards negative infinity.
func (d Decimal) Floor() Decimal {
	if (d.flags & scaleMask) != 0 {
		internalRound(&d, uint32(byte(d.flags>>scaleShift)), Floor)
	}
	return d
}

// Format the Decimal as a string.
func (d Decimal) Format(f fmt.State, c rune) {

	// Swap out Go `c` for Mono `c`
	switch c {
	case 'f', 's':
		c = 'g'
	case 'v':
		if f.Flag('+') {
			// Write the struct instead -- note the reverse ordering
			sb := bytes.NewBufferString("}")
			uint32ToDecChars(sb, d.flags, 1)
			sb.WriteString(" :sgalf ,")
			uint32ToDecChars(sb, d.high, 1)
			sb.WriteString(" :hgih ,")
			uint32ToDecChars(sb, d.mid, 1)
			sb.WriteString(" :dim ,")
			uint32ToDecChars(sb, d.low, 1)
			sb.WriteString(" :wol{lamiceD")
			f.Write([]byte(reverseString(sb)))
			return
		}
		c = 'g'
	}

	// Begin the format string
	sb := new(bytes.Buffer)

	// Construct the Mono-style format string from the state and rune

	if p, ok := f.Precision(); ok {
		if w, ok := f.Width(); ok {
			sb.WriteString(strings.Repeat("#", w-p))
		} else {
			sb.WriteByte('#')
		}
		sb.WriteByte('.')
		sb.WriteString(strings.Repeat("0", p))
	} else {
		sb.WriteByte(byte(c))
		if w, ok := f.Width(); ok {
			sb.WriteString(strconv.Itoa(w))
		}
	}

	// Format the decimal
	s, _ := Format(d, sb.String())
	f.Write([]byte(s))
}

// GreaterThan ...
func (d Decimal) GreaterThan(value Decimal) bool { return varDecCmp(d, value) > 0 }

// GreaterThanOrEqual ...
func (d Decimal) GreaterThanOrEqual(value Decimal) bool { return varDecCmp(d, value) >= 0 }

// LessThan ...
func (d Decimal) LessThan(value Decimal) bool { return varDecCmp(d, value) < 0 }

// LessThanOrEqual ...
func (d Decimal) LessThanOrEqual(value Decimal) bool { return varDecCmp(d, value) <= 0 }

// IsNegative returns true if the Decimal's value is less than zero.
func (d Decimal) IsNegative() bool { return int32(d.flags) < 0 }

// IsZero returns true if the Decimal's value is zero.
func (d Decimal) IsZero() bool { return (d.low | d.mid | d.high) == 0 }

// MarshalBinary encodes the receiver into a binary form and returns the result. The resulting
// encoding is in Big Endian form for portability. flags are placed in the last 4 bytes.
func (d Decimal) MarshalBinary() (data []byte, err error) {
	data = make([]byte, 16)
	err = d.MarshalBinaryTo(data)
	return data, err
}

func (d Decimal) MarshalBinaryTo(data []byte) error {
	if len(data) < 16 {
		return errors.New("marshal to byte array only allowed when length 16")
	}
	binary.BigEndian.PutUint32(data[0:4], d.high)
	binary.BigEndian.PutUint32(data[4:8], d.mid)
	binary.BigEndian.PutUint32(data[8:12], d.low)
	binary.BigEndian.PutUint32(data[12:16], d.flags)
	return nil
}

// MarshalJSON returns the decimal as a text string without quotes
func (d Decimal) MarshalJSON() ([]byte, error) { return d.MarshalText() }

// MarshalText encodes the receiver into UTF-8-encoded text and returns the result.
func (d Decimal) MarshalText() (text []byte, err error) {
	text = []byte(d.String())
	return text, nil
}

// Mul multiplies two Decimal values
func (d Decimal) Mul(value Decimal) Decimal {
	varDecMul(&d, &value)
	return d
}

// Neg returns the negated value of the given Decimal. If d is non-zero,
// the result is -d. If d is zero, the result is zero.
func (d Decimal) Neg() Decimal {
	d.flags ^= signMask
	return d
}

// Rem ...
func (d Decimal) Rem(value Decimal) Decimal {
	varDecMod(&d, &value)
	return d
}

// Round a Decimal value to a given number of decimal places. The value
// given by d is rounded to the number of decimal places given by
// decimals. The decimals argument must be an integer between
// 0 and 28 inclusive.
func (d Decimal) Round(decimals int32, mode RoundingMode) Decimal {
	if uint32(decimals) > 28 {
		panic("argument out of range exception")
	}
	if mode > AwayFromZero {
		panic("invalid rounding mode")
	}

	var scale int32 = int32(d.scale()) - decimals
	if scale > 0 {
		internalRound(&d, uint32(scale), mode)
	}
	return d
}

// Value provides a string value to the database.
func (d Decimal) Value() (driver.Value, error) {
	return d.String(), nil
}

// Scan assigns value from a database driver.
func (d *Decimal) Scan(src interface{}) error {
	switch t := src.(type) {
	case int32:
		*d = NewFromInt32(src.(int32))
		return nil
	case int64:
		*d = NewFromInt64(src.(int64))
		return nil
	case float32:
		*d = NewFromFloat32(src.(float32))
		return nil
	case float64:
		*d = NewFromFloat64(src.(float64))
		return nil
	case string:
		tmp, err := Parse(src.(string))
		*d = tmp
		return err
	case []byte:
		tmp, err := Parse(string(src.([]byte)))
		*d = tmp
		return err
	default:
		return fmt.Errorf("cannot create decimal from %v", t)
	}
}

// Sign returns an int that indicates the sign of the decimal.
func (d Decimal) Sign() int {
	if (d.low | d.mid | d.high) == 0 {
		return 0
	}
	return int((d.flags >> 31) | 1)
}

// String converts this Decimal to a string. The resulting string consists of an
// optional minus sign ("-") followed to a sequence of digits ("0" - "9"),
// optionally followed by a decimal point (".") and another sequence of
// digits.
func (d Decimal) String() string {
	// TODO: implement by calling Format with a specific format string
	// return fmt.Sprintf("%f", d)
	s, _ := Format(d, "G")
	return s
}

// Sub subtracts two Decimal values.
func (d Decimal) Sub(value Decimal) Decimal {
	decAddSub(&d, &value, true)
	return d
}

// ToInt ...
func (d Decimal) ToInt() int {
	if bits.UintSize == 64 {
		return int(d.ToInt64())
	}
	return int(d.ToInt32())
}

// ToUint ...
func (d Decimal) ToUint() uint {
	if bits.UintSize == 64 {
		return uint(d.ToUint64())
	}
	return uint(d.ToUint32())
}

// ToInt32 truncates the decimal to an int32 value. The Decimal argument is rounded
// towards zero to the nearest integer value, corresponding to removing all
// digits after the decimal point.
func (d Decimal) ToInt32() int32 {
	d = d.Truncate()
	if d.high|d.mid == 0 {
		i := int32(d.low)
		if !d.IsNegative() {
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
	panic("int32 overflow excpetion")
}

// ToUint32 converts a Decimal to an uint32. The Decimal
// value is rounded towards zero to the nearest integer value, and the
// result of this operation is returned as an uint32.
func (d Decimal) ToUint32() uint32 {
	d = d.Truncate()
	if d.high|d.mid == 0 {
		i := d.low
		if !d.IsNegative() || i == 0 {
			return i
		}
	}
	panic("uint32 overflow excpetion")
}

// ToInt64 converts a Decimal to an int64. The Decimal value is rounded towards zero
// to the nearest integer value, and the result of this operation is
// returned as a int64.
func (d Decimal) ToInt64() int64 {
	d = d.Truncate()
	if d.high == 0 {
		l := int64(d.low64())
		if !d.IsNegative() {
			if l >= 0 {
				return l
			}
		} else {
			l = -l
			if l <= 0 {
				return l
			}
		}
	}
	panic("int64 overflow excpetion")
}

// ToUint64 converts a Decimal to an uint64. The Decimal value is rounded towards zero
// to the nearest integer value, and the result of this operation is
// returned as a yint64.
func (d Decimal) ToUint64() uint64 {
	d = d.Truncate()
	if d.high == 0 {
		l := d.low64()
		if !d.IsNegative() || l == 0 {
			if l >= 0 {
				return l
			}
		}
	}
	panic("uint64 overflow excpetion")
}

// ToFloat32 converts a Decimal to a float32. Since a float32 has fewer significant
// digits than a Decimal, this operation may produce round-off errors.
func (d Decimal) ToFloat32() float32 { return varR4FromDec(d) }

// ToFloat64 converts a Decimal to a float64. Since a float64 has fewer significant
// digits than a Decimal, this operation may produce round-off errors.
func (d Decimal) ToFloat64() float64 { return varR8FromDec(d) }

// Truncate ...
func (d Decimal) Truncate() Decimal {
	if (d.flags & scaleMask) != 0 {
		internalRound(&d, uint32(byte(d.flags>>scaleShift)), Truncate)
	}
	return d
}

// UnmarshalBinary must be able to decode the form generated by MarshalBinary. UnmarshalBinary must
// copy the data if it wishes to retain the data after returning. The encoded data must be 16 bytes
// of 4x uint32 numbers in Big Endian form.
func (d *Decimal) UnmarshalBinary(data []byte) (err error) {
	if len(data) < 16 {
		return errors.New("insufficient data to unmarshal")
	}
	d.high = binary.BigEndian.Uint32(data[0:4])
	d.mid = binary.BigEndian.Uint32(data[4:8])
	d.low = binary.BigEndian.Uint32(data[8:12])
	d.flags = binary.BigEndian.Uint32(data[12:16])
	return nil
}

// UnmarshalJSON unmarshals the JSON value, ignoring quotes
func (d *Decimal) UnmarshalJSON(text []byte) error {
	return d.UnmarshalText(text)

}

// UnmarshalText unmarshals the decimal from the provided text.
func (d *Decimal) UnmarshalText(text []byte) (err error) {
	*d, err = Parse(string(text))
	return err
}

// -- package functions

// Random returns a random decimal value using the given generator.
func Random(rng *rand.Rand) Decimal {
	const mask = scaleMask | signMask
	return Decimal{
		flags: rng.Uint32() & mask,
		high:  rng.Uint32(),
		low:   rng.Uint32(),
		mid:   rng.Uint32(),
	}
}

// Max returns the greater value of `a` or `b`.
func Max(a, b Decimal) Decimal {
	if a.LessThan(b) {
		return b
	}
	return a
}

// MaxAny returns the greatest value in the list.
func MaxAny(any ...Decimal) Decimal {
	switch n := len(any); n {
	case 0:
		return Zero
	case 1:
		return any[0]
	default:
		max := any[0]
		for i := 1; i < n; i++ {
			if max.LessThan(any[i]) {
				max = any[i]
			}
		}
		return max
	}
}

// Min returns the lesser value of `a` or `b`.
func Min(a, b Decimal) Decimal {
	if a.GreaterThan(b) {
		return b
	}
	return a
}

// MinAny returns the least value in the list.
func MinAny(any ...Decimal) Decimal {
	switch n := len(any); n {
	case 0:
		return Zero
	case 1:
		return any[0]
	default:
		min := any[0]
		for i := 1; i < n; i++ {
			if min.GreaterThan(any[i]) {
				min = any[i]
			}
		}
		return min
	}
}

// Sum the list of values
func Sum(values ...Decimal) Decimal {
	var d Decimal
	for _, v := range values {
		d = d.Add(v)
	}
	return d
}

// Mean returns the mean (average) of the values
func Mean(values ...Decimal) Decimal {
	switch n := len(values); n {
	case 0:
		return Zero
	case 1:
		return values[0]
	default:
		den := NewFromInt64(int64(n))
		return Sum(values...).Div(den)
	}
}

// Median returns the median value from the list.
func Median(values ...Decimal) Decimal {
	switch n := len(values); n {
	case 0:
		return Zero
	case 1:
		return values[0]
	default:
		tmp := make([]Decimal, len(values))
		copy(tmp, values)
		sort.Slice(tmp, func(i, j int) bool { return tmp[i].LessThan(tmp[j]) })
		if n%2 == 0 {
			return tmp[n/2].Add(tmp[n/2+1]).Div(Two)
		}
		return tmp[n/2]
	}
}
