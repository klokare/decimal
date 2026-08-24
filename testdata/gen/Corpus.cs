using System.Globalization;

namespace gen;

/// <summary>
/// The decimal values every generated table is built from.
///
/// The corpus is deliberately adversarial rather than random: it concentrates on
/// the boundaries where a reimplementation is most likely to disagree with .NET --
/// scale extremes, 32/64/96-bit carry boundaries, rounding midpoints, and values
/// that are numerically equal but differently scaled.
/// </summary>
public static class Corpus
{
    public static decimal[] Values { get; } = Build();

    /// <summary>A smaller slice, for tables that are quadratic in the corpus size.</summary>
    public static decimal[] Small { get; } = BuildSmall();

    /// <summary>
    /// The slice used for formatting tables. Those are the product of values,
    /// format strings and cultures, so they grow fast; this keeps the boundary
    /// cases that actually stress a formatter (scale extremes, midpoints, the
    /// 96-bit limits, negative zero) and drops the random sweep, which exercises
    /// nothing a chosen value does not.
    /// </summary>
    public static decimal[] Format { get; } = BuildFormat();

    static decimal[] BuildFormat()
    {
        var v = new List<decimal>(BuildSmall());
        foreach (byte scale in new byte[] { 0, 1, 2, 3, 7, 14, 27, 28 })
        {
            v.Add(new decimal(1, 0, 0, false, scale));
            v.Add(new decimal(1, 0, 0, true, scale));
            v.Add(new decimal(0, 0, 0, true, scale));           // negative zero, scaled
            v.Add(new decimal(-1, -1, -1, false, scale));       // 96-bit maximum, scaled
        }
        v.Add(123456.7891m);
        v.Add(-123456.7892m);
        v.Add(1123456.7890m);
        v.Add(1.0034m);
        return Dedup(v);
    }

    static decimal P(string s) => decimal.Parse(s, NumberStyles.Float, CultureInfo.InvariantCulture);

    static decimal[] BuildSmall()
    {
        var v = new List<decimal>
        {
            0m,
            decimal.Negate(0m),        // negative zero: sign bit set, no digits
            1m, -1m, 2m, -2m, 10m, 0.1m, 0.11m,
            1.0m, 1.00m, 1.0000m,      // equal in value, different scale
            0.5m, 2.5m, 2.25m, -0.5m, -2.5m,
            decimal.MaxValue, decimal.MinValue,
            new decimal(1, 0, 0, false, 28),   // smallest positive
            new decimal(1, 0, 0, true, 28),    // smallest negative
            P("5.5555555555555555555555555555"),
            P("79228162514264337593543950334"),
            P("27703302467091960609331879.532"),
            P("-3203854.9559968181492513385018"),
            P("-48466870444188873796420.0286"),
            P("-48466870444188873796420.02860"),  // same value, extra scale
            P("4294967295"),   // 2^32-1
            P("4294967296"),   // 2^32
            P("18446744073709551615"),  // 2^64-1
            P("18446744073709551616"),  // 2^64
            P("0.0000000000000000000000000001"),
            P("1e-28"), P("1e28"),
            P("3.14159265358979323846264338"),
            P("-1.3356"), P("1.3"), P("3.6"),
            P("12.345678"), P("0.0012"), P("-0.001234"), P("-0.000012"),
            P("123456.7890"), P("-123456.7892"), P("1234.56789"),
            P("45937986975432"), P("43987453"),
        };
        return v.ToArray();
    }

    static decimal[] Build()
    {
        var v = new List<decimal>(BuildSmall());

        // Every power of ten reachable at every scale.
        for (int scale = 0; scale <= 28; scale++)
        {
            v.Add(new decimal(1, 0, 0, false, (byte)scale));
            v.Add(new decimal(1, 0, 0, true, (byte)scale));
        }

        // 96-bit word boundaries, at a few scales.
        uint[] words = { 0, 1, 2, uint.MaxValue - 1, uint.MaxValue };
        foreach (var hi in words)
            foreach (var scale in new byte[] { 0, 1, 14, 28 })
            {
                v.Add(new decimal(unchecked((int)uint.MaxValue), unchecked((int)uint.MaxValue),
                                  unchecked((int)hi), false, scale));
                v.Add(new decimal(0, 0, unchecked((int)hi), true, scale));
            }

        // Rounding midpoints at every scale: 0.5, 0.05, 0.005, ...
        for (int scale = 1; scale <= 28; scale++)
        {
            v.Add(new decimal(5, 0, 0, false, (byte)scale));
            v.Add(new decimal(5, 0, 0, true, (byte)scale));
            v.Add(new decimal(15, 0, 0, false, (byte)scale));
            v.Add(new decimal(25, 0, 0, false, (byte)scale));
        }

        // A reproducible pseudo-random sweep. The seed is fixed so regenerating the
        // tables produces a byte-identical file.
        var rng = new Random(20260824);
        for (int i = 0; i < 160; i++)
        {
            int lo = rng.Next(int.MinValue, int.MaxValue);
            int mid = i % 3 == 0 ? 0 : rng.Next(int.MinValue, int.MaxValue);
            int hi = i % 5 == 0 ? rng.Next(int.MinValue, int.MaxValue) : 0;
            byte scale = (byte)rng.Next(0, 29);
            v.Add(new decimal(lo, mid, hi, rng.Next(2) == 1, scale));
        }

        return Dedup(v);
    }

    /// <summary>
    /// Removes duplicates by representation rather than by value: 1.0m and 1.00m
    /// are equal decimals but different bit patterns, and both are wanted here.
    /// </summary>
    static decimal[] Dedup(IEnumerable<decimal> v)
    {
        return v.GroupBy(d => string.Join(',', decimal.GetBits(d)))
                .Select(g => g.First())
                .ToArray();
    }
}
