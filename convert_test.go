package decimal

import (
	"math"
	"strconv"
	"testing"
)

// The conversion tables record the textual result .NET produced, or the
// exception it threw. Each Go conversion has a panicking and an error-returning
// form; both are checked against the same row so they cannot drift apart.
func TestGoldenToInteger(t *testing.T) {
	for _, tc := range []struct {
		file string
		fn   func(Decimal) string
		fnE  func(Decimal) (string, error)
	}{
		{"to_int8",
			func(d Decimal) string { return strconv.FormatInt(int64(d.Int8()), 10) },
			func(d Decimal) (string, error) { v, err := d.Int8E(); return strconv.FormatInt(int64(v), 10), err }},
		{"to_uint8",
			func(d Decimal) string { return strconv.FormatUint(uint64(d.Uint8()), 10) },
			func(d Decimal) (string, error) { v, err := d.Uint8E(); return strconv.FormatUint(uint64(v), 10), err }},
		{"to_int16",
			func(d Decimal) string { return strconv.FormatInt(int64(d.Int16()), 10) },
			func(d Decimal) (string, error) { v, err := d.Int16E(); return strconv.FormatInt(int64(v), 10), err }},
		{"to_uint16",
			func(d Decimal) string { return strconv.FormatUint(uint64(d.Uint16()), 10) },
			func(d Decimal) (string, error) { v, err := d.Uint16E(); return strconv.FormatUint(uint64(v), 10), err }},
		{"to_int32",
			func(d Decimal) string { return strconv.FormatInt(int64(d.Int32()), 10) },
			func(d Decimal) (string, error) { v, err := d.Int32E(); return strconv.FormatInt(int64(v), 10), err }},
		{"to_uint32",
			func(d Decimal) string { return strconv.FormatUint(uint64(d.Uint32()), 10) },
			func(d Decimal) (string, error) { v, err := d.Uint32E(); return strconv.FormatUint(uint64(v), 10), err }},
		{"to_int64",
			func(d Decimal) string { return strconv.FormatInt(d.Int64(), 10) },
			func(d Decimal) (string, error) { v, err := d.Int64E(); return strconv.FormatInt(v, 10), err }},
		{"to_uint64",
			func(d Decimal) string { return strconv.FormatUint(d.Uint64(), 10) },
			func(d Decimal) (string, error) { v, err := d.Uint64E(); return strconv.FormatUint(v, 10), err }},
	} {
		t.Run(tc.file, func(t *testing.T) {
			golden(t, tc.file, func(t *testing.T, r row) {
				d := r.dec(t, 1)

				var got string
				rec, panicked := wantPanic(func() { got = tc.fn(d) })
				gotE, err := tc.fnE(d)

				if r.ok() {
					want := r.str(t, 0)
					if panicked {
						t.Errorf("%s: unexpected panic: %v  [%s]", r, rec, formatBits(d))
					} else if got != want {
						t.Errorf("%s: want %s, got %s  [%s]", r, want, got, formatBits(d))
					}
					if err != nil {
						t.Errorf("%s: E form returned %v where the panicking form succeeded", r, err)
					} else if gotE != want {
						t.Errorf("%s: E form gave %s, want %s", r, gotE, want)
					}
					return
				}
				if !panicked {
					t.Errorf("%s: expected %s, got %s  [%s]", r, r.outcome, got, formatBits(d))
				}
				if err == nil {
					t.Errorf("%s: E form returned no error where %s was expected", r, r.outcome)
				}
			})
		})
	}
}

