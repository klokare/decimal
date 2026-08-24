package decimal

import "testing"

// These exercise the arithmetic core against tables generated from .NET. Unlike
// the v1 suite they compare representations, so a result that is numerically
// right but carries the wrong scale -- or a zero with the wrong sign -- fails.

func TestGoldenBinary(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func(a, b Decimal) Decimal
	}{
		{"add", func(a, b Decimal) Decimal { return a.Add(b) }},
		{"sub", func(a, b Decimal) Decimal { return a.Sub(b) }},
		{"mul", func(a, b Decimal) Decimal { return a.Mul(b) }},
		{"div", func(a, b Decimal) Decimal { return a.Div(b) }},
		{"rem", func(a, b Decimal) Decimal { return a.Mod(b) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			golden(t, tc.name, func(t *testing.T, r row) {
				a, b := r.dec(t, 1), r.dec(t, 2)
				var got Decimal
				rec, panicked := wantPanic(func() { got = tc.fn(a, b) })
				checkDecimal(t, r, got, panicked, rec)
			})
		})
	}
}

func TestGoldenUnary(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func(d Decimal) Decimal
	}{
		{"abs", Decimal.Abs},
		{"neg", Decimal.Neg},
		{"floor", Decimal.Floor},
		{"ceil", Decimal.Ceil},
		{"truncate", Decimal.Truncate},
	} {
		t.Run(tc.name, func(t *testing.T) {
			golden(t, tc.name, func(t *testing.T, r row) {
				a := r.dec(t, 1)
				var got Decimal
				rec, panicked := wantPanic(func() { got = tc.fn(a) })
				checkDecimal(t, r, got, panicked, rec)
			})
		})
	}
}

func TestGoldenRound(t *testing.T) {
	golden(t, "round", func(t *testing.T, r row) {
		a := r.dec(t, 1)
		places := r.num(t, 2)
		var mode RoundingMode
		switch r.fields[3] {
		case "toeven":
			mode = ToEven
		case "awayfromzero":
			mode = AwayFromZero
		default:
			t.Fatalf("%s: unknown rounding mode %q", r, r.fields[3])
		}
		var got Decimal
		rec, panicked := wantPanic(func() { got = a.Round(places, mode) })
		checkDecimal(t, r, got, panicked, rec)
	})
}

func TestGoldenCompare(t *testing.T) {
	golden(t, "compare", func(t *testing.T, r row) {
		a, b := r.dec(t, 1), r.dec(t, 2)

		want := r.str(t, 0)
		got := joinPredicates(a, b)
		if got != want {
			t.Errorf("%s: predicates disagree\n  want %s\n   got %s\n  (sign,==,!=,<,<=,>,>=,Equals)\n  a=%s b=%s",
				r, want, got, a.String(), b.String())
		}
	})
}

func joinPredicates(a, b Decimal) string {
	bit := func(v bool) string {
		if v {
			return "1"
		}
		return "0"
	}
	sign := "0"
	switch c := a.Cmp(b); {
	case c < 0:
		sign = "-1"
	case c > 0:
		sign = "1"
	}
	return sign + "," +
		bit(a.Equal(b)) + "," +
		bit(!a.Equal(b)) + "," +
		bit(a.LessThan(b)) + "," +
		bit(a.LessThanOrEqual(b)) + "," +
		bit(a.GreaterThan(b)) + "," +
		bit(a.GreaterThanOrEqual(b)) + "," +
		bit(a.Equal(b))
}
