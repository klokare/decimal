package decimal

import (
	"errors"
	"math"
)

// decimalPrecision ...
const decimalPrecision = 28

func newDecimal(low, mid, high uint32, isNegative bool, scale byte) Decimal {
	if scale > 28 {
		panic("argument out of range exception")
	}

	var flags uint32
	flags = uint32(scale) << 16
	if isNegative {
		flags |= signMask
	}
	return Decimal{
		low:   low,
		mid:   mid,
		high:  high,
		flags: flags,
	}
}

// Parse ...
func Parse(s string) (d Decimal, err error) {
	// FROM: Number.Parsing.cs:1710

	// Begin a new buffer
	n := len(s) + 1
	if n > 29+1+1 {
		n = 29 + 1 + 1 // 29 for the longest input + 1 for rounding
	}
	b := &buffer{Digits: make([]byte, 0, n)}

	// Parse the string
	if !parseString(s, b) {
		err = errors.New("failed to parse string: " + s)
	}

	// Parse the number
	if !parseDecimal(b, &d) {
		err = errors.New("failed to parse decimal: " + s)
	}
	return
}

// TODO: only implemented Number for now: AllowLeadingWhite | AllowTrailingWhite | AllowLeadingSign | AllowTrailingSign | AllowDecimalPoint | AllowThousands
// TODO: only implemented '.' and ',' for decimal and group separator. Ignoring all other separators for now.
func parseString(value string, number *buffer) bool {
	// FROM. Number.Parsing.cs

	// End the number buffer with a zero byte.
	defer func() { number.Digits = append(number.Digits, 0) }()

	// Special case: empty string
	if len(value) == 0 {
		return true
	}

	const StateSign = 0x0001
	const StateParens = 0x0002
	const StateDigits = 0x0004
	const StateNonZero = 0x0008
	const StateDecimal = 0x0010
	const StateCurrency = 0x0020

	var state uint32
	var dcnt, p, dend int
	var dmax int = 30
	var c rune
	for p, c = range value {
		switch c {
		case '+':
			if (state & StateDecimal) != 0 {
				return false // sign needs to precede the digits
			}
			if (state & StateSign) != 0 {
				return false // already seen a sign in the string
			}
			state |= StateSign
		case '-':
			if (state & StateDecimal) != 0 {
				return false // sign needs to precede the digits
			}
			if (state & StateSign) != 0 {
				return false // already seen a sign in the string
			}
			state |= StateSign
			number.IsNegative = true
		case '.':
			if (state & StateDecimal) != 0 {
				return false // already have a decimal character
			}
			state |= StateDecimal
		case ',':
			// ignore group separator
		case 'e', 'E':
			goto Scientific
		case ' ', '\t', '\n':
			// ignore white space
		case '"', '\'':
			// ignore quotes
		default:
			if c >= '0' && c <= '9' {
				state |= StateDigits
				if c != '0' || (state&StateNonZero) != 0 {
					if dcnt < dmax {
						number.Digits = append(number.Digits, byte(c))
						dcnt++
						if c != '0' {
							dend = dcnt
						}

					} else if c != '0' {
						// For decimal and binary floating-point numbers, we only
						// need to store digits up to maxDigCount. However, we still
						// need to keep track of whether any additional digits past
						// maxDigCount were non-zero, as that can impact rounding
						// for an input that falls evenly between two representable
						// results.
						number.HasNonZeroTail = true
					}

					if (state & StateDecimal) == 0 {
						number.Scale++
					}
					state |= StateNonZero
				} else if (state & StateDecimal) != 0 {
					number.Scale--
				}
			} else {
				break // cannot parse this character for decimal
			}
		}
	}

Scientific:
	negExp := false
	number.DigitsCount = int32(dend)
	if (state & StateDigits) != 0 {
		if c == 'E' || c == 'e' {
			var exp int32
			for p, c = range value[p+1:] {
				switch c {
				case '+':
					// do nothing
				case '-':
					negExp = true
				default:
					if c >= '0' && c <= '9' {
						exp = exp*10 + (c - '0')
					} else {
						break // unknown character
					}
				}
			}
			if negExp {
				exp = -exp
			}
			number.Scale += exp
		}
	}
	return true
}

