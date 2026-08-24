package decimal

import (
	"bytes"
	"fmt"
	"unicode/utf8"
)

// defaultPrecisionExponentialFormat is the precision "E" uses when none is given.
const defaultPrecisionExponentialFormat = 6

// perMilleUTF8 is U+2030 PER MILLE SIGN encoded as UTF-8. The custom-format
// scanner walks bytes, so the character has to be matched as its byte sequence.
const perMilleUTF8 = "\u2030"

// Format renders d according to a .NET-style format string, using [Invariant].
//
// The standard specifiers are C (currency), E (scientific), F (fixed-point),
// G (general), N (number with group separators), P (percent) and R
// (round-trip), each optionally followed by a precision of 0 to 99. Anything
// else is treated as a custom picture format.
//
// D and X are integral-only in .NET and report [ErrFormat] here.
func Format(d Decimal, format string) (string, error) {
	return FormatWith(d, format, Invariant)
}

// FormatWith renders d according to a .NET-style format string, using the
// symbols and patterns in nf. A nil nf means [Invariant].
func FormatWith(d Decimal, format string, nf *NumberFormat) (string, error) {
	nf = nf.orInvariant()

	fmtChar, digits := parseFormatSpecifier(format)

	number := new(buffer)
	decimalToNumber(d, number)

	sb := new(bytes.Buffer)
	if fmtChar != 0 {
		if err := numberToString(sb, number, fmtChar, digits, nf); err != nil {
			return "", err
		}
	} else {
		numberToStringFormat(sb, number, format, nf)
	}
	return sb.String(), nil
}

