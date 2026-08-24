package decimal

import (
	"bytes"
	"database/sql/driver"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/klokare/decimal/v2/internal/assert"
)

// roundTripCorpus is the set of values every encoding must preserve exactly,
// including scale and the sign of zero.
var roundTripCorpus = []string{
	"0", "1", "-1", "1.0", "1.00", "1.100",
	"0.0000000000000000000000000001", "-0.0000000000000000000000000001",
	"79228162514264337593543950335", "-79228162514264337593543950335",
	"12345678901234567890.12345678", "0.5", "-0.5", "100.00",
}

func forEachCorpus(t *testing.T, fn func(t *testing.T, s string, d Decimal)) {
	t.Helper()
	for _, s := range roundTripCorpus {
		d := MustParse(s)
		t.Run(s, func(t *testing.T) { fn(t, s, d) })
	}
}

// Negative zero survives the binary encodings but not the textual ones, because
// String drops the sign of a zero exactly as .NET's ToString does. That is a
// deliberate asymmetry, so it is pinned here rather than left to be discovered.
func TestNegativeZeroEncoding(t *testing.T) {
	negZero := Zero.Neg()

	assert.Equal(t, "0", negZero.String(), "text drops the sign")

	var viaText Decimal
	assert.NoError(t, viaText.UnmarshalText([]byte(negZero.String())), "UnmarshalText")
	assert.True(t, viaText.Equal(negZero), "the value survives text")
	assert.False(t, viaText == negZero, "but the sign bit does not")

	b, err := negZero.MarshalBinary()
	assert.NoError(t, err, "MarshalBinary")
	var viaBinary Decimal
	assert.NoError(t, viaBinary.UnmarshalBinary(b), "UnmarshalBinary")
	assert.Equal(t, negZero, viaBinary, "binary preserves the sign bit")

	back, err := FromDotNetBytes(negZero.DotNetBytes())
	assert.NoError(t, err, "FromDotNetBytes")
	assert.Equal(t, negZero, back, ".NET bytes preserve the sign bit")
}

func TestTextRoundTrip(t *testing.T) {
	forEachCorpus(t, func(t *testing.T, s string, d Decimal) {
		b, err := d.MarshalText()
		assert.NoError(t, err, "MarshalText")

		var back Decimal
		assert.NoError(t, back.UnmarshalText(b), "UnmarshalText")
		assert.Equal(t, d, back, "text round-trip changed the representation")

		// AppendText must agree with MarshalText and respect the existing bytes.
		appended, err := d.AppendText([]byte("x:"))
		assert.NoError(t, err, "AppendText")
		assert.Equal(t, "x:"+string(b), string(appended), "AppendText")
	})

	// Parse(String(d)) is exact for every representable value, which is the
	// property the whole text layer rests on.
	rng := newTestRand(7)
	for i := 0; i < 20000; i++ {
		d := Random(rng)
		back, err := Parse(d.String())
		if err != nil {
			t.Fatalf("Parse(%q) from %s: %v", d.String(), formatBits(d), err)
		}
		// Equal, not ==: a negative zero comes back positive, since String drops
		// the sign. See TestNegativeZeroEncoding.
		if !back.Equal(d) {
			t.Fatalf("round-trip changed the value: %s -> %q -> %s",
				formatBits(d), d.String(), formatBits(back))
		}
	}
}

func TestBinaryRoundTrip(t *testing.T) {
	forEachCorpus(t, func(t *testing.T, s string, d Decimal) {
		b, err := d.MarshalBinary()
		assert.NoError(t, err, "MarshalBinary")
		assert.Equal(t, 16, len(b), "the encoding is always 16 bytes")

		var back Decimal
		assert.NoError(t, back.UnmarshalBinary(b), "UnmarshalBinary")
		assert.Equal(t, d, back, "binary round-trip changed the representation")

		appended, err := d.AppendBinary([]byte{0xAA})
		assert.NoError(t, err, "AppendBinary")
		assert.Equal(t, append([]byte{0xAA}, b...), appended, "AppendBinary")

		// And the .NET layout.
		dn := d.DotNetBytes()
		backDN, err := FromDotNetBytes(dn)
		assert.NoError(t, err, "FromDotNetBytes")
		assert.Equal(t, d, backDN, ".NET byte round-trip")
	})
}

