package decimal

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeDecimal(t *testing.T, raw string) (Decimal, bool) {
	var d Decimal
	var ok bool
	ok = assert.NotPanics(t, func() { d = NewFromString(raw) })
	return d, ok
}

func makeInt(t *testing.T, raw string) (int, bool) {
	i, err := strconv.ParseInt(raw, 10, 64)
	return int(i), assert.NoError(t, err)
}

func makeInt32(t *testing.T, raw string) (int32, bool) {
	i, err := strconv.ParseInt(raw, 10, 32)
	return int32(i), assert.NoError(t, err)
}

func makeInt64(t *testing.T, raw string) (int64, bool) {
	i, err := strconv.ParseInt(raw, 10, 64)
	return i, assert.NoError(t, err)
}

func makeUint(t *testing.T, raw string) (uint, bool) {
	i, err := strconv.ParseUint(raw, 10, 64)
	return uint(i), assert.NoError(t, err)
}

func makeUint32(t *testing.T, raw string) (uint32, bool) {
	i, err := strconv.ParseUint(raw, 10, 32)
	return uint32(i), assert.NoError(t, err)
}

func makeUint64(t *testing.T, raw string) (uint64, bool) {
	i, err := strconv.ParseUint(raw, 10, 64)
	return i, assert.NoError(t, err)
}

func makeFloat32(t *testing.T, raw string) (float32, bool) {
	f, err := strconv.ParseFloat(raw, 32)
	return float32(f), assert.NoError(t, err)
}

func makeFloat64(t *testing.T, raw string) (float64, bool) {
	f, err := strconv.ParseFloat(raw, 64)
	return f, assert.NoError(t, err)
}

func makeBool(t *testing.T, raw string) (bool, bool) {
	b, err := strconv.ParseBool(raw)
	return b, assert.NoError(t, err)
}

func getRecs(t *testing.T, name string) <-chan []string {
	recs := make(chan []string)
	go func(t *testing.T, name string, recs chan []string) {
		defer close(recs)
		// Iterate the test cases
		f, err := os.Open(os.ExpandEnv(name))
		if !assert.NoError(t, err) {
			return
		}
		defer f.Close()

		r := bufio.NewScanner(f)

		var rec []string
		for r.Scan() {
			rec = strings.Split(r.Text(), "\t")
			if len(rec) == 0 {
				continue
			}
			recs <- rec
		}
	}(t, name, recs)
	return recs
}

func makeBinaryDecimal(t *testing.T, rec []string) (a, b, exp Decimal, panics, ok bool) {
	if a, ok = makeDecimal(t, rec[3]); !ok {
		return
	}
	if b, ok = makeDecimal(t, rec[4]); !ok {
		return
	}
	if panics, ok = makeBool(t, rec[1]); ok {
		if exp, ok = makeDecimal(t, rec[2]); !ok {
			return
		}
	}
	return
}

func makeUnaryDecimal(t *testing.T, rec []string) (a, exp Decimal, panics, ok bool) {
	if a, ok = makeDecimal(t, rec[3]); !ok {
		return
	}
	if panics, ok = makeBool(t, rec[1]); ok {
		if exp, ok = makeDecimal(t, rec[2]); !ok {
			return
		}
	}
	return
}

func makeBinaryBool(t *testing.T, rec []string) (a, b Decimal, exp bool, panics, ok bool) {
	if a, ok = makeDecimal(t, rec[3]); !ok {
		return
	}
	if b, ok = makeDecimal(t, rec[4]); !ok {
		return
	}
	if panics, ok = makeBool(t, rec[1]); ok {
		if exp, ok = makeBool(t, rec[2]); !ok {
			return
		}
	}
	return
}

func makeUnaryBool(t *testing.T, rec []string) (a Decimal, exp bool, panics, ok bool) {
	if a, ok = makeDecimal(t, rec[3]); !ok {
		return
	}
	if panics, ok = makeBool(t, rec[1]); ok {
		if exp, ok = makeBool(t, rec[2]); !ok {
			return
		}
	}
	return
}

