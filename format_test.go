package decimal

import (
	"strconv"
	"testing"
)

// nfFor maps a culture name in the golden tables to its NumberFormat.
func nfFor(t *testing.T, name string) *NumberFormat {
	t.Helper()
	switch name {
	case "invariant":
		return Invariant
	case "en-US":
		return EnUS
	default:
		t.Fatalf("no NumberFormat for culture %q", name)
		return nil
	}
}

// checkString compares a produced string against the golden row, honouring the
// recorded outcome.
func checkString(t *testing.T, r row, got string, err error, panicked bool, rec any, ctx string) {
	t.Helper()
	if r.ok() {
		switch {
		case panicked:
			t.Errorf("%s: unexpected panic: %v  [%s]", r, rec, ctx)
		case err != nil:
			t.Errorf("%s: unexpected error: %v  [%s]", r, err, ctx)
		default:
			if want := r.str(t, 0); got != want {
				t.Errorf("%s: want %q, got %q  [%s]", r, want, got, ctx)
			}
		}
		return
	}
	if err == nil && !panicked {
		t.Errorf("%s: expected %s, got %q  [%s]", r, r.outcome, got, ctx)
	}
}

func TestGoldenToStringDefault(t *testing.T) {
	golden(t, "tostring_default", func(t *testing.T, r row) {
		d := r.dec(t, 1)
		nf := nfFor(t, r.fields[2])
		var got string
		var err error
		rec, panicked := wantPanic(func() { got, err = FormatWith(d, "G", nf) })
		checkString(t, r, got, err, panicked, rec, formatBits(d))
	})
}

func TestGoldenToStringStandard(t *testing.T) {
	golden(t, "tostring_standard", func(t *testing.T, r row) {
		d := r.dec(t, 1)
		format := r.str(t, 2)
		nf := nfFor(t, r.fields[3])
		var got string
		var err error
		rec, panicked := wantPanic(func() { got, err = FormatWith(d, format, nf) })
		checkString(t, r, got, err, panicked, rec, formatBits(d)+" "+format)
	})
}

func TestGoldenToStringCustom(t *testing.T) {
	golden(t, "tostring_custom", func(t *testing.T, r row) {
		d := r.dec(t, 1)
		format := r.str(t, 2)
		nf := nfFor(t, r.fields[3])
		var got string
		var err error
		rec, panicked := wantPanic(func() { got, err = FormatWith(d, format, nf) })
		checkString(t, r, got, err, panicked, rec, formatBits(d)+" "+format)
	})
}

func TestGoldenParse(t *testing.T) {
	golden(t, "parse", func(t *testing.T, r row) {
		input := r.str(t, 1)
		style := r.fields[2]
		nf := nfFor(t, r.fields[3])

		var got Decimal
		var err error
		rec, panicked := wantPanic(func() { got, err = ParseStyle(input, styleFor(t, style), nf) })

		ctx := "input=" + strconv.Quote(input) + " style=" + style + " culture=" + r.fields[3]
		if r.ok() {
			switch {
			case panicked:
				t.Errorf("%s: unexpected panic: %v  [%s]", r, rec, ctx)
			case err != nil:
				t.Errorf("%s: unexpected error: %v  [%s]", r, err, ctx)
			default:
				want := r.dec(t, 0)
				if got != want {
					t.Errorf("%s: want %s (%s), got %s (%s)  [%s]",
						r, formatBits(want), want.String(), formatBits(got), got.String(), ctx)
				}
			}
			return
		}
		if err == nil && !panicked {
			t.Errorf("%s: expected %s, got %s (%s)  [%s]",
				r, r.outcome, formatBits(got), got.String(), ctx)
		}
	})
}

func styleFor(t *testing.T, name string) Styles {
	t.Helper()
	switch name {
	case "number":
		return Number
	case "float":
		return Float
	case "any":
		return Any
	case "integer":
		return Integer
	default:
		t.Fatalf("unknown NumberStyles name %q", name)
		return 0
	}
}