func TestBinaryRejectsBadInput(t *testing.T) {
	var d Decimal
	assert.True(t, errors.Is(d.UnmarshalBinary(nil), ErrSyntax), "nil input")
	assert.True(t, errors.Is(d.UnmarshalBinary(make([]byte, 15)), ErrSyntax), "short input")
	assert.True(t, errors.Is(d.UnmarshalBinary(make([]byte, 17)), ErrSyntax), "long input")

	// A scale of 29 is not a legal Decimal, and must not be accepted silently.
	bad := make([]byte, 16)
	bad[12], bad[13], bad[14], bad[15] = 0x00, 0x1D, 0x00, 0x00 // big-endian flags = 0x001D0000
	err := d.UnmarshalBinary(bad)
	assert.True(t, errors.Is(err, ErrScaleRange), "scale 29 should be rejected, got %v", err)
}

// The .NET layout is little-endian and ordered low, mid, high, flags. Pin the
// exact bytes so the interop contract cannot drift.
func TestDotNetByteLayout(t *testing.T) {
	d := MustParse("1.100") // coefficient 1100 = 0x44C, scale 3
	got := d.DotNetBytes()
	want := [16]byte{
		0x4C, 0x04, 0x00, 0x00, // low, little-endian
		0x00, 0x00, 0x00, 0x00, // mid
		0x00, 0x00, 0x00, 0x00, // high
		0x00, 0x00, 0x03, 0x00, // flags: scale 3 in bits 16..23
	}
	assert.Equal(t, want, got, "DotNetBytes layout")
}

func TestGobRoundTrip(t *testing.T) {
	forEachCorpus(t, func(t *testing.T, s string, d Decimal) {
		var buf bytes.Buffer
		assert.NoError(t, gob.NewEncoder(&buf).Encode(d), "gob encode")

		var back Decimal
		assert.NoError(t, gob.NewDecoder(&buf).Decode(&back), "gob decode")
		assert.Equal(t, d, back, "gob round-trip changed the representation")
	})
}

func TestJSONIsAString(t *testing.T) {
	b, err := json.Marshal(MustParse("1.100"))
	assert.NoError(t, err, "Marshal")
	assert.Equal(t, `"1.100"`, string(b), "a Decimal encodes as a JSON string")

	// The reason: a bare number loses the value in any float64 consumer.
	big := MustParse("12345678901234567890.12345678")
	b, err = json.Marshal(big)
	assert.NoError(t, err, "Marshal")

	var back Decimal
	assert.NoError(t, json.Unmarshal(b, &back), "Unmarshal")
	assert.Equal(t, big, back, "JSON round-trip is exact")

	// Whereas via float64 it is not.
	var viaFloat float64
	assert.NoError(t, json.Unmarshal([]byte(big.String()), &viaFloat), "Unmarshal into float64")
	assert.False(t, fmt.Sprint(viaFloat) == big.String(), "float64 cannot hold it, for contrast")
}

func TestJSONAcceptsBothForms(t *testing.T) {
	for _, in := range []string{`"1.100"`, `1.100`, ` "1.100" `, `1.100e0`} {
		var d Decimal
		assert.NoError(t, json.Unmarshal([]byte(in), &d), "Unmarshal(%s)", in)
		assert.True(t, d.Equal(MustParse("1.1")), "Unmarshal(%s) gave %s", in, d)
	}

	// Scientific notation is accepted, since JSON numbers permit it.
	var d Decimal
	assert.NoError(t, json.Unmarshal([]byte(`1.5e3`), &d), "Unmarshal exponent")
	assert.Equal(t, "1500", d.String(), "1.5e3")

	// Malformed input is an error, not a silent zero.
	assert.True(t, errors.Is(json.Unmarshal([]byte(`"abc"`), &d), ErrSyntax), "garbage string")
	assert.True(t, errors.Is(json.Unmarshal([]byte(`"1.2.3"`), &d), ErrSyntax), "malformed number")
}