// NOTE: the `number`, if not empty, will be assumed to have a trailing zero byte to signify the end. This is so the following code matches the c# as closely as possible.
func parseDecimal(number *buffer, result *Decimal) bool {
	// FROM: Number.Parsing.cs:1570

	const DecimalPrecision int32 = 29 // FROM: Number.Formatting.cs:245

	// Special case: empty buffer
	if len(number.Digits) == 0 {
		*result = Zero
		return true
	}

	// Number has been filled, try to parse.
	var p int // index in the digit slice
	var e int32 = number.Scale
	var sign bool = number.IsNegative
	var c byte // a digit from the slice
	c = number.Digits[p]

	if c == 0 {
		// To avoid risking an app-compat issue with pre 4.5 (where some app was illegally using Reflection to examine the internal scale bits), we'll only force
		// the scale to 0 if the scale was previously positive (previously, such cases were unparsable to a bug.)
		// TODO: implement this clamping?
		var flags uint32
		e = -e
		if e < 0 {
			e = 0
		} else if e > 28 {
			e = 28
		}
		flags = uint32(e) << 16
		if sign {
			flags = signMask
		}
		*result = Decimal{flags: flags}
		return true
	}

	if e > DecimalPrecision {
		return false
	}

	var low64 uint64
	for e > -28 {
		e--
		low64 *= 10
		low64 += uint64(c - '0')
		p++
		c = number.Digits[p]
		if low64 >= math.MaxUint64/10 {
			break
		}
		if c == 0 {
			for e > 0 {
				e--
				low64 *= 10
				if low64 >= math.MaxUint64/10 {
					break
				}
			}
			break
		}
	}

	var high uint32
	for (e > 0 || (c != 0 && e > -28)) && (high < math.MaxUint32/10 || (high == math.MaxUint32/10 && (low64 < 0x99999999_99999999 || (low64 == 0x99999999_99999999 && c <= '5')))) {

		// Multiply by 10
		var tmpLow uint64 = uint64(uint32(low64)) * 10
		var tmp64 uint64 = uint64(uint32(low64>>32))*10 + (tmpLow >> 32)
		low64 = uint64(uint32(tmpLow)) + (tmp64 << 32)
		high = uint32(tmp64>>32) + high*10

		if c != 0 {
			c -= '0'
			low64 += uint64(c)
			if low64 < uint64(c) {
				high++
			}
			p++
			c = number.Digits[p]
		}
		e--
	}

	if c >= '5' {
		if (c == '5') && ((low64 & 1) == 0) {
			p++
			c = number.Digits[p]

			hasZeroTail := !number.HasNonZeroTail

			// We might still have some additional digits, in which case they need
			// to be considered as part of hasZeroTail. Some examples of this are:
			//  * 3.0500000000000000000001e-27
			//  * 3.05000000000000000000001e-27
			// In these cases, we will have processed 3 and 0, and ended on 5. The
			// buffer, however, will still contain a number of trailing zeros and
			// a trailing non-zero number.

			for (c != 0) && hasZeroTail {
				hasZeroTail = hasZeroTail && (c == '0')
				p++
				c = number.Digits[p]
			}

			// We should either be at the end of the stream or have a non-zero tail
			if hasZeroTail {
				// When the next digit is 5, the number is even, and all following
				// digits are zero we don't need to round.
				goto NoRounding
			}
		}

		if low64++; low64 == 0 {
			if high++; high == 0 {
				low64 = 0x99999999_9999999A
				high = math.MaxUint32 / 10
				e++
			}
		}
	}

NoRounding:
	if e > 0 {
		return false
	}

	if e <= -DecimalPrecision {
		// Parsing a large scale zero can give you more precision than fits in the decimal.
		// This should only happen for actual zeros or very small numbers that round to zero.
		*result = newDecimal(0, 0, 0, sign, byte(DecimalPrecision-1))
	} else {
		*result = newDecimal(uint32(low64), uint32(low64>>32), high, sign, byte(-e))
	}
	return true
}