func makeBinaryInt(t *testing.T, rec []string) (a, b Decimal, exp int, panics, ok bool) {
	if a, ok = makeDecimal(t, rec[3]); !ok {
		return
	}
	if b, ok = makeDecimal(t, rec[4]); !ok {
		return
	}
	if panics, ok = makeBool(t, rec[1]); ok {
		if exp, ok = makeInt(t, rec[2]); !ok {
			return
		}
	}
	return
}

func makeUnaryInt(t *testing.T, rec []string) (a Decimal, exp int, panics, ok bool) {
	if a, ok = makeDecimal(t, rec[3]); !ok {
		return
	}
	if panics, ok = makeBool(t, rec[1]); ok {
		if exp, ok = makeInt(t, rec[2]); !ok {
			return
		}
	}
	return
}

func makeUnaryInt32(t *testing.T, rec []string) (a Decimal, exp int32, panics, ok bool) {
	if a, ok = makeDecimal(t, rec[3]); !ok {
		return
	}
	if panics, ok = makeBool(t, rec[1]); ok {
		if exp, ok = makeInt32(t, rec[2]); !ok {
			return
		}
	}
	return
}

func makeUnaryInt64(t *testing.T, rec []string) (a Decimal, exp int64, panics, ok bool) {
	if a, ok = makeDecimal(t, rec[3]); !ok {
		return
	}
	if panics, ok = makeBool(t, rec[1]); ok {
		if exp, ok = makeInt64(t, rec[2]); !ok {
			return
		}
	}
	return
}

func makeUnaryUint(t *testing.T, rec []string) (a Decimal, exp uint, panics, ok bool) {
	if a, ok = makeDecimal(t, rec[3]); !ok {
		return
	}
	if panics, ok = makeBool(t, rec[1]); ok {
		if exp, ok = makeUint(t, rec[2]); !ok {
			return
		}
	}
	return
}

func makeUnaryUint32(t *testing.T, rec []string) (a Decimal, exp uint32, panics, ok bool) {
	if a, ok = makeDecimal(t, rec[3]); !ok {
		return
	}
	if panics, ok = makeBool(t, rec[1]); ok {
		if exp, ok = makeUint32(t, rec[2]); !ok {
			return
		}
	}
	return
}

func makeUnaryUint64(t *testing.T, rec []string) (a Decimal, exp uint64, panics, ok bool) {
	if a, ok = makeDecimal(t, rec[3]); !ok {
		return
	}
	if panics, ok = makeBool(t, rec[1]); ok {
		if exp, ok = makeUint64(t, rec[2]); !ok {
			return
		}
	}
	return
}

func makeUnaryFloat32(t *testing.T, rec []string) (a Decimal, exp float32, panics, ok bool) {
	if a, ok = makeDecimal(t, rec[3]); !ok {
		return
	}
	if panics, ok = makeBool(t, rec[1]); ok {
		if exp, ok = makeFloat32(t, rec[2]); !ok {
			return
		}
	}
	return
}

func makeUnaryFloat64(t *testing.T, rec []string) (a Decimal, exp float64, panics, ok bool) {
	if a, ok = makeDecimal(t, rec[3]); !ok {
		return
	}
	if panics, ok = makeBool(t, rec[1]); ok {
		if exp, ok = makeFloat64(t, rec[2]); !ok {
			return
		}
	}
	return
}

func makeUnaryRound(t *testing.T, rec []string) (a Decimal, mode RoundingMode, digits int32, exp Decimal, panics, ok bool) {
	if a, ok = makeDecimal(t, rec[3]); !ok {
		return
	}
	var tmp int32
	if tmp, ok = makeInt32(t, rec[4]); !ok {
		return
	}
	digits = tmp
	if tmp, ok = makeInt32(t, rec[4]); !ok {
		return
	}
	mode = RoundingMode(tmp)

	if panics, ok = makeBool(t, rec[1]); ok {
		if exp, ok = makeDecimal(t, rec[2]); !ok {
			return
		}
	}
	return
}

