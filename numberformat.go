package decimal

// NumberFormat carries the symbols and layout patterns used to render and parse
// a Decimal. It is this package's stand-in for .NET's NumberFormatInfo.
//
// Go has no ambient culture: there is no thread-current locale to consult and no
// locale database in the standard library. So a NumberFormat is passed
// explicitly, and the package-level entry points default to [Invariant]. The
// pattern fields keep .NET's numbering exactly, so a value read from
// NumberFormatInfo can be copied across without translation.
//
// A NumberFormat is read-only once in use. Build one with [NumberFormat.Clone]
// and mutate the copy rather than modifying [Invariant] or [EnUS], which are
// shared.
type NumberFormat struct {
	// Number formatting, used by the "F" and "N" specifiers and by custom
	// picture formats.
	NumberDecimalSeparator string
	NumberGroupSeparator   string
	NumberGroupSizes       []int
	NumberDecimalDigits    int
	// NumberNegativePattern selects from: 0:(n) 1:-n 2:- n 3:n- 4:n -
	NumberNegativePattern int

	NegativeSign string
	PositiveSign string

	// Currency formatting, used by the "C" specifier.
	CurrencySymbol           string
	CurrencyDecimalSeparator string
	CurrencyGroupSeparator   string
	CurrencyGroupSizes       []int
	CurrencyDecimalDigits    int
	// CurrencyPositivePattern selects from: 0:$n 1:n$ 2:$ n 3:n $
	CurrencyPositivePattern int
	// CurrencyNegativePattern selects from:
	//  0:($n)  1:-$n  2:$-n  3:$n-   4:(n$)  5:-n$   6:n-$   7:n$-
	//  8:-n $  9:-$ n 10:n $- 11:$ n- 12:$ -n 13:n- $ 14:($ n) 15:(n $) 16:$- n
	CurrencyNegativePattern int

	// Percent formatting, used by the "P" specifier.
	PercentSymbol           string
	PercentDecimalSeparator string
	PercentGroupSeparator   string
	PercentGroupSizes       []int
	PercentDecimalDigits    int
	// PercentPositivePattern selects from: 0:n % 1:n% 2:%n 3:% n
	PercentPositivePattern int
	// PercentNegativePattern selects from:
	//  0:-n %  1:-n%  2:-%n  3:%-n  4:%n-  5:n-%
	//  6:n%-   7:-% n 8:n %- 9:% n- 10:% -n 11:n- %
	PercentNegativePattern int
}

// Invariant matches .NET's NumberFormatInfo.InvariantInfo. Its currency symbol
// is the generic currency sign U+00A4, not a national one.
var Invariant = &NumberFormat{
	NumberDecimalSeparator: ".",
	NumberGroupSeparator:   ",",
	NumberGroupSizes:       []int{3},
	NumberDecimalDigits:    2,
	NumberNegativePattern:  1,

	NegativeSign: "-",
	PositiveSign: "+",

	CurrencySymbol:           "¤",
	CurrencyDecimalSeparator: ".",
	CurrencyGroupSeparator:   ",",
	CurrencyGroupSizes:       []int{3},
	CurrencyDecimalDigits:    2,
	CurrencyPositivePattern:  0,
	CurrencyNegativePattern:  0,

	PercentSymbol:           "%",
	PercentDecimalSeparator: ".",
	PercentGroupSeparator:   ",",
	PercentGroupSizes:       []int{3},
	PercentDecimalDigits:    2,
	PercentPositivePattern:  0,
	PercentNegativePattern:  0,
}

// EnUS matches .NET's en-US culture. It differs from [Invariant] in more than
// the currency symbol: en-US asks for three decimal digits by default in the
// number and percent formats, and uses different negative patterns.
var EnUS = &NumberFormat{
	NumberDecimalSeparator: ".",
	NumberGroupSeparator:   ",",
	NumberGroupSizes:       []int{3},
	NumberDecimalDigits:    3,
	NumberNegativePattern:  1,

	NegativeSign: "-",
	PositiveSign: "+",

	CurrencySymbol:           "$",
	CurrencyDecimalSeparator: ".",
	CurrencyGroupSeparator:   ",",
	CurrencyGroupSizes:       []int{3},
	CurrencyDecimalDigits:    2,
	CurrencyPositivePattern:  0,
	CurrencyNegativePattern:  1,

	PercentSymbol:           "%",
	PercentDecimalSeparator: ".",
	PercentGroupSeparator:   ",",
	PercentGroupSizes:       []int{3},
	PercentDecimalDigits:    3,
	PercentPositivePattern:  1,
	PercentNegativePattern:  1,
}

// Clone returns a deep copy, safe to modify.
func (nf *NumberFormat) Clone() *NumberFormat {
	if nf == nil {
		return Invariant.Clone()
	}
	c := *nf
	c.NumberGroupSizes = append([]int(nil), nf.NumberGroupSizes...)
	c.CurrencyGroupSizes = append([]int(nil), nf.CurrencyGroupSizes...)
	c.PercentGroupSizes = append([]int(nil), nf.PercentGroupSizes...)
	return &c
}

// orInvariant lets every entry point accept a nil NumberFormat.
func (nf *NumberFormat) orInvariant() *NumberFormat {
	if nf == nil {
		return Invariant
	}
	return nf
}

// The layout patterns, indexed by the corresponding *Pattern field. Ported
// verbatim from Number.Formatting.cs so the indices keep their .NET meanings.
//
// In each pattern '#' is the formatted number, '$' the currency symbol, '%' the
// percent symbol, '-' the negative sign, and any other character is a literal.
var (
	posCurrencyFormats = [...]string{
		"$#", "#$", "$ #", "# $",
	}
	negCurrencyFormats = [...]string{
		"($#)", "-$#", "$-#", "$#-",
		"(#$)", "-#$", "#-$", "#$-",
		"-# $", "-$ #", "# $-", "$ #-",
		"$ -#", "#- $", "($ #)", "(# $)",
		"$- #",
	}
	posPercentFormats = [...]string{
		"# %", "#%", "%#", "% #",
	}
	negPercentFormats = [...]string{
		"-# %", "-#%", "-%#",
		"%-#", "%#-",
		"#-%", "#%-",
		"-% #", "# %-", "% #-",
		"% -#", "#- %",
	}
	negNumberFormats = [...]string{
		"(#)", "-#", "- #", "#-", "# -",
	}
	// The positive number format has no pattern field in .NET; it is always
	// the bare number.
	posNumberFormat = "#"
)

// pattern picks entry i from table, falling back to the .NET default index when
// i is out of range. .NET validates these on assignment to NumberFormatInfo; a
// Go struct literal has no such hook, so an out-of-range value degrades to the
// default rather than panicking deep inside the formatter.
func pattern(table []string, i, fallback int) string {
	if i < 0 || i >= len(table) {
		i = fallback
	}
	return table[i]
}
