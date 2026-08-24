package decimal

import (
	"fmt"
	"math"
	"strings"
)

// decimalPrecision is the number of significant digits a Decimal can hold.
const decimalPrecision = 28

// parsePrecision is the working precision of the parser: one digit beyond what
// fits, so the 29th digit can drive the rounding decision.
const parsePrecision = 29

// Styles selects which textual conventions [ParseStyle] will accept. It mirrors
// .NET's NumberStyles, restricted to the flags meaningful for a Decimal --
// AllowHexSpecifier has no decimal analogue and is deliberately absent.
type Styles uint32

// Individual permissions, and the combinations .NET names.
const (
	// AllowLeadingWhite permits whitespace before the number.
	AllowLeadingWhite Styles = 1 << iota
	// AllowTrailingWhite permits whitespace after the number.
	AllowTrailingWhite
	// AllowLeadingSign permits a sign before the number.
	AllowLeadingSign
	// AllowTrailingSign permits a sign after the number.
	AllowTrailingSign
	// AllowParentheses permits parentheses around the number to mean negative.
	AllowParentheses
	// AllowDecimalPoint permits a decimal separator.
	AllowDecimalPoint
	// AllowThousands permits group separators among the integer digits.
	AllowThousands
	// AllowExponent permits scientific notation.
	AllowExponent
	// AllowCurrencySymbol permits the culture's currency symbol.
	AllowCurrencySymbol

	// None accepts digits only.
	None Styles = 0
	// Integer is AllowLeadingWhite | AllowTrailingWhite | AllowLeadingSign.
	Integer = AllowLeadingWhite | AllowTrailingWhite | AllowLeadingSign
	// Number adds a trailing sign, a decimal point and group separators.
	Number = Integer | AllowTrailingSign | AllowDecimalPoint | AllowThousands
	// Float adds an exponent but, like .NET, drops group separators.
	Float = AllowLeadingWhite | AllowTrailingWhite | AllowLeadingSign | AllowDecimalPoint | AllowExponent
	// Currency adds parentheses and the currency symbol.
	Currency = Number | AllowParentheses | AllowCurrencySymbol
	// Any accepts everything this package understands.
	Any = Currency | AllowExponent
)

// Parse converts a string to a Decimal using [Number] and [Invariant]. That
// accepts an optional sign, digits with optional group separators, and an
// optional fractional part -- but not scientific notation. Use [ParseStyle]
// with [Float] or [Any] for that.
func Parse(s string) (Decimal, error) {
	return ParseStyle(s, Number, Invariant)
}

// MustParse is [Parse] with the error turned into a panic. It suits literals
// fixed at compile time, where a failure is a programming error:
//
//	var rate = decimal.MustParse("0.0825")
func MustParse(s string) Decimal {
	d, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return d
}

// ParseStyle converts a string to a Decimal, accepting the conventions selected
// by style and described by nf. A nil nf means [Invariant].
func ParseStyle(s string, style Styles, nf *NumberFormat) (Decimal, error) {
	nf = nf.orInvariant()

	var b buffer
	b.Digits = make([]byte, 0, parsePrecision+2)
	if !parseString(s, &b, style, nf) {
		return Decimal{}, fmt.Errorf("%w: %q", ErrSyntax, s)
	}

	var d Decimal
	if !parseDecimal(&b, &d) {
		return Decimal{}, fmt.Errorf("%w: %q is out of range for a Decimal", ErrOverflow, s)
	}
	return d, nil
}

