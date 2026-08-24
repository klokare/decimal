using System.Globalization;

namespace gen;

/// <summary>
/// Generates the golden-data tables the Go test suite checks itself against.
///
/// Usage: dotnet run --project testdata/gen -- &lt;output-dir&gt;
///
/// Output is deterministic: the corpus is fixed, the pseudo-random sweep is
/// seeded, and newlines are forced to "\n". Regenerating on any machine must
/// produce byte-identical files, which is what makes `make testdata &amp;&amp;
/// git diff --exit-code testdata/` a usable CI check.
/// </summary>
public static class Program
{
    static string _dir = ".";

    public static int Main(string[] args)
    {
        _dir = args.Length > 0 ? args[0] : ".";
        var all = Corpus.Values;
        var small = Corpus.Small;
        Console.WriteLine($"corpus: {all.Length} values, {small.Length} quadratic, {Corpus.Format.Length} formatting");

        Binary(small);
        Unary(all);
        Rounding(all);
        Conversions(all);
        Comparisons(small);
        Parsing();
        Formatting(Corpus.Format);
        NumberFormats();

        Console.WriteLine("done");
        return 0;
    }

    static Writer W(string name, params string[] cols) => new(_dir, name, cols);

    // ---- binary arithmetic -------------------------------------------------

    static void Binary(decimal[] v)
    {
        var ops = new (string Name, Func<decimal, decimal, decimal> Fn)[]
        {
            ("add", (a, b) => decimal.Add(a, b)),
            ("sub", (a, b) => decimal.Subtract(a, b)),
            ("mul", (a, b) => decimal.Multiply(a, b)),
            ("div", (a, b) => decimal.Divide(a, b)),
            ("rem", (a, b) => decimal.Remainder(a, b)),
        };

        foreach (var (name, fn) in ops)
        {
            using var w = W(name, "result", "a", "b");
            foreach (var a in v)
                foreach (var b in v)
                    w.Try(() => fn(a, b), Writer.Bits(a), Writer.Bits(b));
            Report(name, w);
        }
    }

    // ---- unary -------------------------------------------------------------

    static void Unary(decimal[] v)
    {
        var ops = new (string Name, Func<decimal, decimal> Fn)[]
        {
            ("abs", d => Math.Abs(d)),
            ("neg", d => decimal.Negate(d)),
            ("floor", d => decimal.Floor(d)),
            ("ceil", d => decimal.Ceiling(d)),
            ("truncate", d => decimal.Truncate(d)),
        };

        foreach (var (name, fn) in ops)
        {
            using var w = W(name, "result", "a");
            foreach (var d in v) w.Try(() => fn(d), Writer.Bits(d));
            Report(name, w);
        }
    }

    // ---- rounding ----------------------------------------------------------

    static void Rounding(decimal[] v)
    {
        // .NET's Decimal.Round exposes only these two midpoint modes. The Go port
        // additionally offers Truncate/Floor/Ceiling, which are covered by the
        // truncate/floor/ceil tables above applied at a scale.
        var modes = new (string Name, MidpointRounding Mode)[]
        {
            ("toeven", MidpointRounding.ToEven),
            ("awayfromzero", MidpointRounding.AwayFromZero),
        };

        using var w = W("round", "result", "a", "decimals", "mode");
        foreach (var d in v)
            foreach (var (mname, mode) in modes)
                for (int places = 0; places <= 28; places++)
                    w.Try(() => decimal.Round(d, places, mode),
                          Writer.Bits(d),
                          places.ToString(CultureInfo.InvariantCulture),
                          mname);
        Report("round", w);
    }

    // ---- conversions -------------------------------------------------------