func TestDotnetAbs(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_abs.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, exp, panics, ok := makeUnaryDecimal(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.Abs() })
				} else {
					var act Decimal
					if assert.NotPanics(t, func() { act = a.Abs() }) {
						assert.True(t, act.Equal(exp))
					}
				}
			}
		})
	}
}

func TestDotnetCeil(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_ceil.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, exp, panics, ok := makeUnaryDecimal(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.Ceil() })
				} else {
					var act Decimal
					if assert.NotPanics(t, func() { act = a.Ceil() }) {
						assert.True(t, act.Equal(exp))
					}
				}
			}
		})
	}
}

func TestDotnetFloor(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_floor.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, exp, panics, ok := makeUnaryDecimal(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.Floor() })
				} else {
					var act Decimal
					if assert.NotPanics(t, func() { act = a.Floor() }) {
						assert.True(t, act.Equal(exp))
					}
				}
			}
		})
	}
}

func TestDotnetNeg(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_neg.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, exp, panics, ok := makeUnaryDecimal(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.Neg() })
				} else {
					var act Decimal
					if assert.NotPanics(t, func() { act = a.Neg() }) {
						assert.True(t, act.Equal(exp))
					}
				}
			}
		})
	}
}

func TestDotnetTruncate(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_truncate.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, exp, panics, ok := makeUnaryDecimal(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.Truncate() })
				} else {
					var act Decimal
					if assert.NotPanics(t, func() { act = a.Truncate() }) {
						assert.True(t, act.Equal(exp))
					}
				}
			}
		})
	}
}

func TestDotnetAdd(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_add.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, b, exp, panics, ok := makeBinaryDecimal(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.Add(b) })
				} else {
					var act Decimal
					if assert.NotPanics(t, func() { act = a.Add(b) }) {
						assert.True(t, act.Equal(exp))
					}
				}
			}
		})
	}
}

func TestDotnetSub(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_sub.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, b, exp, panics, ok := makeBinaryDecimal(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.Sub(b) })
				} else {
					var act Decimal
					if assert.NotPanics(t, func() { act = a.Sub(b) }) {
						assert.True(t, act.Equal(exp))
					}
				}
			}
		})
	}
}

func TestDotnetMul(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_mul.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, b, exp, panics, ok := makeBinaryDecimal(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.Mul(b) })
				} else {
					var act Decimal
					if assert.NotPanics(t, func() { act = a.Mul(b) }) {
						assert.True(t, act.Equal(exp))
					}
				}
			}
		})
	}
}

func TestDotnetDiv(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_div.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, b, exp, panics, ok := makeBinaryDecimal(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.Div(b) })
				} else {
					var act Decimal
					if assert.NotPanics(t, func() { act = a.Div(b) }) {
						assert.True(t, act.Equal(exp))
					}
				}
			}
		})
	}
}

func TestDotnetRem(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_rem.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, b, exp, panics, ok := makeBinaryDecimal(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.Rem(b) })
				} else {
					var act Decimal
					if assert.NotPanics(t, func() { act = a.Rem(b) }) {
						assert.True(t, act.Equal(exp))
					}
				}
			}
		})
	}
}

func TestDotnetEqual(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_equal.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, b, exp, panics, ok := makeBinaryBool(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.Equal(b) })
				} else {
					var act bool
					if assert.NotPanics(t, func() { act = a.Equal(b) }) {
						assert.Equal(t, exp, act)
					}
				}
			}
		})
	}
}

func TestDotnetGreaterThan(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_gt.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, b, exp, panics, ok := makeBinaryBool(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.GreaterThan(b) })
				} else {
					var act bool
					if assert.NotPanics(t, func() { act = a.GreaterThan(b) }) {
						assert.Equal(t, exp, act)
					}
				}
			}
		})
	}
}

func TestDotnetGreaterThanOrEqual(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_gte.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, b, exp, panics, ok := makeBinaryBool(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.GreaterThanOrEqual(b) })
				} else {
					var act bool
					if assert.NotPanics(t, func() { act = a.GreaterThanOrEqual(b) }) {
						assert.Equal(t, exp, act)
					}
				}
			}
		})
	}
}

