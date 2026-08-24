package decimal

import "testing"

// The arithmetic paths must stay allocation-free; run with -benchmem.

var (
	benchDecimal Decimal
	benchString  string
	benchInt     int64
	benchErr     error
)

func BenchmarkAdd(b *testing.B) {
	x, y := MustParse("12345678901234567890.1234"), MustParse("98765.4321")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDecimal = x.Add(y)
	}
}

func BenchmarkSub(b *testing.B) {
	x, y := MustParse("12345678901234567890.1234"), MustParse("98765.4321")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDecimal = x.Sub(y)
	}
}

func BenchmarkMul(b *testing.B) {
	x, y := MustParse("123456789.123456789"), MustParse("987654321.987654321")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDecimal = x.Mul(y)
	}
}

func BenchmarkDiv(b *testing.B) {
	x, y := MustParse("79228162514264337593543950.335"), MustParse("3.14159265358979323846264338")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDecimal = x.Div(y)
	}
}

func BenchmarkMod(b *testing.B) {
	x, y := MustParse("45937986975432"), MustParse("43987453")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDecimal = x.Mod(y)
	}
}

func BenchmarkCmp(b *testing.B) {
	x, y := MustParse("1.0000000000000000000000000001"), MustParse("1.0000000000000000000000000002")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchInt = int64(x.Cmp(y))
	}
}

func BenchmarkRound(b *testing.B) {
	x := MustParse("5.5555555555555555555555555555")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDecimal = x.Round(4, ToEven)
	}
}

func BenchmarkParse(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDecimal, benchErr = Parse("12345678901234567890.1234")
	}
}

func BenchmarkParseSmall(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchDecimal, benchErr = Parse("1.25")
	}
}

func BenchmarkString(b *testing.B) {
	x := MustParse("12345678901234567890.1234")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchString = x.String()
	}
}

func BenchmarkFormatCurrency(b *testing.B) {
	x := MustParse("1234567.89")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchString = MustFormat(x, "C2")
	}
}

func BenchmarkInt64(b *testing.B) {
	x := MustParse("12345678901234.5678")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchInt = x.Int64()
	}
}

// A realistic sequence, for a single number that tracks overall cost.
func BenchmarkPipeline(b *testing.B) {
	price, qty, rate := MustParse("19.99"), MustParse("3"), MustParse("0.0825")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		subtotal := price.Mul(qty)
		tax := subtotal.Mul(rate).Round(2, AwayFromZero)
		benchDecimal = subtotal.Add(tax)
	}
}