// MustFormat is [Format] with the error turned into a panic. It suits format
// strings fixed at compile time, where a failure is a programming error.
func MustFormat(d Decimal, format string) string {
	s, err := Format(d, format)
	if err != nil {
		panic(err)
	}
	return s
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

	i := int32(sb.Len())
	number.DigitsCount = i
	number.Scale = i - int32(d.scale())

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
				d := int32(format[1] - '0')
				if uint32(d) < 10 {
					digits = d
					return c, digits
				}
			} else if len(format) == 3 {
				// Fast path for symbol and double digit, e.g. "F12"
				d1 := int32(format[1] - '0')
				d2 := int32(format[2] - '0')
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

// numberToString renders a standard format specifier.
//
// FROM: Number.Formatting.cs NumberToString
func numberToString(sb *bytes.Buffer, number *buffer, format byte, maxDigits int32, nf *NumberFormat) error {
	isCorrectlyRounded := number.Kind == floatingPointKind

	switch format {
	case 'C', 'c':
		if maxDigits < 0 {
			maxDigits = int32(nf.CurrencyDecimalDigits)
		}
		// Rounding is relative to the current scale, not to digit position, so
		// this must not be rewritten in terms of digPos.
		roundNumber(number, int(number.Scale+maxDigits), isCorrectlyRounded)
		formatCurrency(sb, number, maxDigits, nf)

	case 'F', 'f':
		if maxDigits < 0 {
			maxDigits = int32(nf.NumberDecimalDigits)
		}
		roundNumber(number, int(number.Scale+maxDigits), isCorrectlyRounded)
		if number.IsNegative {
			sb.WriteString(nf.NegativeSign)
		}
		formatFixed(sb, number, maxDigits, nil, nf.NumberDecimalSeparator, "")

	case 'N', 'n':
		if maxDigits < 0 {
			maxDigits = int32(nf.NumberDecimalDigits)
		}
		roundNumber(number, int(number.Scale+maxDigits), isCorrectlyRounded)
		formatNumber(sb, number, maxDigits, nf)

	case 'E', 'e':
		if maxDigits < 0 {
			maxDigits = defaultPrecisionExponentialFormat
		}
		maxDigits++
		roundNumber(number, int(maxDigits), isCorrectlyRounded)
		if number.IsNegative {
			sb.WriteString(nf.NegativeSign)
		}
		formatScientific(sb, number, maxDigits, format, nf)

	case 'P', 'p':
		if maxDigits < 0 {
			maxDigits = int32(nf.PercentDecimalDigits)
		}
		// Percent is the number scaled by 100, so shift the decimal point two
		// places before rounding.
		number.Scale += 2
		roundNumber(number, int(number.Scale+maxDigits), isCorrectlyRounded)
		formatPercent(sb, number, maxDigits, nf)

	case 'G', 'g', 'R', 'r':
		// For a decimal, R and G agree: the value is exact, so round-tripping is
		// what G already does. (.NET Framework threw for R on decimal; .NET Core
		// does not, and the golden tables follow the current runtime.)
		expChar := byte('E')
		if format == 'g' || format == 'r' {
			expChar = 'e'
		}

		noRounding := false
		if maxDigits < 1 {
			if maxDigits == -1 {
				// Turn off rounding for ECMA compliance, so trailing zeros after
				// the decimal point stay significant.
				noRounding = true
				if number.Digits[0] == 0 {
					// -0 formats as 0 for decimal. RoundNumber would normally
					// handle that, and it is being skipped.
					goto skipSign
				}
				goto skipRounding
			}
			// Pad out to the correct place even at the default precision.
			maxDigits = number.DigitsCount
		}

		roundNumber(number, int(maxDigits), isCorrectlyRounded)

	skipRounding:
		if number.IsNegative {
			sb.WriteString(nf.NegativeSign)
		}

	skipSign:
		formatGeneral(sb, number, maxDigits, expChar, noRounding, nf)

	default:
		return fmt.Errorf("%w: %q is not valid for Decimal", ErrFormat, string(format))
	}
	return nil
}

// formatCurrency wraps the number in the culture's currency pattern.
func formatCurrency(sb *bytes.Buffer, number *buffer, maxDigits int32, nf *NumberFormat) {
	var p string
	if number.IsNegative {
		p = pattern(negCurrencyFormats[:], nf.CurrencyNegativePattern, 0)
	} else {
		p = pattern(posCurrencyFormats[:], nf.CurrencyPositivePattern, 0)
	}
	for _, ch := range p {
		switch ch {
		case '#':
			formatFixed(sb, number, maxDigits, nf.CurrencyGroupSizes,
				nf.CurrencyDecimalSeparator, nf.CurrencyGroupSeparator)
		case '-':
			sb.WriteString(nf.NegativeSign)
		case '$':
			sb.WriteString(nf.CurrencySymbol)
		default:
			sb.WriteRune(ch)
		}
	}
}

// formatNumber wraps the number in the culture's number pattern.
func formatNumber(sb *bytes.Buffer, number *buffer, maxDigits int32, nf *NumberFormat) {
	p := posNumberFormat
	if number.IsNegative {
		p = pattern(negNumberFormats[:], nf.NumberNegativePattern, 1)
	}
	for _, ch := range p {
		switch ch {
		case '#':
			formatFixed(sb, number, maxDigits, nf.NumberGroupSizes,
				nf.NumberDecimalSeparator, nf.NumberGroupSeparator)
		case '-':
			sb.WriteString(nf.NegativeSign)
		default:
			sb.WriteRune(ch)
		}
	}
}

// formatPercent wraps the number in the culture's percent pattern.
func formatPercent(sb *bytes.Buffer, number *buffer, maxDigits int32, nf *NumberFormat) {
	var p string
	if number.IsNegative {
		p = pattern(negPercentFormats[:], nf.PercentNegativePattern, 0)
	} else {
		p = pattern(posPercentFormats[:], nf.PercentPositivePattern, 0)
	}
	for _, ch := range p {
		switch ch {
		case '#':
			formatFixed(sb, number, maxDigits, nf.PercentGroupSizes,
				nf.PercentDecimalSeparator, nf.PercentGroupSeparator)
		case '-':
			sb.WriteString(nf.NegativeSign)
		case '%':
			sb.WriteString(nf.PercentSymbol)
		default:
			sb.WriteRune(ch)
		}
	}
}

// formatFixed writes the digits in fixed-point notation with exactly maxDigits
// fractional places, inserting group separators when groupSizes is non-nil.
//
// groupSizes holds the size of each group counting outward from the decimal
// point; the last entry repeats for the remaining groups, and a zero entry
// disables grouping from that point on.
//
// FROM: Number.Formatting.cs FormatFixed
func formatFixed(sb *bytes.Buffer, number *buffer, maxDigits int32, groupSizes []int, sDecimal, sGroup string) {
	digPos := number.Scale
	dig := 0 // index into number.Digits

	// digitAt yields the digit at i, or '0' once the buffer is exhausted.
	digitAt := func(i int) byte {
		if i < len(number.Digits) && number.Digits[i] != 0 {
			return number.Digits[i]
		}
		return '0'
	}

	if digPos > 0 {
		if groupSizes != nil {
			// Work out where the separators land, then emit right to left so each
			// group is measured from the decimal point outward.
			groupSize := 0
			if len(groupSizes) != 0 {
				groupSizeIndex := 0
				groupSizeCount := groupSizes[groupSizeIndex]
				for digPos > int32(groupSizeCount) {
					groupSize = groupSizes[groupSizeIndex]
					if groupSize == 0 {
						break
					}
					if groupSizeIndex < len(groupSizes)-1 {
						groupSizeIndex++
					}
					groupSizeCount += groupSizes[groupSizeIndex]
				}
				if groupSizeCount == 0 {
					// An array whose entries are all zero disables grouping.
					groupSize = 0
				} else {
					groupSize = groupSizes[0]
				}
			}

			digLength := number.DigitsCount
			digStart := digPos
			if digLength < digStart {
				digStart = digLength
			}

			out := make([]byte, 0, int(digPos)+len(sGroup)*int(digPos))
			groupSizeIndex := 0
			digitCount := 0
			for i := digPos - 1; i >= 0; i-- {
				if i < digStart {
					out = append(out, digitAt(int(i)))
				} else {
					out = append(out, '0')
				}

				if groupSize > 0 {
					digitCount++
					if digitCount == groupSize && i != 0 {
						for j := len(sGroup) - 1; j >= 0; j-- {
							out = append(out, sGroup[j])
						}
						if groupSizeIndex < len(groupSizes)-1 {
							groupSizeIndex++
							groupSize = groupSizes[groupSizeIndex]
						}
						digitCount = 0
					}
				}
			}
			for i := len(out) - 1; i >= 0; i-- {
				sb.WriteByte(out[i])
			}
			dig = int(digStart)
		} else {
			for {
				sb.WriteByte(digitAt(dig))
				if dig < len(number.Digits) && number.Digits[dig] != 0 {
					dig++
				}
				if digPos--; digPos <= 0 {
					break
				}
			}
		}
	} else {
		sb.WriteByte('0')
	}

	if maxDigits > 0 {
		sb.WriteString(sDecimal)
		if digPos < 0 {
			zeroes := -digPos
			if maxDigits < zeroes {
				zeroes = maxDigits
			}
			for i := int32(0); i < zeroes; i++ {
				sb.WriteByte('0')
			}
			// The reference also advances digPos here, but neither it nor this
			// port reads the value again, so the store is dropped.
			maxDigits -= zeroes
		}
		for ; maxDigits > 0; maxDigits-- {
			sb.WriteByte(digitAt(dig))
			if dig < len(number.Digits) && number.Digits[dig] != 0 {
				dig++
			}
		}
	}
}

// numberToStringFormat ...
func numberToStringFormat(sb *bytes.Buffer, number *buffer, format string, nf *NumberFormat) {

	var digitCount, decimalPos, firstDigit, lastDigit, digPos, thousandPos, thousandCount, scaleAdjust, adjust, section, src int32
	var scientific, thousandSeps bool
	dig := number.Digits
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
			// U+2030 is three bytes in UTF-8, so it has to be matched before the
			// byte-wise switch below can mistake its lead byte for something else.
			if hasPrefixAt(format, src, perMilleUTF8) {
				src += int32(len(perMilleUTF8))
				scaleAdjust += 3
				continue
			}
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
			case '\'', '"':
				// The reference advances past the closing quote as part of the
				// test (`pFormat[src++] != ch`). Stopping on the quote instead
				// leaves it to be read as the start of a second quoted run, which
				// swallows whatever follows it.
				for src < int32(len(format)) && format[src] != 0 {
					q := format[src]
					src++
					if q == ch {
						break
					}
				}
			case '\\':
				// An escaped character is a literal and must not be counted as a
				// digit placeholder. Without this case, `\#0` scans as one '#'
				// placeholder plus a '0' placeholder rather than a literal '#'
				// followed by one placeholder.
				if src < int32(len(format)) && format[src] != 0 {
					src++
				}
			case 'E', 'e':
				// The look-ahead is for the character '0', not for a NUL. Testing
				// against 0 means scientific notation is never detected, so "0E0"
				// renders as a literal E.
				if (src < int32(len(format)) && format[src] == '0') ||
					(src+1 < int32(len(format)) && (format[src] == '+' || format[src] == '-') && format[src+1] == '0') {
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
		if len(nf.NumberGroupSeparator) > 0 {
			// We need this array to figure out where to insert the thousands separator. We would have to traverse the string
			// backwards. PIC formatting always traverses forwards. These indices are precomputed to tell us where to insert
			// the thousands separator so we can get away with traversing forwards. Note we only have to compute up to digPos.
			// The max is not bound since you can have formatting strings of the form "000,000..", and this
			// should handle that case too.

			groupDigits := [1]int32{3} // internal int[] _numberGroupSizes = new int[] { 3 };
			var groupSizeIndex int32   // Index into the groupDigits array.
			var groupTotalSizeCount int32
			groupSizeLen := int32(len(groupDigits)) // The length of groupDigits array.
			if groupSizeLen != 0 {
				groupTotalSizeCount = groupDigits[groupSizeIndex] // The current running total of group size.
			}
			groupSize := groupTotalSizeCount
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
		sb.WriteString(nf.NegativeSign)
	}

	decimalWritten := false
	cur := 0
	for src < int32(len(format)) {
		if hasPrefixAt(format, src, perMilleUTF8) {
			src += int32(len(perMilleUTF8))
			sb.WriteString(perMilleUTF8)
			continue
		}
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
							sb.WriteString(nf.NumberGroupSeparator)
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
						sb.WriteString(nf.NumberGroupSeparator)
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
				sb.WriteString(nf.NumberDecimalSeparator)
				decimalWritten = true
			}
		case '%':
			sb.WriteString(nf.PercentSymbol)
		case ',':
			// Deliberately empty. A comma is a grouping directive, and it was
			// consumed entirely by the scanning pass above, which set
			// thousandSeps and thousandPos; the separators themselves are
			// emitted by the '#' and '0' cases from thousandSepPos. The
			// reference has `case ',': break;` here only because C# requires a
			// terminating statement in a case -- in Go that break is redundant,
			// and staticcheck flags it (SA4011).
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
				// The comparison is against the character '0', not against NUL --
				// the same mistranslation as in the scanning pass. Testing for 0
				// sends "0E0" down the literal path, which emits the E and drops
				// the exponent digits.
				if src < int32(len(format)) && format[src] == '0' {
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
				formatExponent(sb, exp, ch, i, positiveSign, nf)
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
		// The reference does sb.Insert(0, NegativeSign). Doing it by hand needs a
		// copy first: sb.Bytes() aliases the buffer's array, and Reset only sets
		// the length to zero, so writing the sign back overwrites the very bytes
		// still being read out.
		body := append([]byte(nil), sb.Bytes()...)
		sb.Reset()
		sb.WriteString(nf.NegativeSign)
		sb.Write(body)
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
func formatScientific(sb *bytes.Buffer, number *buffer, maxDigits int32, expChar byte, nf *NumberFormat) {
	var dig int
	if number.Digits[dig] != 0 {
		sb.WriteByte(number.Digits[dig])
		dig++
	} else {
		sb.WriteByte('0')
	}

	if maxDigits != 1 { //For E0 we would like to suppress the decimal point
		sb.WriteString(nf.NumberDecimalSeparator)
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
	formatExponent(sb, e, expChar, 3, true, nf)
}

// formatExponent ...
func formatExponent(sb *bytes.Buffer, value int32, expChar byte, minDigits int32, positiveSign bool, nf *NumberFormat) {
	// FROM: Number.Formatting.cs:2253
	sb.WriteByte(expChar)

	if value < 0 {
		sb.WriteString(nf.NegativeSign)
		value = -value
	} else if positiveSign {
		sb.WriteString(nf.PositiveSign)
	}

	digits := new(bytes.Buffer)
	uint32ToDecChars(digits, uint32(value), minDigits)
	sb.WriteString(reverseString(digits))
}

// formatGeneral ...
func formatGeneral(sb *bytes.Buffer, number *buffer, maxDigits int32, expChar byte, suppressScientific bool, nf *NumberFormat) {
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
		sb.WriteString(nf.NumberDecimalSeparator)

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
		formatExponent(sb, number.Scale-1, expChar, 2, true, nf)
	}
}

// roundNumber ...
func roundNumber(number *buffer, pos int, isCorrectlyRounded bool) {
	dig := number.Digits
	var i int
	for i < pos && i < len(dig) && dig[i] != 0 {
		i++
	}

	// The digit at the rounding position decides. The reference reads dig[i]
	// here; passing a zero-valued local instead means shouldRoundUp always takes
	// its "digit is 0" fast path and nothing is ever rounded up.
	var atPos byte
	if i < len(dig) {
		atPos = dig[i]
	}

	if i == pos && shouldRoundUp(atPos, isCorrectlyRounded) {
		for i > 0 && dig[i-1] == '9' {
			i--
		}
		if i > 0 {
			dig[i-1]++
		} else {
			number.Scale++
			dig[0] = '1'
			i = 1
		}
	} else {
		for i > 0 && dig[i-1] == '0' {
			i--
		}
	}

	if i == 0 {
		// Everything rounded away. The reference resets the scale here, and also
		// drops the sign so that -0.4 formatted to zero places is "0", not "-0".
		number.Scale = 0
		number.IsNegative = false
	}
	if i < len(dig) {
		dig[i] = 0
	}
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

// hasPrefixAt reports whether s contains prefix starting at byte offset i.
func hasPrefixAt(s string, i int32, prefix string) bool {
	return i >= 0 && int(i)+len(prefix) <= len(s) && s[i:int(i)+len(prefix)] == prefix
}