func TestJSONNumber(t *testing.T) {
	b, err := json.Marshal(JSONNumber(MustParse("1.100")))
	assert.NoError(t, err, "Marshal")
	assert.Equal(t, `1.100`, string(b), "JSONNumber encodes as a bare number")

	var n JSONNumber
	assert.NoError(t, json.Unmarshal([]byte(`1.100`), &n), "Unmarshal")
	assert.Equal(t, MustParse("1.100"), n.Decimal(), "JSONNumber round-trip")

	// Inside a struct, which is how it will actually be used.
	type payload struct {
		Amount JSONNumber `json:"amount"`
		Fee    Decimal    `json:"fee"`
	}
	b, err = json.Marshal(payload{Amount: JSONNumber(MustParse("9.99")), Fee: MustParse("0.30")})
	assert.NoError(t, err, "Marshal struct")
	assert.Equal(t, `{"amount":9.99,"fee":"0.30"}`, string(b), "struct encoding")
}

func TestJSONNull(t *testing.T) {
	// A null leaves a Decimal untouched, matching how the standard library
	// treats null for other scalar types.
	d := MustParse("5")
	assert.NoError(t, json.Unmarshal([]byte(`null`), &d), "null into Decimal")
	assert.Equal(t, "5", d.String(), "null should leave the value alone")
}

func TestNullDecimalJSON(t *testing.T) {
	b, err := json.Marshal(NullDecimal{})
	assert.NoError(t, err, "Marshal invalid")
	assert.Equal(t, "null", string(b), "an invalid NullDecimal is null")

	b, err = json.Marshal(NewNullDecimal(MustParse("1.100")))
	assert.NoError(t, err, "Marshal valid")
	assert.Equal(t, `"1.100"`, string(b), "a valid NullDecimal is the string form")

	var n NullDecimal
	assert.NoError(t, json.Unmarshal([]byte(`null`), &n), "Unmarshal null")
	assert.False(t, n.Valid, "null gives an invalid NullDecimal")
	assert.Equal(t, "NULL", n.String(), "String of an invalid NullDecimal")

	assert.NoError(t, json.Unmarshal([]byte(`"2.50"`), &n), "Unmarshal string")
	assert.True(t, n.Valid, "a value gives a valid NullDecimal")
	assert.Equal(t, "2.50", n.Decimal.String(), "the value survives")

	// A struct round-trip, which is where the embedded-vs-named field bug bit.
	type row struct {
		A NullDecimal `json:"a"`
		B NullDecimal `json:"b"`
	}
	b, err = json.Marshal(row{A: NewNullDecimal(MustParse("1.5"))})
	assert.NoError(t, err, "Marshal struct")
	assert.Equal(t, `{"a":"1.5","b":null}`, string(b), "struct encoding")

	var back row
	assert.NoError(t, json.Unmarshal(b, &back), "Unmarshal struct")
	assert.True(t, back.A.Valid, "a is valid")
	assert.False(t, back.B.Valid, "b is not")
	assert.Equal(t, "1.5", back.A.Decimal.String(), "a's value")
}

func TestSQLValue(t *testing.T) {
	forEachCorpus(t, func(t *testing.T, s string, d Decimal) {
		v, err := d.Value()
		assert.NoError(t, err, "Value")

		// A string, so no precision is lost on the way to a NUMERIC column.
		str, ok := v.(string)
		assert.True(t, ok, "Value should be a string, got %T", v)
		assert.Equal(t, d.String(), str, "Value")

		var back Decimal
		assert.NoError(t, back.Scan(str), "Scan back")
		assert.Equal(t, d, back, "Value/Scan round-trip")
	})
}