// parseString scans value into number, reporting whether the whole input was
// consumed as a well-formed number.
//
// FROM: Number.Parsing.cs ParseNumber
func parseString(value string, number *buffer, style Styles, nf *NumberFormat) bool {
	// The scan is a small state machine over the input. Each state records what
	// has already been seen so that a second sign, a second decimal point or a
	// group separator in the fraction can be rejected.
	const (
		stateSign = 1 << iota
		stateParens
		stateDigits
		stateNonZero
		stateDecimal
		stateCurrency
	)

	var state int
	var digitCount int
	var digitEnd int
	const digitMax = parsePrecision + 1

	// Separators are strings, not bytes: a culture may use a multi-byte one.
	decSep := nf.NumberDecimalSeparator
	grpSep := nf.NumberGroupSeparator
	if style&AllowCurrencySymbol != 0 {
		// .NET switches to the currency separators once a currency symbol is
		// permitted, and accepts either spelling.
		decSep = nf.CurrencyDecimalSeparator
		grpSep = nf.CurrencyGroupSeparator
	}

	i := 0
	n := len(value)

	skipWhite := func() {
		for i < n && isWhite(value[i]) {
			i++
		}
	}
	// Whitespace between the sign and the digits is only accepted once a currency
	// symbol has been seen, or when the culture's negative pattern is "- n"
	// (index 2). Otherwise "- 1" is not a number, which is what .NET reports.
	skipLeadingWhite := func() {
		if style&AllowLeadingWhite == 0 {
			return
		}
		if state&stateSign != 0 && state&stateCurrency == 0 && nf.NumberNegativePattern != 2 {
			return
		}
		skipWhite()
	}
	// eat consumes tok at the cursor if present.
	eat := func(tok string) bool {
		if tok != "" && strings.HasPrefix(value[i:], tok) {
			i += len(tok)
			return true
		}
		return false
	}

	skipLeadingWhite()

	// Leading sign, currency symbol and opening parenthesis may appear in either
	// order, and .NET accepts a currency symbol on either side of the sign.
	for i < n {
		switch {
		case style&AllowLeadingSign != 0 && state&stateSign == 0 && eat(nf.NegativeSign):
			state |= stateSign
			number.IsNegative = true
		case style&AllowLeadingSign != 0 && state&stateSign == 0 && eat(nf.PositiveSign):
			state |= stateSign
		case style&AllowParentheses != 0 && state&stateSign == 0 && value[i] == '(':
			i++
			state |= stateSign | stateParens
			number.IsNegative = true
		case style&AllowCurrencySymbol != 0 && state&stateCurrency == 0 && eat(nf.CurrencySymbol):
			state |= stateCurrency
		default:
			goto digits
		}
		skipLeadingWhite()
	}

digits:
	for i < n {
		c := value[i]
		if c >= '0' && c <= '9' {
			state |= stateDigits
			i++
			if c != '0' || state&stateNonZero != 0 {
				if digitCount < digitMax {
					number.Digits = append(number.Digits, c)
					digitCount++
					if c != '0' {
						digitEnd = digitCount
					}
				} else if c != '0' {
					// Beyond the digits that fit, only whether anything non-zero
					// followed still matters, for the midpoint rounding decision.
					number.HasNonZeroTail = true
				}
				if state&stateDecimal == 0 {
					number.Scale++
				}
				state |= stateNonZero
			} else if state&stateDecimal != 0 {
				number.Scale--
			}
			continue
		}

		if style&AllowDecimalPoint != 0 && state&stateDecimal == 0 && eat(decSep) {
			state |= stateDecimal
			continue
		}

		// A group separator is only meaningful among the integer digits.
		if style&AllowThousands != 0 && state&stateDigits != 0 && state&stateDecimal == 0 && eat(grpSep) {
			continue
		}

		break
	}

	if state&stateDigits == 0 {
		// No digits at all is not a number, however well-formed the decoration.
		return false
	}
	number.DigitsCount = int32(digitEnd)

	// Exponent.
	if style&AllowExponent != 0 && i < n && (value[i] == 'e' || value[i] == 'E') {
		save := i
		i++
		negExp := false
		if i < n {
			if strings.HasPrefix(value[i:], nf.NegativeSign) {
				negExp = true
				i += len(nf.NegativeSign)
			} else if strings.HasPrefix(value[i:], nf.PositiveSign) {
				i += len(nf.PositiveSign)
			}
		}
		if i < n && value[i] >= '0' && value[i] <= '9' {
			var exp int32
			for i < n && value[i] >= '0' && value[i] <= '9' {
				if exp < 1_000_000 { // saturate; anything this large is out of range anyway
					exp = exp*10 + int32(value[i]-'0')
				}
				i++
			}
			if negExp {
				exp = -exp
			}
			number.Scale += exp
		} else {
			// "1e" and "1e+" are not exponents. .NET rewinds and lets the trailing
			// checks reject the leftover text.
			i = save
		}
	}

	// Trailing sign, currency symbol and closing parenthesis.
	for i < n {
		if style&AllowTrailingWhite != 0 {
			skipWhite()
			if i >= n {
				break
			}
		}
		switch {
		case style&AllowTrailingSign != 0 && state&stateSign == 0 && eat(nf.NegativeSign):
			state |= stateSign
			number.IsNegative = true
		case style&AllowTrailingSign != 0 && state&stateSign == 0 && eat(nf.PositiveSign):
			state |= stateSign
		case style&AllowCurrencySymbol != 0 && state&stateCurrency == 0 && eat(nf.CurrencySymbol):
			state |= stateCurrency
		case state&stateParens != 0 && value[i] == ')':
			i++
			state &^= stateParens
		default:
			goto done
		}
	}

done:
	if style&AllowTrailingWhite != 0 {
		skipWhite()
	}

	// An opening parenthesis that was never closed is malformed.
	if state&stateParens != 0 {
		return false
	}

	// Anything left over means the input was not a number.
	if i != n {
		return false
	}

	// The scan loop reads digits until they no longer fit; parseDecimal expects a
	// trailing zero byte marking the end, as the C# buffer has.
	number.Digits = append(number.Digits, 0)
	return true
}

