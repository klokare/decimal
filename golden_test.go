package decimal

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The golden tables in testdata/ are generated from the .NET runtime by
// testdata/gen. Each row is:
//
//	id \t outcome \t result \t operand...
//
// A decimal field is "lo,mid,hi,flags" in hex, so a mismatch in scale, in the
// sign of a zero, or in trailing zeros is caught -- none of which survives a
// comparison of rendered numbers.
//
// outcome is "ok" or the kind of exception .NET raised: overflow,
// dividebyzero, format, range, argument.

// row is one decoded record.
type row struct {
	file    string
	line    int
	id      string
	outcome string
	fields  []string
}

func (r row) String() string { return fmt.Sprintf("%s:%d id=%s", r.file, r.line, r.id) }

// ok reports whether .NET completed the operation without throwing.
func (r row) ok() bool { return r.outcome == "ok" }

// dec decodes field i as a Decimal.
func (r row) dec(t *testing.T, i int) Decimal {
	t.Helper()
	d, err := parseBits(r.fields[i])
	if err != nil {
		t.Fatalf("%s: field %d: %v", r, i, err)
	}
	return d
}

// str decodes field i as a string, undoing the generator's escaping.
func (r row) str(t *testing.T, i int) string {
	t.Helper()
	s, err := unescape(r.fields[i])
	if err != nil {
		t.Fatalf("%s: field %d: %v", r, i, err)
	}
	return s
}

// num decodes field i as an integer.
func (r row) num(t *testing.T, i int) int {
	t.Helper()
	n, err := strconv.Atoi(r.fields[i])
	if err != nil {
		t.Fatalf("%s: field %d: %v", r, i, err)
	}
	return n
}

// parseBits decodes "lo,mid,hi,flags" in hex into a Decimal, bypassing any
// validation so that malformed representations from the generator would surface
// as test failures rather than being silently normalised.
func parseBits(s string) (Decimal, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return Decimal{}, fmt.Errorf("want 4 words, got %d in %q", len(parts), s)
	}
	var w [4]uint32
	for i, p := range parts {
		v, err := strconv.ParseUint(p, 16, 32)
		if err != nil {
			return Decimal{}, fmt.Errorf("word %d of %q: %w", i, s, err)
		}
		w[i] = uint32(v)
	}
	return Decimal{low: w[0], mid: w[1], high: w[2], flags: w[3]}, nil
}

// formatBits renders a Decimal the way the golden files do, for failure messages.
func formatBits(d Decimal) string {
	return fmt.Sprintf("%08x,%08x,%08x,%08x", d.low, d.mid, d.high, d.flags)
}

// unescape reverses Writer.Str from the generator.
func unescape(s string) (string, error) {
	switch s {
	case `\e`:
		return "", nil
	case `\0`:
		return "", nil
	}
	if !strings.ContainsRune(s, '\\') {
		return s, nil
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' {
			b.WriteByte(s[i])
			continue
		}
		i++
		if i >= len(s) {
			return "", fmt.Errorf("trailing backslash in %q", s)
		}
		switch s[i] {
		case 't':
			b.WriteByte('\t')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case '\\':
			b.WriteByte('\\')
		case 'u':
			if i+4 >= len(s) {
				return "", fmt.Errorf("short \\u escape in %q", s)
			}
			v, err := strconv.ParseUint(s[i+1:i+5], 16, 32)
			if err != nil {
				return "", fmt.Errorf("bad \\u escape in %q: %w", s, err)
			}
			b.WriteRune(rune(v))
			i += 4
		default:
			return "", fmt.Errorf("unknown escape %q in %q", s[i], s)
		}
	}
	return b.String(), nil
}

// golden streams a table, calling fn for each record. It skips the file (rather
// than failing) when it is absent, so a partially regenerated testdata directory
// still runs everything else.
func golden(t *testing.T, name string, fn func(t *testing.T, r row)) {
	t.Helper()
	path := filepath.Join("testdata", name+".txt")

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		t.Skipf("%s not generated; run `make testdata`", path)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var n int
	for line := 1; sc.Scan(); line++ {
		text := sc.Text()
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		parts := strings.Split(text, "\t")
		if len(parts) < 3 {
			t.Fatalf("%s:%d: want at least 3 columns, got %d", path, line, len(parts))
		}
		r := row{file: name, line: line, id: parts[0], outcome: parts[1], fields: parts[2:]}
		fn(t, r)
		n++
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatalf("%s contained no records", path)
	}
	t.Logf("%s: %d records", path, n)
}

// wantPanic runs fn, reporting whether it panicked and with what.
func wantPanic(fn func()) (recovered any, panicked bool) {
	defer func() {
		if v := recover(); v != nil {
			recovered, panicked = v, true
		}
	}()
	fn()
	return nil, false
}

// checkDecimal compares a produced Decimal against the golden row, honouring the
// recorded outcome. It reports whether the case matched.
func checkDecimal(t *testing.T, r row, got Decimal, panicked bool, recovered any) {
	t.Helper()
	if r.ok() {
		if panicked {
			t.Errorf("%s: unexpected panic: %v\n  operands: %v", r, recovered, r.fields[1:])
			return
		}
		want := r.dec(t, 0)
		if got != want {
			t.Errorf("%s: wrong result\n  want %s (%s)\n   got %s (%s)\n  operands: %v",
				r, formatBits(want), want.String(), formatBits(got), got.String(), r.fields[1:])
		}
		return
	}
	if !panicked {
		t.Errorf("%s: expected %s, got %s (%s)\n  operands: %v",
			r, r.outcome, formatBits(got), got.String(), r.fields[1:])
	}
}