    static void Conversions(decimal[] v)
    {
        // decimal -> integral. Each records the textual value so a Go failure is
        // readable without decoding hex by hand.
        var toInt = new (string Name, Func<decimal, string> Fn)[]
        {
            ("to_int8",   d => ((sbyte)d).ToString(CultureInfo.InvariantCulture)),
            ("to_uint8",  d => ((byte)d).ToString(CultureInfo.InvariantCulture)),
            ("to_int16",  d => ((short)d).ToString(CultureInfo.InvariantCulture)),
            ("to_uint16", d => ((ushort)d).ToString(CultureInfo.InvariantCulture)),
            ("to_int32",  d => decimal.ToInt32(d).ToString(CultureInfo.InvariantCulture)),
            ("to_uint32", d => decimal.ToUInt32(d).ToString(CultureInfo.InvariantCulture)),
            ("to_int64",  d => decimal.ToInt64(d).ToString(CultureInfo.InvariantCulture)),
            ("to_uint64", d => decimal.ToUInt64(d).ToString(CultureInfo.InvariantCulture)),
        };

        foreach (var (name, fn) in toInt)
        {
            using var w = W(name, "result", "a");
            foreach (var d in v) w.TryStr(() => fn(d), Writer.Bits(d));
            Report(name, w);
        }

        // decimal -> float. Round-trip ("R") so no precision is lost in the file.
        using (var w = W("to_float32", "result", "a"))
        {
            foreach (var d in v)
                w.TryStr(() => ((float)d).ToString("R", CultureInfo.InvariantCulture), Writer.Bits(d));
            Report("to_float32", w);
        }
        using (var w = W("to_float64", "result", "a"))
        {
            foreach (var d in v)
                w.TryStr(() => ((double)d).ToString("R", CultureInfo.InvariantCulture), Writer.Bits(d));
            Report("to_float64", w);
        }

        // integral -> decimal, over the boundary values of each width.
        using (var w = W("from_int64", "result", "a"))
        {
            foreach (var n in IntegerCorpus())
                w.Try(() => new decimal(n), n.ToString(CultureInfo.InvariantCulture));
            Report("from_int64", w);
        }
        using (var w = W("from_uint64", "result", "a"))
        {
            foreach (var n in UintCorpus())
                w.Try(() => new decimal(n), n.ToString(CultureInfo.InvariantCulture));
            Report("from_uint64", w);
        }

        // float -> decimal, including the values that must fail.
        using (var w = W("from_float32", "result", "a"))
        {
            foreach (var f in FloatCorpus())
                w.Try(() => new decimal((float)f), ((float)f).ToString("R", CultureInfo.InvariantCulture));
            Report("from_float32", w);
        }
        using (var w = W("from_float64", "result", "a"))
        {
            foreach (var f in FloatCorpus())
                w.Try(() => new decimal(f), f.ToString("R", CultureInfo.InvariantCulture));
            Report("from_float64", w);
        }

        // OLE Automation currency, which the Go port mirrors.
        using (var w = W("oacurrency", "result", "a"))
        {
            foreach (var d in v)
                w.TryStr(() => decimal.ToOACurrency(d).ToString(CultureInfo.InvariantCulture), Writer.Bits(d));
            Report("oacurrency", w);
        }
    }

    static IEnumerable<long> IntegerCorpus()
    {
        long[] seeds = { 0, 1, -1, 2, -2, 9, 10, 99, 100,
                         sbyte.MaxValue, sbyte.MinValue, byte.MaxValue,
                         short.MaxValue, short.MinValue, ushort.MaxValue,
                         int.MaxValue, int.MinValue, uint.MaxValue,
                         long.MaxValue, long.MinValue };
        foreach (var s in seeds)
        {
            yield return s;
            if (s != long.MaxValue) yield return s + 1;
            if (s != long.MinValue) yield return s - 1;
        }
    }

    static IEnumerable<ulong> UintCorpus()
    {
        ulong[] seeds = { 0, 1, 2, byte.MaxValue, ushort.MaxValue, uint.MaxValue,
                          (ulong)uint.MaxValue + 1, ulong.MaxValue, ulong.MaxValue - 1 };
        foreach (var s in seeds) yield return s;
    }

    static IEnumerable<double> FloatCorpus()
    {
        double[] seeds =
        {
            0d, -0d, 1d, -1d, 0.1d, -0.1d, 0.5d, 2.5d, 1e-28d, 1e28d,
            double.MaxValue, double.MinValue, double.Epsilon, -double.Epsilon,
            double.NaN, double.PositiveInfinity, double.NegativeInfinity,
            7.9228162514264337593543950335e28d,   // just at the decimal ceiling
            7.9228162514264338e28d,               // just over it
            1e29d, -1e29d, 1e-29d, 123456789.123456789d, 3.14159265358979d,
            (double)float.MaxValue, (double)float.MinValue, (double)float.Epsilon,
        };
        return seeds;
    }

