package decimal_test

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/klokare/decimal/v2"
)

func Example() {
	// An invoice line, computed exactly.
	price := decimal.MustParse("19.99")
	quantity := decimal.FromInt(3)
	taxRate := decimal.MustParse("0.0825")

	subtotal := price.Mul(quantity)
	tax := subtotal.Mul(taxRate).Round(2, decimal.AwayFromZero)
	total := subtotal.Add(tax)

	fmt.Println("subtotal:", subtotal)
	fmt.Println("tax:     ", tax)
	fmt.Println("total:   ", total)
	// Output:
	// subtotal: 59.97
	// tax:      4.95
	// total:    64.92
}

// Scale is part of a Decimal's value, so trailing zeros survive arithmetic.
func Example_scale() {
	a := decimal.MustParse("1.10")
	b := decimal.MustParse("2.20")

	fmt.Println(a.Add(b))         // not 3.3
	fmt.Println(a.Add(b).Scale()) // the scale says so too
	fmt.Println(decimal.MustParse("0.1").Add(decimal.MustParse("0.2")))

	// For contrast, the same sum in binary floating point.
	f1, f2 := 0.1, 0.2
	fmt.Println(f1 + f2)
	// Output:
	// 3.30
	// 2
	// 0.3
	// 0.30000000000000004
}

// Go's == compares the representation, not the value. Equal compares the value,
// and Normalize makes equal values share a representation.
func Example_equality() {
	a := decimal.MustParse("1.0")
	b := decimal.MustParse("1.00")

	fmt.Println("a == b:            ", a == b)
	fmt.Println("a.Equal(b):        ", a.Equal(b))
	fmt.Println("normalized equal:  ", a.Normalize() == b.Normalize())
	// Output:
	// a == b:             false
	// a.Equal(b):         true
	// normalized equal:   true
}

func ExampleParse() {
	d, err := decimal.Parse("1,234.56")
	fmt.Println(d, err)

	// Scientific notation needs a style that allows an exponent.
	_, err = decimal.Parse("1.5e3")
	fmt.Println(errors.Is(err, decimal.ErrSyntax))

	d, err = decimal.ParseStyle("1.5e3", decimal.StyleFloat, decimal.Invariant)
	fmt.Println(d, err)
	// Output:
	// 1234.56 <nil>
	// true
	// 1500 <nil>
}

func ExampleMustParse() {
	// Suits values fixed at compile time, where a failure is a programming error.
	var vatRate = decimal.MustParse("0.20")
	fmt.Println(vatRate)
	// Output: 0.20
}

func ExampleFormat() {
	d := decimal.MustParse("-1234567.891")

	for _, f := range []string{"G", "F2", "N2", "E3", "P1", "#,##0.00;(#,##0.00)"} {
		s, err := decimal.Format(d, f)
		if err != nil {
			fmt.Println(f, "->", err)
			continue
		}
		fmt.Printf("%-22s %s\n", f, s)
	}
	// Output:
	// G                      -1234567.891
	// F2                     -1234567.89
	// N2                     -1,234,567.89
	// E3                     -1.235E+006
	// P1                     -123,456,789.1 %
	// #,##0.00;(#,##0.00)    (1,234,567.89)
}

func ExampleFormatWith() {
	d := decimal.MustParse("1234567.89")

	usd, _ := decimal.FormatWith(d, "C2", decimal.EnUS)
	fmt.Println(usd)

	// A culture with European conventions.
	de := decimal.Invariant.Clone()
	de.CurrencySymbol = "€"
	de.CurrencyDecimalSeparator = ","
	de.CurrencyGroupSeparator = "."
	de.CurrencyPositivePattern = 3 // "n $"

	eur, _ := decimal.FormatWith(d, "C2", de)
	fmt.Println(eur)
	// Output:
	// $1,234,567.89
	// 1.234.567,89 €
}

func ExampleDecimal_Round() {
	d := decimal.MustParse("2.5")

	fmt.Println(d.Round(0, decimal.ToEven))       // banker's rounding
	fmt.Println(d.Round(0, decimal.AwayFromZero)) // what most people expect
	fmt.Println(d.Round(0, decimal.Floor))
	fmt.Println(d.Round(0, decimal.Ceiling))
	fmt.Println(d.Round(0, decimal.Truncate))
	// Output:
	// 2
	// 3
	// 2
	// 3
	// 2
}

