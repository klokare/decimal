package decimal

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// String returns d in the general format: an optional minus sign, digits, and
// an optional decimal point with more digits. Trailing zeros are preserved, so
// MustParse("1.100").String() is "1.100".
//
// It matches .NET's decimal.ToString() under the invariant culture.
func (d Decimal) String() string {
	s, err := FormatWith(d, "G", Invariant)
	if err != nil {
		// "G" is always valid, so this cannot happen; report it rather than
		// silently returning "".
		return "%!decimal(" + err.Error() + ")"
	}
	return s
}

// Format implements [fmt.Formatter].
//
// The verbs follow Go's conventions rather than .NET's:
//
//	%v, %s   the general format, as String
//	%f, %F   fixed-point; the precision, or the value's own scale if none
//	%e, %E   scientific; the precision, or 6 if none
//	%g, %G   general; the precision limits significant digits
//	%+v      the underlying representation, for debugging
//
// Width, and the '-', '+', ' ' and '0' flags, work as they do for numbers.
func (d Decimal) Format(f fmt.State, verb rune) {
	if verb == 'v' && f.Flag('+') {
		b := d.Bits()
		_, _ = fmt.Fprintf(f, "decimal.Decimal{low:%d, mid:%d, high:%d, flags:%#08x}", b[0], b[1], b[2], b[3])
		return
	}

	prec, hasPrec := f.Precision()

	var spec string
	switch verb {
	case 'v', 's':
		spec = "G"
	case 'f', 'F':
		if !hasPrec {
			prec = int(d.Scale())
		}
		spec = "F" + strconv.Itoa(prec)
	case 'e', 'E':
		if !hasPrec {
			prec = 6
		}
		spec = string(verb) + strconv.Itoa(prec)
	case 'g', 'G':
		spec = "G"
		if hasPrec {
			spec += strconv.Itoa(prec)
		}
	default:
		_, _ = fmt.Fprintf(f, "%%!%c(decimal.Decimal=%s)", verb, d.String())
		return
	}

	s, err := FormatWith(d, spec, Invariant)
	if err != nil {
		_, _ = fmt.Fprintf(f, "%%!%c(decimal.Decimal=%v)", verb, err)
		return
	}

	// An explicit sign, when asked for and not already present.
	if !strings.HasPrefix(s, "-") {
		if f.Flag('+') {
			s = "+" + s
		} else if f.Flag(' ') {
			s = " " + s
		}
	}

	pad(f, s)
}

// pad writes s honouring the width and the '-' and '0' flags.
func pad(f fmt.State, s string) {
	width, hasWidth := f.Width()
	if !hasWidth || len(s) >= width {
		_, _ = f.Write([]byte(s))
		return
	}
	fill := strings.Repeat(" ", width-len(s))
	switch {
	case f.Flag('-'):
		_, _ = f.Write([]byte(s + fill))
	case f.Flag('0'):
		// Zero padding goes after any sign, not before it.
		zeros := strings.Repeat("0", width-len(s))
		if len(s) > 0 && (s[0] == '-' || s[0] == '+' || s[0] == ' ') {
			_, _ = f.Write([]byte(s[:1] + zeros + s[1:]))
			return
		}
		_, _ = f.Write([]byte(zeros + s))
	default:
		_, _ = f.Write([]byte(fill + s))
	}
}

// -- text --------------------------------------------------------------------

// MarshalText implements [encoding.TextMarshaler], producing the same text as
// [Decimal.String].
func (d Decimal) MarshalText() ([]byte, error) { return []byte(d.String()), nil }

// AppendText implements [encoding.TextAppender].
func (d Decimal) AppendText(b []byte) ([]byte, error) { return append(b, d.String()...), nil }

// UnmarshalText implements [encoding.TextUnmarshaler].
func (d *Decimal) UnmarshalText(text []byte) error {
	v, err := Parse(string(text))
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// -- json --------------------------------------------------------------------

// MarshalJSON implements [json.Marshaler].
//
// A Decimal is encoded as a JSON *string*, not as a bare number. A bare number
// would be read back by most JSON consumers -- JavaScript above all -- as an
// IEEE 754 double, which silently discards the precision this package exists to
// keep. Use [JSONNumber] where a bare number is required by an existing format.
//
// [Decimal.UnmarshalJSON] accepts either spelling.
func (d Decimal) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, 32)
	b = append(b, '"')
	b = append(b, d.String()...)
	b = append(b, '"')
	return b, nil
}