    // ---- comparison --------------------------------------------------------

    static void Comparisons(decimal[] v)
    {
        using var w = W("compare", "result", "a", "b");
        foreach (var a in v)
            foreach (var b in v)
            {
                // One row carries every predicate, so the Go side can prove they
                // stay mutually consistent as well as individually correct.
                int c = decimal.Compare(a, b);
                w.Row("ok",
                    string.Join(",",
                        Math.Sign(c).ToString(CultureInfo.InvariantCulture),
                        a == b ? "1" : "0",
                        a != b ? "1" : "0",
                        a < b ? "1" : "0",
                        a <= b ? "1" : "0",
                        a > b ? "1" : "0",
                        a >= b ? "1" : "0",
                        a.Equals(b) ? "1" : "0"),
                    Writer.Bits(a), Writer.Bits(b));
            }
        Report("compare", w);
    }

    // ---- parsing -----------------------------------------------------------

    static void Parsing()
    {
        using var w = W("parse", "result", "input", "style", "culture");

        var cultures = new (string Name, CultureInfo Info)[]
        {
            ("invariant", CultureInfo.InvariantCulture),
            ("en-US", new CultureInfo("en-US")),
        };

        var styles = new (string Name, NumberStyles Style)[]
        {
            ("number", NumberStyles.Number),
            ("float", NumberStyles.Float),
            ("any", NumberStyles.Any),
            ("integer", NumberStyles.Integer),
        };

        foreach (var input in ParseCorpus())
            foreach (var (sname, style) in styles)
                foreach (var (cname, culture) in cultures)
                    w.Try(() => decimal.Parse(input, style, culture),
                          Writer.Str(input), sname, cname);
        Report("parse", w);
    }

    static IEnumerable<string> ParseCorpus() => new[]
    {
        // well formed
        "0", "-0", "+0", "1", "-1", "1.0", "1.00", "1.000000000000000000000000000",
        "0.1", ".1", "1.", "0.0000000000000000000000000001",
        "79228162514264337593543950335", "-79228162514264337593543950335",
        "79228162514264337593543950336",           // one past the maximum
        "7.9228162514264337593543950335",
        "1,234", "1,234,567.89", "1,2,3", "1,234.5",
        "  12  ", "\t12\t", "12 ", " 12",
        "(1.5)", "1.5-", "-1.5", "1.5+",
        "$1,234.56", "$-1234", "-$1234",
        "1e5", "1E5", "1e+5", "1e-5", "1E-28", "1e28", "1e29", "1e-29",
        "1.5e3", "-1.5e-3",
        "0000000000000000000000000000001",
        "1.00000000000000000000000000005",          // rounds at the 29th digit
        "1.00000000000000000000000000015",
        "0.500000000000000000000000000050",
        // malformed -- every one of these must fail
        "", " ", "\t", ".", "-", "+", "--1", "1-2", "1.2.3", "1..2",
        "abc", "12abc", "ysaidufljasdf", "0x10", "1_000",
        "NaN", "Infinity", "-Infinity", "1e", "e5", "1e+", "1e999999",
        "(1.5", "1.5)", "()", "1,,234", ",", "1 234",
        " ", "​1",
    };

    // ---- formatting --------------------------------------------------------

    static void Formatting(decimal[] v)
    {
        var cultures = new (string Name, CultureInfo Info)[]
        {
            ("invariant", CultureInfo.InvariantCulture),
            ("en-US", new CultureInfo("en-US")),
        };

        // Standard specifiers, upper and lower case, bare and with 0..9 digits,
        // plus a few wider precisions that exercise the padding paths.
        var standard = new List<string>();
        foreach (char c in "CEFGNPR")
            foreach (char cc in new[] { c, char.ToLowerInvariant(c) })
            {
                standard.Add(cc.ToString());
                for (int d = 0; d <= 9; d++) standard.Add($"{cc}{d}");
                // Past the 28-digit limit the padding paths take over.
                foreach (int d in new[] { 15, 28, 30 }) standard.Add($"{cc}{d}");
            }
        // D and X are integral-only and must fail for decimal whatever follows them,
        // so a couple of samples each is enough to pin the error.
        standard.AddRange(new[] { "D", "d", "D2", "X", "x", "X2" });

        using (var w = W("tostring_standard", "result", "a", "format", "culture"))
        {
            foreach (var d in v)
                foreach (var f in standard)
                    foreach (var (cname, culture) in cultures)
                        w.TryStr(() => d.ToString(f, culture), Writer.Bits(d), Writer.Str(f), cname);
            Report("tostring_standard", w);
        }

        // The default ToString(), which is what Go's String() must match.
        using (var w = W("tostring_default", "result", "a", "culture"))
        {
            foreach (var d in v)
                foreach (var (cname, culture) in cultures)
                    w.TryStr(() => d.ToString(culture), Writer.Bits(d), cname);
            Report("tostring_default", w);
        }

        // Custom picture formats.
        using (var w = W("tostring_custom", "result", "a", "format", "culture"))
        {
            foreach (var d in v)
                foreach (var f in CustomFormats())
                    foreach (var (cname, culture) in cultures)
                        w.TryStr(() => d.ToString(f, culture), Writer.Bits(d), Writer.Str(f), cname);
            Report("tostring_custom", w);
        }
    }

