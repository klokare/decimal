package decimal

import (
	"bytes"
	"errors"
	"math"
	"unicode/utf8"
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

// Format the Decimal as a string.
func Format(d Decimal, format string) (destination string, err error) {
	// FROM Number.Formatting.cs: 326

	fmt, digits := parseFormatSpecifier(format)

	// Convert the decimal to a number
	number := new(buffer)
	decimalToNumber(d, number)

	// Convert number to string
	sb := new(bytes.Buffer)
	if fmt != 0 {
		numberToString(sb, number, fmt, digits)
	} else {
		numberToStringFormat(sb, number, format)
	}
	return sb.String(), nil
}

func decimalToNumber(d Decimal, number *buffer) {
	// FROM: Number.Formatting.cs: 350

	number.DigitsCount = decimalPrecision
	number.IsNegative = d.IsNegative()

	sb := new(bytes.Buffer)
	for (d.mid | d.high) != 0 {
		uint32ToDecChars(sb, decDivMod1E9(&d), 9)
	}
	uint32ToDecChars(sb, d.low, 0)

	var i int32 = int32(sb.Len())
	number.DigitsCount = i
	number.Scale = i - int32(d.Scale())

	number.Digits = make([]byte, i+1)
	copy(number.Digits, []byte(reverseString(sb)))
}

// parseFormatSpecifier ...
func parseFormatSpecifier(format string) (fmt byte, digits int32) {
	// FROM: Number.Formatting.cs:1516

	var c byte
	if len(format) > 0 {
		// If the format begins with a symbol, see if it's a standard format
		// with or without a specified number of digits.
		c = format[0]
		if (c-'A') <= 'Z'-'A' || (c-'a') <= 'z'-'a' {

			// Fast path for sole symbol, e.g. "D"
			if len(format) == 1 {
				digits = -1
				return c, digits
			}

			if len(format) == 2 {
				// Fast path for symbol and single digit, e.g. "X4"
				var d int32 = int32(format[1] - '0')
				if uint32(d) < 10 {
					digits = d
					return c, digits
				}
			} else if len(format) == 3 {
				// Fast path for symbol and double digit, e.g. "F12"
				var d1 int32 = int32(format[1] - '0')
				var d2 int32 = int32(format[2] - '0')
				if uint32(d1) < 10 && uint32(d2) < 10 {
					digits = d1*10 + d2
					return c, digits
				}
			}

			// Fallback for symbol and any length digits.  The digits value must be >= 0 && <= 99,
			// but it can begin with any number of 0s, and thus we may need to check more than two
			// digits.  Further, for compat, we need to stop when we hit a null char.
			var i int
			var n int32
			for i = 1; i < len(format) && uint32(format[i])-'0' < 10 && n < 10; i++ {
				n = (n * 10) + int32(format[i]-'0')
			}

			// If we're at the end of the digits rather than having stopped because we hit something
			// other than a digit or overflowed, return the standard format info.
			if i == len(format) || format[i] == 0 {
				digits = n
				return c, n
			}

		}
	}

	// Default empty format to be "G"; custom format is signified with '\0'.
	digits = -1
	if len(format) == 0 || c == 0 { // For compat, treat '\0' as the end of the specifier, even if the specifier extends beyond it.
		return 'G', digits
	}
	return 0, digits
}

// numberToString ...
func numberToString(sb *bytes.Buffer, number *buffer, format byte, maxDigits int32) {
	// FROM: Number.Formatting.cs:1582

	isCorrectlyRounded := number.Kind == floatingPointKind

	switch format {
	case 'G', 'g':
		// decimal
		noRounding := false
		if maxDigits < 1 {
			if maxDigits == -1 {
				noRounding = true // Turn off rounding for ECMA compliance to output trailing 0's after decimal as significant
				if number.Digits[0] == 0 {
					// -0 should be formatted as 0 for decimal. This is normally handled by RoundNumber (which we are skipping)
					goto SkipSign
				}
				goto SkipRounding
			} else {
				// This ensures that the PAL code pads out to the correct place even when we use the default precision
				maxDigits = number.DigitsCount
			}
		}

		roundNumber(number, int(maxDigits), false) // TODO: since this for decimal only, could round the decimal before converting it to a number.

	SkipRounding:
		if number.IsNegative {
			sb.WriteByte('-')
		}

	SkipSign:
		formatGeneral(sb, number, maxDigits, byte(format)-('G'-'E'), noRounding)

	case 'E', 'e':
		if maxDigits < 0 {
			maxDigits = 6 // private const int DefaultPrecisionExponentialFormat = 6;
		}
		maxDigits++

		roundNumber(number, int(maxDigits), isCorrectlyRounded)

		if number.IsNegative {
			sb.WriteByte('-') // info.NegativeSign
		}

		formatScientific(sb, number, maxDigits, format)

	// case 'C', 'c':
	// 	// currency
	// 	panic("Currency not implemented")
	// case 'F', 'f':
	// 	// fixed
	// 	panic("Fixed not implemented")
	// case 'N', 'n':
	// 	// number
	// 	panic("Number not implemented")
	// case 'P', 'p':
	// 	// percentage
	// 	panic("Percentage not implemented")
	// case 'R', 'r':
	// 	// floating point
	// 	panic("Floating point not implemented")
	default:
		panic("bad format specifier: " + string(format))
	}
}

// numberToStringFormat ...
func numberToStringFormat(sb *bytes.Buffer, number *buffer, format string) {

	var digitCount, decimalPos, firstDigit, lastDigit, digPos, thousandPos, thousandCount, scaleAdjust, adjust, section, src int32
	var scientific, thousandSeps bool
	var dig []byte = number.Digits
	var ch byte

	var tmp int32
	if dig[0] == 0 {
		tmp = 2
	} else {
		if number.IsNegative {
			tmp = 1
		} else {
			tmp = 0
		}
	}
	section = findSection(format, tmp)

	for {
		digitCount = 0
		decimalPos = -1
		firstDigit = 0x7FFFFFFF
		lastDigit = 0
		scientific = false
		thousandPos = -1
		thousandSeps = false
		scaleAdjust = 0
		src = section

		for src < int32(len(format)) {
			ch = format[src]
			src++
			if ch == 0 || ch == ';' {
				break
			}
			switch ch {
			case '#':
				digitCount++
			case '0':
				if firstDigit == 0x7FFFFFFF {
					firstDigit = digitCount
				}
				digitCount++
				lastDigit = digitCount
			case '.':
				if decimalPos < 0 {
					decimalPos = digitCount
				}
			case ',':
				if digitCount > 0 && decimalPos < 0 {
					if thousandPos >= 0 {
						if thousandPos == digitCount {
							thousandCount++
							break
						}
						thousandSeps = true
					}
					thousandPos = digitCount
					thousandCount = 1
				}
			case '%':
				scaleAdjust += 2
			// case '\x2030':
			// 	scaleAdjust += 3
			case '\'', '"':
				for src < int32(len(format)) && format[src] != 0 && format[src] != ch {
					src++
				}
			case 'E', 'e':
				if (src < int32(len(format)) && format[src] == 0) ||
					(src+1 < int32(len(format))) && (format[src] == '+' || format[src] == '-') && format[src+1] == '0' {
					src++
					for src < int32(len(format)) && format[src] == '0' {
						src++
					}
					scientific = true
				}

			}
		}

		if decimalPos < 0 {
			decimalPos = digitCount
		}

		if thousandPos >= 0 {
			if thousandPos == decimalPos {
				scaleAdjust -= thousandCount * 3
			} else {
				thousandSeps = true
			}
		}

		if dig[0] != 0 {
			number.Scale += scaleAdjust
			var pos int32
			if scientific {
				pos = digitCount
			} else {
				pos = number.Scale + digitCount - decimalPos
			}
			roundNumber(number, int(pos), false)
			dig = number.Digits
			if dig[0] == 0 {
				src = findSection(format, 2)
				if src != section {
					section = src
					continue
				}
			}
		} else {
			if number.Kind != floatingPointKind {
				// The integer types don't have a concept of -0 and decimal always format -0 as 0
				number.IsNegative = false
			}
			number.Scale = 0 // Decimals with scale ('0.00') should be rounded.
		}
		break
	}

	if firstDigit < decimalPos {
		firstDigit = decimalPos - firstDigit
	} else {
		firstDigit = 0
	}
	if lastDigit > decimalPos {
		lastDigit = decimalPos - lastDigit
	} else {
		lastDigit = 0
	}
	if scientific {
		digPos = decimalPos
		adjust = 0
	} else {
		if number.Scale > decimalPos {
			digPos = number.Scale
		} else {
			digPos = decimalPos
		}
		adjust = number.Scale - decimalPos
	}
	src = section

	// Adjust can be negative, so we make this an int instead of an unsigned int.
	// Adjust represents the number of characters over the formatting e.g. format string is "0000" and you are trying to
	// format 100000 (6 digits). Means adjust will be 2. On the other hand if you are trying to format 10 adjust will be
	// -2 and we'll need to fixup these digits with 0 padding if we have 0 formatting as in this example.
	thousandSepPos := make([]int32, 0, 4)
	var thousandsSepCtr int32 = -1
	if thousandSeps {
		// We need to precompute this outside the number formatting loop
		if true { // info.NumberGroupSeparator.Length > 0
			// We need this array to figure out where to insert the thousands separator. We would have to traverse the string
			// backwards. PIC formatting always traverses forwards. These indices are precomputed to tell us where to insert
			// the thousands separator so we can get away with traversing forwards. Note we only have to compute up to digPos.
			// The max is not bound since you can have formatting strings of the form "000,000..", and this
			// should handle that case too.

			groupDigits := [1]int32{3} // internal int[] _numberGroupSizes = new int[] { 3 };
			var groupSizeIndex int32   // Index into the groupDigits array.
			var groupTotalSizeCount int32
			var groupSizeLen int32 = int32(len(groupDigits)) // The length of groupDigits array.
			if groupSizeLen != 0 {
				groupTotalSizeCount = groupDigits[groupSizeIndex] // The current running total of group size.
			}
			var groupSize int32 = groupTotalSizeCount
			var totalDigits int32 // Actual number of digits in o/p
			if adjust < 0 {
				totalDigits = digPos + adjust
			} else {
				totalDigits = digPos + 0
			}
			var numDigits int32
			if firstDigit > totalDigits {
				numDigits = firstDigit
			} else {
				numDigits = totalDigits
			}

			for numDigits > groupTotalSizeCount {
				if groupSize == 0 {
					break
				}
				thousandsSepCtr++
				thousandSepPos = append(thousandSepPos, groupTotalSizeCount)
				if groupSizeIndex < groupSizeLen-2 {
					groupSizeIndex++
					groupSize = groupDigits[groupSizeIndex]
				}
				groupTotalSizeCount += groupSize
			}
		}
	}

	if number.IsNegative && section == 0 && number.Scale != 0 {
		sb.WriteByte('-') // info.NegativeSign
	}

	decimalWritten := false
	cur := 0
	for src < int32(len(format)) {
		ch = format[src]
		src++
		if ch == 0 || ch == ';' {
			break
		}

		if adjust > 0 {
			switch ch {
			case '#', '0', '.':
				for adjust > 0 {
					// digPos will be one greater than thousandsSepPos[thousandsSepCtr] since we are at
					// the character after which the groupSeparator needs to be appended.
					if dig[cur] != 0 {
						sb.WriteByte(dig[cur])
						cur++
					} else {
						sb.WriteByte('0')
					}
					if thousandSeps && digPos > 1 && thousandsSepCtr >= 0 {
						if digPos == thousandSepPos[thousandsSepCtr]+1 {
							sb.WriteByte(',') // info.NumberGroupSeparator
							thousandsSepCtr--
						}
					}
					digPos--
					adjust--
				}
			}
		}

		switch ch {
		case '#', '0':
			if adjust < 0 {
				adjust++
				if digPos <= firstDigit {
					ch = '0'
				} else {
					ch = 0
				}
			} else {
				if dig[cur] != 0 {
					ch = dig[cur]
					cur++
				} else {
					if digPos > lastDigit {
						ch = '0'
					} else {
						ch = 0
					}
				}
			}
			if ch != 0 {
				sb.WriteByte(ch)
				if thousandSeps && digPos > 1 && thousandsSepCtr >= 0 {
					if digPos == thousandSepPos[thousandsSepCtr]+1 {
						sb.WriteByte(',') // info.NumberGroupSeparator
						thousandsSepCtr--
					}
				}
			}
			digPos--
		case '.':
			if digPos != 0 || decimalWritten {
				// For compatibility, don't echo repeated decimals
				break
			}
			// If the format has trailing zeros or the format has a decimal and digits remain
			if lastDigit < 0 || (decimalPos < digitCount && dig[cur] != 0) {
				sb.WriteByte('.') // info.NumberDecimalSeparator
				decimalWritten = true
			}
		// case '\x2030':
		// 	sb.Append(info.PerMilleSymbol);
		// 	break;
		case '%':
			sb.WriteByte('%') // info.PercentSymbol
		case ',':
			break
		case '\'', '"':
			for src < int32(len(format)) && format[src] != 0 && format[src] != ch {
				sb.WriteByte(format[src])
				src++
			}
			if src < int32(len(format)) && format[src] != 0 {
				src++
			}
		case '\\':
			if src < int32(len(format)) && format[src] != 0 {
				sb.WriteByte(format[src])
				src++
			}
		case 'E', 'e':
			positiveSign := false
			var i int32 = 0
			if scientific {
				if src < int32(len(format)) && format[src] == 0 {
					// Handles E0, which should format the same as E-0
					i++
				} else if src+1 < int32(len(format)) && format[src] == '+' && format[src+1] == '0' {
					// Handles E+0
					positiveSign = true
				} else if src+1 < int32(len(format)) && format[src] == '-' && format[src+1] == '0' {
					// Handles E-0
					// Do nothing, this is just a place holder s.t. we don't break out of the loop.
				} else {
					sb.WriteByte(ch)
					break
				}

				for src++; src < int32(len(format)) && format[src] == '0'; src++ {
					i++
				}
				if i > 10 {
					i = 10
				}
				var exp int32
				if dig[0] == 0 {
					exp = 0
				} else {
					exp = number.Scale - decimalPos
				}
				formatExponent(sb, exp, ch, i, positiveSign)
				scientific = false
			} else {
				sb.WriteByte(ch) // Copy E or e to output
				if src < int32(len(format)) {
					if format[src] == '+' || format[src] == '-' {
						sb.WriteByte(format[src])
						src++
					}
					for src < int32(len(format)) && format[src] == '0' {
						sb.WriteByte(format[src])
						src++
					}
				}
			}
		default:
			sb.WriteByte(ch)
		}
	}

	if number.IsNegative && section == 0 && number.Scale == 0 && sb.Len() > 0 {
		tmp := sb.Bytes()
		sb.Reset()
		sb.WriteByte('-') // NegativeSign
		for _, b := range tmp {
			sb.WriteByte(b)
		}
	}
}

// findSection ...
func findSection(format string, section int32) int32 {
	if section == 0 {
		return 0
	}

	var src int32
	var ch byte
	for {
		if src >= int32(len(format)) {
			return 0
		}

		ch = format[src]
		src++
		switch ch {
		case '\'', '"':
			for src < int32(len(format)) && format[src] != 0 {
				tmp := format[src]
				src++
				if tmp == ch {
					break
				}
			}
		case '\\':
			if src < int32(len(format)) && format[src] != 0 {
				src++
			}
		case ';':
			if section--; section != 0 {
				break
			}
			if src < int32(len(format)) && format[src] != 0 && format[src] != ';' {
				return src
			}
			return 0
		case 0:
			return 0
		}
	}
}

// formatScientific ...
func formatScientific(sb *bytes.Buffer, number *buffer, maxDigits int32, expChar byte) {
	var dig int
	if number.Digits[dig] != 0 {
		sb.WriteByte(number.Digits[dig])
		dig++
	} else {
		sb.WriteByte('0')
	}

	if maxDigits != 1 { //For E0 we would like to suppress the decimal point
		sb.WriteByte('.') // info.NumberDecimalSeparator
	}

	for maxDigits--; maxDigits > 0; maxDigits-- {
		if number.Digits[dig] != 0 {
			sb.WriteByte(number.Digits[dig])
			dig++
		} else {
			sb.WriteByte('0')
		}
	}

	var e int32
	if number.Digits[0] == 0 {
		e = 0
	} else {
		e = number.Scale - 1
	}
	formatExponent(sb, e, expChar, 3, true)
}

// formatExponent ...
func formatExponent(sb *bytes.Buffer, value int32, expChar byte, minDigits int32, positiveSign bool) {
	// FROM: Number.Formatting.cs:2253
	sb.WriteByte(expChar)

	if value < 0 {
		sb.WriteByte('-') // info.NegativeSign
		value = -value
	} else {
		if positiveSign {
			sb.WriteByte('+') // info.PositiveSign
		}
	}

	digits := new(bytes.Buffer)
	uint32ToDecChars(digits, uint32(value), minDigits)
	sb.WriteString(reverseString(digits))
}

// formatGeneral ...
func formatGeneral(sb *bytes.Buffer, number *buffer, maxDigits int32, expChar byte, suppressScientific bool) {
	// FROM: Number.Formatting.cs:2273

	digPos := number.Scale
	scientific := false

	if !suppressScientific {
		// Don't switch to scientific notation
		if digPos > maxDigits || digPos < -3 {
			digPos = 1
			scientific = true
		}
	}

	dig := 0
	if digPos > 0 {
		for ok := true; ok; ok = digPos > 0 {
			if number.Digits[dig] != 0 {
				sb.WriteByte(number.Digits[dig])
				dig++
			} else {
				sb.WriteByte('0')
			}
			digPos--
		}
	} else {
		sb.WriteByte('0')
	}

	if number.Digits[dig] != 0 || digPos < 0 {
		sb.WriteByte('.') // info.NumberDecimalSeparator

		for digPos < 0 {
			sb.WriteByte('0')
			digPos++
		}

		for number.Digits[dig] != 0 {
			sb.WriteByte(number.Digits[dig])
			dig++
		}
	}

	if scientific {
		formatExponent(sb, number.Scale-1, expChar, 2, true)
	}
}

// roundNumber ...
func roundNumber(number *buffer, pos int, isCorrectlyRounded bool) {
	var dig byte
	var i int
	for i < pos && number.Digits[i] != 0 {
		i++
	}

	if i == pos && shouldRoundUp(dig, isCorrectlyRounded) {
		for i > 0 && number.Digits[i-1] == '9' {
			i--
		}
		if i > 0 {
			number.Digits[i-1]++
		} else {
			number.Scale++
			number.Digits[0] = '1'
			i = 1
		}
	} else {
		for i > 0 && number.Digits[i-1] == '0' {
			i--
		}
	}

	if i == 0 {
		number.Scale = 0
	}
	number.Digits[i] = 0
	number.DigitsCount = int32(i)
}

// shouldRoundUp ...
func shouldRoundUp(digit byte, isCorrectlyRounded bool) bool {
	// We only want to round up if the digit is greater than or equal to 5 and we are
	// not rounding a floating-point number. If we are rounding a floating-point number
	// we have one of two cases.
	//
	// In the case of a standard numeric-format specifier, the exact and correctly rounded
	// string will have been produced. In this scenario, pos will have pointed to the
	// terminating null for the buffer and so this will return false.
	//
	// However, in the case of a custom numeric-format specifier, we currently fall back
	// to generating Single/DoublePrecisionCustomFormat digits and then rely on this
	// function to round correctly instead. This can unfortunately lead to double-rounding
	// bugs but is the best we have right now due to back-compat concerns.

	if (digit == '0') || isCorrectlyRounded {
		// Fast path for the common case with no rounding
		return false
	}

	// Values greater than or equal to 5 should round up, otherwise we round down. The IEEE
	// 754 spec actually dictates that ties (exactly 5) should round to the nearest even number
	// but that can have undesired behavior for custom numeric format strings. This probably
	// needs further thought for .NET 5 so that we can be spec compliant and so that users
	// can get the desired rounding behavior for their needs.

	return digit >= '5'

}

// uint32ToDecChars ...
func uint32ToDecChars(sb *bytes.Buffer, value uint32, digits int32) {
	// FROM: Number.Formating.cs:1204

	var remainder uint32
	digits--
	for ; digits >= 0 || value != 0; digits-- {
		value, remainder = divRem(value, 10)
		sb.WriteByte(byte(remainder) + '0')
	}
}

// divRem ...
func divRem(a, b uint32) (div, rem uint32) {
	// FROM: Math.cs:204
	div = a / b
	rem = a - (div * b)
	return div, rem
}

type buffer struct {
	DigitsCount    int32
	Scale          int32
	Kind           bufferKind
	IsNegative     bool
	HasNonZeroTail bool
	Digits         []byte
}

type bufferKind byte

// Buffer Kinds
const (
	decimalKind bufferKind = iota // not the same value as the C# enum but this makes a Go zero-value buffer default to Decimal
	floatingPointKind
)

// reverseString ... based on https://stackoverflow.com/a/34521190
func reverseString(sb *bytes.Buffer) string {
	s := sb.String()
	size := len(s)
	buf := make([]byte, size)
	for start := 0; start < size; {
		r, n := utf8.DecodeRuneInString(s[start:])
		start += n
		utf8.EncodeRune(buf[size-start:], r)
	}
	return string(buf)
}