func TestSQLScanTypes(t *testing.T) {
	cases := []struct {
		src  any
		want string
	}{
		{"1.100", "1.100"},
		{[]byte("2.50"), "2.50"},
		{int64(42), "42"},
		{int32(-7), "-7"},
		{int(3), "3"},
		{uint64(18446744073709551615), "18446744073709551615"},
		{uint32(99), "99"},
		{float64(1.5), "1.5"},
		{float32(2.5), "2.5"},
		// Drivers differ in how they render a NUMERIC column.
		{"1.5e3", "1500"},
		{"1,234.56", "1234.56"},
	}
	for _, c := range cases {
		var d Decimal
		if err := d.Scan(c.src); err != nil {
			t.Errorf("Scan(%#v): %v", c.src, err)
			continue
		}
		assert.Equal(t, c.want, d.String(), "Scan(%#v)", c.src)
	}

	// NULL into a plain Decimal is an error; that is what NullDecimal is for.
	var d Decimal
	assert.True(t, errors.Is(d.Scan(nil), ErrSyntax), "NULL into a Decimal")
	assert.True(t, errors.Is(d.Scan(true), ErrSyntax), "a bool")
	assert.True(t, errors.Is(d.Scan(struct{}{}), ErrSyntax), "an unrelated type")
	assert.True(t, errors.Is(d.Scan("not a number"), ErrSyntax), "unparseable text")
}

func TestNullDecimalSQL(t *testing.T) {
	var n NullDecimal
	assert.NoError(t, n.Scan(nil), "Scan NULL")
	assert.False(t, n.Valid, "NULL gives an invalid NullDecimal")

	v, err := n.Value()
	assert.NoError(t, err, "Value of an invalid NullDecimal")
	assert.Equal(t, nil, v, "an invalid NullDecimal is NULL")

	assert.NoError(t, n.Scan("3.75"), "Scan a value")
	assert.True(t, n.Valid, "a value gives a valid NullDecimal")
	v, err = n.Value()
	assert.NoError(t, err, "Value")
	assert.Equal(t, "3.75", v, "Value of a valid NullDecimal")

	// A failed scan must leave it invalid rather than half-populated.
	assert.Error(t, n.Scan("nope"), "Scan garbage")
	assert.False(t, n.Valid, "a failed scan leaves it invalid")

	// The interfaces are satisfied by the value type, not just the pointer.
	var _ driver.Valuer = NullDecimal{}
	var _ driver.Valuer = Decimal{}
}

func TestFmtVerbs(t *testing.T) {
	d := MustParse("-1234.5678")
	cases := []struct {
		format string
		arg    any
		want   string
	}{
		{"%v", d, "-1234.5678"},
		{"%s", d, "-1234.5678"},
		{"%f", d, "-1234.5678"},
		{"%.2f", d, "-1234.57"},
		{"%.0f", d, "-1235"},
		{"%12.2f", d, "    -1234.57"},
		{"%-12.2f|", d, "-1234.57    |"},
		{"%012.2f", d, "-00001234.57"},
		{"%.3e", d, "-1.235e+003"},
		{"%.3E", d, "-1.235E+003"},
		{"%.6g", d, "-1234.57"},
		{"%d", d, "%!d(decimal.Decimal=-1234.5678)"},
		{"%v", MustParse("1.100"), "1.100"},
		{"%+v", One, "decimal.Decimal{low:1, mid:0, high:0, flags:0x00000000}"},
		{"%+.2f", MustParse("1.5"), "+1.50"},
		{"% .2f", MustParse("1.5"), " 1.50"},
		{"%8s|", MustParse("1.5"), "     1.5|"},
		{"%v", NewNullDecimal(MustParse("1.5")), "1.5"},
		{"%v", NullDecimal{}, "NULL"},
	}
	for _, c := range cases {
		if got := fmt.Sprintf(c.format, c.arg); got != c.want {
			t.Errorf("Sprintf(%q) = %q, want %q", c.format, got, c.want)
		}
	}
}