// UnmarshalJSON implements [json.Unmarshaler]. It accepts a JSON string or a
// JSON number, and treats null as leaving the value untouched.
func (d *Decimal) UnmarshalJSON(text []byte) error {
	s := strings.TrimSpace(string(text))
	if s == "null" {
		return nil
	}
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		unquoted, err := strconv.Unquote(s)
		if err != nil {
			return wrapf(ErrSyntax, "%s is not a JSON string", s)
		}
		s = unquoted
	}
	// JSON numbers permit an exponent, which the default Number style does not.
	v, err := ParseStyle(s, StyleFloat|AllowThousands, Invariant)
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// JSONNumber is a Decimal that encodes as a bare JSON number rather than as a
// string. Prefer [Decimal] unless an existing wire format requires the number
// form; see [Decimal.MarshalJSON] for why.
type JSONNumber Decimal

// Decimal returns n as a Decimal.
func (n JSONNumber) Decimal() Decimal { return Decimal(n) }

// MarshalJSON implements [json.Marshaler], writing a bare JSON number.
func (n JSONNumber) MarshalJSON() ([]byte, error) { return []byte(Decimal(n).String()), nil }

// UnmarshalJSON implements [json.Unmarshaler], accepting a number or a string.
func (n *JSONNumber) UnmarshalJSON(text []byte) error { return (*Decimal)(n).UnmarshalJSON(text) }

// String returns n in the general format.
func (n JSONNumber) String() string { return Decimal(n).String() }

// -- binary ------------------------------------------------------------------

// MarshalBinary implements [encoding.BinaryMarshaler].
//
// The encoding is 16 bytes, big-endian, ordered low, mid, high, flags. It is
// stable across releases and across architectures, and is what [encoding/gob]
// uses. For the layout .NET itself writes, see [Decimal.DotNetBytes].
func (d Decimal) MarshalBinary() ([]byte, error) {
	b := make([]byte, 16)
	d.putBinary(b)
	return b, nil
}

// AppendBinary implements [encoding.BinaryAppender].
func (d Decimal) AppendBinary(b []byte) ([]byte, error) {
	var buf [16]byte
	d.putBinary(buf[:])
	return append(b, buf[:]...), nil
}

func (d Decimal) putBinary(b []byte) {
	binary.BigEndian.PutUint32(b[0:4], d.low)
	binary.BigEndian.PutUint32(b[4:8], d.mid)
	binary.BigEndian.PutUint32(b[8:12], d.high)
	binary.BigEndian.PutUint32(b[12:16], d.flags)
}

// UnmarshalBinary implements [encoding.BinaryUnmarshaler]. It validates the
// flags word, reporting [ErrScaleRange] for a scale above 28 or any reserved bit
// set -- .NET performs the same check when deserialising.
func (d *Decimal) UnmarshalBinary(data []byte) error {
	if len(data) != 16 {
		return wrapf(ErrSyntax, "need exactly 16 bytes, got %d", len(data))
	}
	v, err := FromBits([4]uint32{
		binary.BigEndian.Uint32(data[0:4]),
		binary.BigEndian.Uint32(data[4:8]),
		binary.BigEndian.Uint32(data[8:12]),
		binary.BigEndian.Uint32(data[12:16]),
	})
	if err != nil {
		return err
	}
	*d = v
	return nil
}

// DotNetBytes returns the 16-byte layout .NET writes for a decimal: four
// little-endian 32-bit words ordered low, mid, high, flags. Use it to exchange
// values with a .NET process; use [Decimal.MarshalBinary] otherwise.
func (d Decimal) DotNetBytes() [16]byte {
	var b [16]byte
	binary.LittleEndian.PutUint32(b[0:4], d.low)
	binary.LittleEndian.PutUint32(b[4:8], d.mid)
	binary.LittleEndian.PutUint32(b[8:12], d.high)
	binary.LittleEndian.PutUint32(b[12:16], d.flags)
	return b
}

// FromDotNetBytes rebuilds a Decimal from the layout .NET writes. See
// [Decimal.DotNetBytes].
func FromDotNetBytes(b [16]byte) (Decimal, error) {
	return FromBits([4]uint32{
		binary.LittleEndian.Uint32(b[0:4]),
		binary.LittleEndian.Uint32(b[4:8]),
		binary.LittleEndian.Uint32(b[8:12]),
		binary.LittleEndian.Uint32(b[12:16]),
	})
}