func TestGoldenToFloat(t *testing.T) {
	t.Run("float64", func(t *testing.T) {
		golden(t, "to_float64", func(t *testing.T, r row) {
			d := r.dec(t, 1)
			want, err := strconv.ParseFloat(r.str(t, 0), 64)
			if err != nil {
				t.Fatalf("%s: %v", r, err)
			}
			if got := d.Float64(); got != want {
				t.Errorf("%s: want %v, got %v  [%s]", r, want, got, formatBits(d))
			}
		})
	})

	t.Run("float32", func(t *testing.T) {
		golden(t, "to_float32", func(t *testing.T, r row) {
			d := r.dec(t, 1)
			want64, err := strconv.ParseFloat(r.str(t, 0), 32)
			if err != nil {
				t.Fatalf("%s: %v", r, err)
			}
			want := float32(want64)
			if got := d.Float32(); got != want {
				t.Errorf("%s: want %v, got %v  [%s]", r, want, got, formatBits(d))
			}
		})
	})
}

func TestGoldenFromInteger(t *testing.T) {
	t.Run("int64", func(t *testing.T) {
		golden(t, "from_int64", func(t *testing.T, r row) {
			n, err := strconv.ParseInt(r.fields[1], 10, 64)
			if err != nil {
				t.Fatalf("%s: %v", r, err)
			}
			checkDecimal(t, r, FromInt(n), false, nil)
		})
	})

	t.Run("uint64", func(t *testing.T) {
		golden(t, "from_uint64", func(t *testing.T, r row) {
			n, err := strconv.ParseUint(r.fields[1], 10, 64)
			if err != nil {
				t.Fatalf("%s: %v", r, err)
			}
			checkDecimal(t, r, FromUint(n), false, nil)
		})
	})
}

func TestGoldenFromFloat(t *testing.T) {
	t.Run("float64", func(t *testing.T) {
		golden(t, "from_float64", func(t *testing.T, r row) {
			f := parseFloatField(t, r, 64)
			got, err := FromFloat64(f)
			checkConverted(t, r, got, err, strconv.FormatFloat(f, 'g', -1, 64))
		})
	})

	t.Run("float32", func(t *testing.T) {
		golden(t, "from_float32", func(t *testing.T, r row) {
			f := parseFloatField(t, r, 32)
			got, err := FromFloat32(float32(f))
			checkConverted(t, r, got, err, strconv.FormatFloat(f, 'g', -1, 32))
		})
	})
}

// parseFloatField decodes the operand column, which may be a special value that
// strconv spells differently from .NET.
func parseFloatField(t *testing.T, r row, bitSize int) float64 {
	t.Helper()
	switch s := r.str(t, 1); s {
	case "NaN":
		return math.NaN()
	case "Infinity", "∞":
		return math.Inf(1)
	case "-Infinity", "-∞":
		return math.Inf(-1)
	default:
		f, err := strconv.ParseFloat(s, bitSize)
		if err != nil {
			t.Fatalf("%s: %v", r, err)
		}
		return f
	}
}

func checkConverted(t *testing.T, r row, got Decimal, err error, ctx string) {
	t.Helper()
	if r.ok() {
		if err != nil {
			t.Errorf("%s: unexpected error: %v  [%s]", r, err, ctx)
			return
		}
		if want := r.dec(t, 0); got != want {
			t.Errorf("%s: want %s (%s), got %s (%s)  [%s]",
				r, formatBits(want), want, formatBits(got), got, ctx)
		}
		return
	}
	if err == nil {
		t.Errorf("%s: expected %s, got %s (%s)  [%s]", r, r.outcome, formatBits(got), got, ctx)
	}
}

func TestGoldenOACurrency(t *testing.T) {
	golden(t, "oacurrency", func(t *testing.T, r row) {
		d := r.dec(t, 1)
		var got int64
		rec, panicked := wantPanic(func() { got = d.ToOACurrency() })
		if r.ok() {
			want, err := strconv.ParseInt(r.str(t, 0), 10, 64)
			if err != nil {
				t.Fatalf("%s: %v", r, err)
			}
			if panicked {
				t.Errorf("%s: unexpected panic: %v  [%s]", r, rec, formatBits(d))
			} else if got != want {
				t.Errorf("%s: want %d, got %d  [%s]", r, want, got, formatBits(d))
			}
			return
		}
		if !panicked {
			t.Errorf("%s: expected %s, got %d  [%s]", r, r.outcome, got, formatBits(d))
		}
	})
}