    static IEnumerable<string> CustomFormats() => new[]
    {
        "#", "0", "0.0", "0.00", "#.##", "#.00", "0.###", "#,#", "#,##0",
        "#,##0.00", "#,,", "#,##0,,", "0%", "#.##%", "0.0%",
        "00000", "###00.##", ".##", "#.",
        "0.0E+0", "0.0E-0", "#.##E+00", "0E0",
        "0;(0)", "0;(0);zero", "#,##0.00;(#,##0.00);-",
        @"\#0", "'lit'0", "\"q\"0", "0'x'", "0 'units'",
        "0.00 %", "$#,##0.00", "#,##0.00 EUR",
        "", "abc", "0000000000000000000000000000000000",
    };

    // ---- culture data ------------------------------------------------------

    /// <summary>
    /// Dumps the NumberFormatInfo fields the formatter and parser read, so the Go
    /// defaults are transcribed from the runtime rather than from memory.
    /// </summary>
    static void NumberFormats()
    {
        using var w = W("numberformat", "value", "culture", "field");

        var cultures = new (string Name, NumberFormatInfo Info)[]
        {
            ("invariant", CultureInfo.InvariantCulture.NumberFormat),
            ("en-US", new CultureInfo("en-US").NumberFormat),
        };

        foreach (var (cname, f) in cultures)
        {
            void S(string field, string v) => w.Row("ok", Writer.Str(v), cname, field);
            void I(string field, int v) => w.Row("ok", v.ToString(CultureInfo.InvariantCulture), cname, field);
            void A(string field, int[] v) => w.Row("ok",
                string.Join(",", v.Select(x => x.ToString(CultureInfo.InvariantCulture))), cname, field);

            S("NumberDecimalSeparator", f.NumberDecimalSeparator);
            S("NumberGroupSeparator", f.NumberGroupSeparator);
            A("NumberGroupSizes", f.NumberGroupSizes);
            I("NumberDecimalDigits", f.NumberDecimalDigits);
            I("NumberNegativePattern", f.NumberNegativePattern);

            S("NegativeSign", f.NegativeSign);
            S("PositiveSign", f.PositiveSign);

            S("CurrencySymbol", f.CurrencySymbol);
            S("CurrencyDecimalSeparator", f.CurrencyDecimalSeparator);
            S("CurrencyGroupSeparator", f.CurrencyGroupSeparator);
            A("CurrencyGroupSizes", f.CurrencyGroupSizes);
            I("CurrencyDecimalDigits", f.CurrencyDecimalDigits);
            I("CurrencyPositivePattern", f.CurrencyPositivePattern);
            I("CurrencyNegativePattern", f.CurrencyNegativePattern);

            S("PercentSymbol", f.PercentSymbol);
            S("PercentDecimalSeparator", f.PercentDecimalSeparator);
            S("PercentGroupSeparator", f.PercentGroupSeparator);
            A("PercentGroupSizes", f.PercentGroupSizes);
            I("PercentDecimalDigits", f.PercentDecimalDigits);
            I("PercentPositivePattern", f.PercentPositivePattern);
            I("PercentNegativePattern", f.PercentNegativePattern);
        }
        Report("numberformat", w);
    }

    static void Report(string name, Writer w) =>
        Console.WriteLine($"  {name,-20} {w.Count,8} rows");
}