func TestDotnetLessThan(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_lt.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, b, exp, panics, ok := makeBinaryBool(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.LessThan(b) })
				} else {
					var act bool
					if assert.NotPanics(t, func() { act = a.LessThan(b) }) {
						assert.Equal(t, exp, act)
					}
				}
			}
		})
	}
}

func TestDotnetLessThanOrEquall(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_lte.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, b, exp, panics, ok := makeBinaryBool(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.LessThanOrEqual(b) })
				} else {
					var act bool
					if assert.NotPanics(t, func() { act = a.LessThanOrEqual(b) }) {
						assert.Equal(t, exp, act)
					}
				}
			}
		})
	}
}

func TestDotnetIsNegative(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_isneg.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, exp, panics, ok := makeUnaryBool(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.IsNegative() })
				} else {
					var act bool
					if assert.NotPanics(t, func() { act = a.IsNegative() }) {
						assert.Equal(t, exp, act)
					}
				}
			}
		})
	}
}

func TestDotnetIsZero(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_iszero.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, exp, panics, ok := makeUnaryBool(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.IsZero() })
				} else {
					var act bool
					if assert.NotPanics(t, func() { act = a.IsZero() }) {
						assert.Equal(t, exp, act)
					}
				}
			}
		})
	}
}

func TestDotnetCmp(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_cmp.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, b, exp, panics, ok := makeBinaryInt(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.Equal(b) })
				} else {
					var act int
					if assert.NotPanics(t, func() { act = a.Cmp(b) }) {
						assert.Equal(t, exp, act)
					}
				}
			}
		})
	}
}

func TestInt32(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_int32.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, exp, panics, ok := makeUnaryInt32(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.ToInt32() })
				} else {
					var act int32
					if assert.NotPanics(t, func() { act = a.ToInt32() }) {
						assert.Equal(t, exp, act)
					}
				}
			}
		})
	}
}

func TestInt64(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_int64.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, exp, panics, ok := makeUnaryInt64(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.ToInt64() })
				} else {
					var act int64
					if assert.NotPanics(t, func() { act = a.ToInt64() }) {
						assert.Equal(t, exp, act)
					}
				}
			}
		})
	}
}

func TestUint32(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_uint32.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, exp, panics, ok := makeUnaryUint32(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.ToUint32() })
				} else {
					var act uint32
					if assert.NotPanics(t, func() { act = a.ToUint32() }) {
						assert.Equal(t, exp, act)
					}
				}
			}
		})
	}
}

func TestUint64(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_uint64.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, exp, panics, ok := makeUnaryUint64(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.ToUint64() })
				} else {
					var act uint64
					if assert.NotPanics(t, func() { act = a.ToUint64() }) {
						assert.Equal(t, exp, act)
					}
				}
			}
		})
	}
}

func TestFloat32(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_float32.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, exp, panics, ok := makeUnaryFloat32(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.ToFloat32() })
				} else {
					var act float32
					if assert.NotPanics(t, func() { act = a.ToFloat32() }) {
						assert.Equal(t, exp, act)
					}
				}
			}
		})
	}
}

func TestFloat64(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_float64.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, exp, panics, ok := makeUnaryFloat64(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.ToFloat64() })
				} else {
					var act float64
					if assert.NotPanics(t, func() { act = a.ToFloat64() }) {
						assert.Equal(t, exp, act)
					}
				}
			}
		})
	}
}

func TestDotnetRound(t *testing.T) {
	recs := getRecs(t, "./testdata/dotnet_round.txt")
	for rec := range recs {
		t.Run(rec[0], func(t *testing.T) {
			a, mode, digits, exp, panics, ok := makeUnaryRound(t, rec)
			if ok {
				if panics {
					assert.Panics(t, func() { a.Round(digits, mode) })
				} else {
					var act Decimal
					if assert.NotPanics(t, func() { act = a.Round(digits, mode) }) {
						assert.True(t, act.Equal(exp))
					}
				}
			}
		})
		break
	}
}
