package decimal

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

// The Invariant and EnUS defaults are transcribed from .NET's NumberFormatInfo.
// This checks the transcription against the runtime itself rather than trusting
// it, using the culture dump the generator emits.
func TestNumberFormatMatchesDotNet(t *testing.T) {
	byCulture := map[string]*NumberFormat{
		"invariant": Invariant,
		"en-US":     EnUS,
	}

	seen := map[string]int{}

	golden(t, "numberformat", func(t *testing.T, r row) {
		culture, field := r.fields[1], r.fields[2]
		nf, ok := byCulture[culture]
		if !ok {
			t.Fatalf("%s: no NumberFormat for culture %q", r, culture)
		}
		seen[culture]++

		want := r.str(t, 0)
		got, err := nfField(nf, field)
		if err != nil {
			t.Fatalf("%s: %v", r, err)
		}
		if got != want {
			t.Errorf("%s: %s.%s = %q, .NET says %q", r, culture, field, got, want)
		}
	})

	// Guard against the dump silently losing fields.
	for culture, n := range seen {
		if n != 21 {
			t.Errorf("%s: checked %d fields, expected 21", culture, n)
		}
	}
}

// nfField renders one NumberFormat field the way the generator wrote it.
func nfField(nf *NumberFormat, field string) (string, error) {
	itoa := strconv.Itoa
	join := func(v []int) string {
		parts := make([]string, len(v))
		for i, n := range v {
			parts[i] = itoa(n)
		}
		return strings.Join(parts, ",")
	}

	switch field {
	case "NumberDecimalSeparator":
		return nf.NumberDecimalSeparator, nil
	case "NumberGroupSeparator":
		return nf.NumberGroupSeparator, nil
	case "NumberGroupSizes":
		return join(nf.NumberGroupSizes), nil
	case "NumberDecimalDigits":
		return itoa(nf.NumberDecimalDigits), nil
	case "NumberNegativePattern":
		return itoa(nf.NumberNegativePattern), nil
	case "NegativeSign":
		return nf.NegativeSign, nil
	case "PositiveSign":
		return nf.PositiveSign, nil
	case "CurrencySymbol":
		return nf.CurrencySymbol, nil
	case "CurrencyDecimalSeparator":
		return nf.CurrencyDecimalSeparator, nil
	case "CurrencyGroupSeparator":
		return nf.CurrencyGroupSeparator, nil
	case "CurrencyGroupSizes":
		return join(nf.CurrencyGroupSizes), nil
	case "CurrencyDecimalDigits":
		return itoa(nf.CurrencyDecimalDigits), nil
	case "CurrencyPositivePattern":
		return itoa(nf.CurrencyPositivePattern), nil
	case "CurrencyNegativePattern":
		return itoa(nf.CurrencyNegativePattern), nil
	case "PercentSymbol":
		return nf.PercentSymbol, nil
	case "PercentDecimalSeparator":
		return nf.PercentDecimalSeparator, nil
	case "PercentGroupSeparator":
		return nf.PercentGroupSeparator, nil
	case "PercentGroupSizes":
		return join(nf.PercentGroupSizes), nil
	case "PercentDecimalDigits":
		return itoa(nf.PercentDecimalDigits), nil
	case "PercentPositivePattern":
		return itoa(nf.PercentPositivePattern), nil
	case "PercentNegativePattern":
		return itoa(nf.PercentNegativePattern), nil
	default:
		return "", fmt.Errorf("unknown NumberFormatInfo field %q", field)
	}
}

func TestNumberFormatClone(t *testing.T) {
	c := EnUS.Clone()
	c.CurrencySymbol = "€"
	c.NumberGroupSizes[0] = 4
	if EnUS.CurrencySymbol != "$" {
		t.Errorf("Clone shared the currency symbol: EnUS is now %q", EnUS.CurrencySymbol)
	}
	if EnUS.NumberGroupSizes[0] != 3 {
		t.Errorf("Clone shared the group sizes: EnUS is now %v", EnUS.NumberGroupSizes)
	}
	if got := (*NumberFormat)(nil).Clone(); got.CurrencySymbol != Invariant.CurrencySymbol {
		t.Errorf("nil.Clone() should copy Invariant, got %q", got.CurrencySymbol)
	}
}

func TestPatternFallback(t *testing.T) {
	// .NET validates pattern indices on assignment; a Go struct literal cannot,
	// so out-of-range values must degrade rather than panic.
	for _, i := range []int{-1, 99} {
		if got := pattern(negNumberFormats[:], i, 1); got != "-#" {
			t.Errorf("pattern(negNumberFormats, %d) = %q, want the fallback %q", i, got, "-#")
		}
	}
	if got := pattern(negCurrencyFormats[:], 16, 0); got != "$- #" {
		t.Errorf("index 16 should be in range for negCurrencyFormats, got %q", got)
	}
}
