package decimal

import (
	"encoding/json"
	"errors"
	"testing"
)

// The fuzz targets assert properties rather than specific values: nothing
// panics on hostile input, and the algebraic identities that must hold, do.

func FuzzParse(f *testing.F) {
	for _, s := range []string{
		"", " ", "0", "-0", "1.100", "1,234.56", "(1.5)", "1e5", "1E-28",
		"79228162514264337593543950335", "-79228162514264337593543950336",
		".", "-", "+", "1.2.3", "abc", "\x00", "1\x001",
		"0.0000000000000000000000000001", "1.00000000000000000000000000005",
	} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		// Every style must reject or accept without panicking.
		for _, style := range []Styles{StyleNone, StyleInteger, StyleNumber, StyleFloat, StyleCurrency, StyleAny} {
			for _, nf := range []*NumberFormat{Invariant, EnUS} {
				d, err := ParseStyle(s, style, nf)
				if err != nil {
					// A failed parse must not leave a value behind.
					if d != (Decimal{}) {
						t.Fatalf("ParseStyle(%q, %d) failed but returned %s", s, style, formatBits(d))
					}
					if !errors.Is(err, ErrSyntax) && !errors.Is(err, ErrOverflow) {
						t.Fatalf("ParseStyle(%q, %d) returned an unexpected error: %v", s, style, err)
					}
					continue
				}

				// Anything that parses must be a legal Decimal.
				if d.Scale() > 28 {
					t.Fatalf("ParseStyle(%q, %d) produced scale %d", s, style, d.Scale())
				}
				if d.flags&^(signMask|scaleMask) != 0 {
					t.Fatalf("ParseStyle(%q, %d) set reserved flag bits: %s", s, style, formatBits(d))
				}

				// And it must survive a text round-trip by value.
				back, err := Parse(d.String())
				if err != nil {
					t.Fatalf("Parse(%q.String() = %q) failed: %v", s, d.String(), err)
				}
				if !back.Equal(d) {
					t.Fatalf("round-trip changed %q: %s -> %q -> %s",
						s, formatBits(d), d.String(), formatBits(back))
				}
			}
		}
	})
}

func FuzzFormat(f *testing.F) {
	for _, format := range []string{
		"", "G", "G4", "E", "E30", "F", "F99", "N", "C", "P", "R",
		"#,##0.00", "0.0E+0", "0;(0);zero", `\#0`, "'lit'0", "#.##%",
		"D", "X", "Q", "\x00", "0000000000000000000000000000000000",
	} {
		f.Add(format, uint32(1), uint32(0), uint32(0), uint32(0))
	}

	f.Fuzz(func(t *testing.T, format string, low, mid, high, flags uint32) {
		// Only exercise legal representations; FromBits rejects the rest.
		d, err := FromBits([4]uint32{low, mid, high, flags})
		if err != nil {
			t.Skip()
		}

		for _, nf := range []*NumberFormat{Invariant, EnUS} {
			s, err := FormatWith(d, format, nf)
			if err != nil {
				if !errors.Is(err, ErrFormat) {
					t.Fatalf("FormatWith(%s, %q) returned an unexpected error: %v",
						formatBits(d), format, err)
				}
				if s != "" {
					t.Fatalf("FormatWith(%s, %q) failed but returned %q", formatBits(d), format, s)
				}
			}
		}
	})
}

func FuzzArithmetic(f *testing.F) {
	seeds := [][8]uint32{
		{1, 0, 0, 0, 1, 0, 0, 0},
		{0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0, 2, 0, 0, 0},
		{1, 0, 0, 28 << 16, 1, 0, 0, 0},
		{0, 0, 0, signMask, 1, 0, 0, 1 << 16},
	}
	for _, s := range seeds {
		f.Add(s[0], s[1], s[2], s[3], s[4], s[5], s[6], s[7])
	}

	f.Fuzz(func(t *testing.T, aLow, aMid, aHigh, aFlags, bLow, bMid, bHigh, bFlags uint32) {
		a, err := FromBits([4]uint32{aLow, aMid, aHigh, aFlags})
		if err != nil {
			t.Skip()
		}
		b, err := FromBits([4]uint32{bLow, bMid, bHigh, bFlags})
		if err != nil {
			t.Skip()
		}

		// Cmp is a consistent total order, and the predicates agree with it.
		c := a.Cmp(b)
		if c != -b.Cmp(a) {
			t.Fatalf("Cmp is not antisymmetric for %s and %s", formatBits(a), formatBits(b))
		}
		if (c == 0) != a.Equal(b) || (c < 0) != a.LessThan(b) || (c > 0) != a.GreaterThan(b) {
			t.Fatalf("the predicates disagree with Cmp for %s and %s", formatBits(a), formatBits(b))
		}

		// Abs and Neg are involutions on the magnitude.
		if !a.Neg().Neg().Equal(a) {
			t.Fatalf("Neg is not an involution for %s", formatBits(a))
		}
		if !a.Neg().Abs().Equal(a.Abs()) {
			t.Fatalf("Abs(Neg(x)) != Abs(x) for %s", formatBits(a))
		}

		// (a + b) - b == a whenever nothing overflowed. Rounding can lose the
		// low digits when the scales differ wildly, so compare after rounding
		// both to the smaller of the two scales.
		sum, err := a.TryAdd(b)
		if err != nil {
			return
		}
		back, err := sum.TrySub(b)
		if err != nil {
			return
		}
		places := int(a.Scale())
		if p := int(sum.Scale()); p < places {
			places = p
		}
		want, err1 := a.TryRound(places, ToEven)
		got, err2 := back.TryRound(places, ToEven)
		if err1 != nil || err2 != nil {
			return
		}
		if !got.Equal(want) {
			t.Fatalf("(a+b)-b != a at %d places: a=%s b=%s got=%s want=%s",
				places, formatBits(a), formatBits(b), formatBits(got), formatBits(want))
		}
	})
}

func FuzzJSON(f *testing.F) {
	for _, s := range []string{`"1.5"`, `1.5`, `null`, `"-0"`, `""`, `"abc"`, `1e5`, `[]`} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		var d Decimal
		if err := json.Unmarshal([]byte(s), &d); err != nil {
			return
		}
		// Anything that unmarshals must re-marshal and come back the same.
		b, err := json.Marshal(d)
		if err != nil {
			t.Fatalf("Marshal(%s) failed: %v", formatBits(d), err)
		}
		var back Decimal
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("Unmarshal(%s) failed: %v", b, err)
		}
		if !back.Equal(d) {
			t.Fatalf("JSON round-trip changed %s -> %s -> %s", s, b, formatBits(back))
		}
	})
}