func ExampleDecimal_TryDiv() {
	// The arithmetic methods panic so that expressions chain. The Try twins
	// return an error instead.
	_, err := decimal.One.TryDiv(decimal.Zero)
	fmt.Println(errors.Is(err, decimal.ErrDivideByZero))

	_, err = decimal.MaxValue.TryAdd(decimal.One)
	fmt.Println(errors.Is(err, decimal.ErrOverflow))

	// A panic carries the same sentinel, so recover can identify it.
	func() {
		defer func() {
			if r := recover(); r != nil {
				err, _ := r.(error)
				fmt.Println(errors.Is(err, decimal.ErrDivideByZero))
			}
		}()
		_ = decimal.One.Div(decimal.Zero)
	}()
	// Output:
	// true
	// true
	// true
}

func ExampleDecimal_MarshalJSON() {
	type invoice struct {
		Total decimal.Decimal    `json:"total"`
		Fee   decimal.JSONNumber `json:"fee"`
	}

	b, _ := json.Marshal(invoice{
		Total: decimal.MustParse("1234.50"),
		Fee:   decimal.JSONNumber(decimal.MustParse("2.99")),
	})
	fmt.Println(string(b))

	// A Decimal encodes as a string so no consumer can round-trip it through a
	// float64. JSONNumber opts into the bare-number form when a wire format
	// requires it. Both are accepted on the way back in.
	var back invoice
	_ = json.Unmarshal([]byte(`{"total":1234.50,"fee":"2.99"}`), &back)
	fmt.Println(back.Total, back.Fee)
	// Output:
	// {"total":"1234.50","fee":2.99}
	// 1234.50 2.99
}

func ExampleNullDecimal() {
	var n decimal.NullDecimal

	b, _ := json.Marshal(n)
	fmt.Println(string(b))

	n = decimal.NewNullDecimal(decimal.MustParse("42.50"))
	b, _ = json.Marshal(n)
	fmt.Println(string(b))

	v, _ := n.Value()
	fmt.Printf("%q\n", v)
	// Output:
	// null
	// "42.50"
	// "42.50"
}

// A Decimal implements driver.Valuer and sql.Scanner, so it works directly with
// a NUMERIC or DECIMAL column. Values are sent as text, so nothing is lost.
func ExampleDecimal_Scan() {
	var d decimal.Decimal

	// What a driver hands back for a NUMERIC column varies; all of these work.
	for _, src := range []any{"1234.50", []byte("1234.50"), int64(1234), 1234.5} {
		if err := d.Scan(src); err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Printf("%-20T %s\n", src, d)
	}

	// A NULL needs NullDecimal.
	err := d.Scan(nil)
	fmt.Println(errors.Is(err, decimal.ErrSyntax))

	var _ sql.Scanner = &d
	// Output:
	// string               1234.50
	// []uint8              1234.50
	// int64                1234
	// float64              1234.5
	// true
}

func ExampleDecimal_Bits() {
	// Bits exposes the representation: a 96-bit integer, a scale and a sign.
	d := decimal.MustParse("-1.100")
	b := d.Bits()

	fmt.Printf("coefficient %d, scale %d, negative %v\n", b[0], d.Scale(), d.IsNegative())
	fmt.Printf("flags %#08x\n", b[3])

	back, _ := decimal.FromBits(b)
	fmt.Println(back == d)
	// Output:
	// coefficient 1100, scale 3, negative true
	// flags 0x80030000
	// true
}

func ExampleFromInt() {
	// One constructor covers every signed width, including named types.
	type cents int64

	fmt.Println(decimal.FromInt(42))
	fmt.Println(decimal.FromInt(int8(-128)))
	fmt.Println(decimal.FromInt(cents(1999)))
	fmt.Println(decimal.FromUint(uint64(18446744073709551615)))
	// Output:
	// 42
	// -128
	// 1999
	// 18446744073709551615
}

func ExampleDecimal_Format() {
	d := decimal.MustParse("-1234.5678")

	fmt.Printf("%v\n", d)
	fmt.Printf("%.2f\n", d)
	fmt.Printf("%12.2f|\n", d)
	fmt.Printf("%-12.2f|\n", d)
	fmt.Printf("%.3e\n", d)
	// Output:
	// -1234.5678
	// -1234.57
	//     -1234.57|
	// -1234.57    |
	// -1.235e+003
}