// isWhite reports whether c is whitespace for parsing purposes, matching the
// characters .NET's parser skips.
func isWhite(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	}
	return false
}

// parseDecimal converts a scanned digit buffer into a Decimal, reporting whether
// the value fits. number.Digits must end in a zero byte.
//
// FROM: Number.Parsing.cs TryParseDecimal
func parseDecimal(number *buffer, result *Decimal) bool {
	if len(number.Digits) == 0 {
		*result = Zero
		return true
	}

	var p int
	e := number.Scale
	sign := number.IsNegative
	c := number.Digits[p]

	if c == 0 {
		// All digits were zero, so the value is zero. Only the scale and sign
		// survive, and the scale is clamped into the representable range.
		e = -e
		if e < 0 {
			e = 0
		} else if e > decimalPrecision {
			e = decimalPrecision
		}
		flags := uint32(e) << scaleShift
		if sign {
			// This has to OR into flags: assigning would drop the scale, which is
			// observable as -0.00 losing its trailing zeros.
			flags |= signMask
		}
		*result = Decimal{flags: flags}
		return true
	}

	if e > parsePrecision {
		return false
	}

	var low64 uint64
	for e > -decimalPrecision {
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
	for (e > 0 || (c != 0 && e > -decimalPrecision)) &&
		(high < math.MaxUint32/10 ||
			(high == math.MaxUint32/10 &&
				(low64 < 0x9999999999999999 ||
					(low64 == 0x9999999999999999 && c <= '5')))) {

		// Multiply the 96-bit accumulator by ten.
		tmpLow := uint64(uint32(low64)) * 10
		tmp64 := uint64(uint32(low64>>32))*10 + (tmpLow >> 32)
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
		if c == '5' && low64&1 == 0 {
			p++
			c = number.Digits[p]

			hasZeroTail := !number.HasNonZeroTail

			// Digits may remain past the point where the accumulator filled up.
			// They count towards the tail: 3.0500000000000000000000000001e-27
			// is not a midpoint even though the digit at the boundary is 5.
			for c != 0 && hasZeroTail {
				hasZeroTail = c == '0'
				p++
				c = number.Digits[p]
			}

			if hasZeroTail {
				// Exactly a midpoint with an even result: banker's rounding leaves
				// it alone.
				goto noRounding
			}
		}

		if low64++; low64 == 0 {
			if high++; high == 0 {
				low64 = 0x999999999999999A
				high = math.MaxUint32 / 10
				e++
			}
		}
	}

noRounding:
	if e > 0 {
		return false
	}

	if e <= -parsePrecision {
		// A very small value, or a zero written with more scale than fits.
		*result = newDecimal(0, 0, 0, sign, parsePrecision-1)
	} else {
		*result = newDecimal(uint32(low64), uint32(low64>>32), high, sign, byte(-e))
	}
	return true
}

// newDecimal assembles a Decimal from its parts. scale must be 0 to 28.
func newDecimal(low, mid, high uint32, isNegative bool, scale byte) Decimal {
	if scale > decimalPrecision {
		panic(fmt.Errorf("%w: %d", ErrScaleRange, scale))
	}
	flags := uint32(scale) << scaleShift
	if isNegative {
		flags |= signMask
	}
	return Decimal{low: low, mid: mid, high: high, flags: flags}
}
