package decimal

import (
	"math"
	"unsafe"

	"github.com/klokare/decimal/internal/platform"
)

// Constants ...
const (
	signMask   uint32 = 0x80000000
	scaleMask  uint32 = 0x00FF0000
	scaleShift int32  = 16 // not in Decimal.cs but necessary here to have no circular dependency with dec package

	decScaleMax int32 = 28

	tenToPowerNine     uint32 = 1000000000
	tenToPowerEighteen uint64 = 1000000000000000000

	// The maximum power of 10 that a 32 bit integer can store
	maxInt32Scale int32 = 9

	// The maximum power of 10 that a 64 bit integer can store
	maxInt64Scale int32 = 19
)

// Variables ...
var (
	// Fast access for 10^n where n is 0-9
	sPowers10 = [10]uint32{
		1,
		10,
		100,
		1000,
		10000,
		100000,
		1000000,
		10000000,
		100000000,
		1000000000,
	}

	// Fast access for 10^n where n is 1-19
	sUlongPowers10 = [19]uint64{
		10,
		100,
		1000,
		10000,
		100000,
		1000000,
		10000000,
		100000000,
		1000000000,
		10000000000,
		100000000000,
		1000000000000,
		10000000000000,
		100000000000000,
		1000000000000000,
		10000000000000000,
		100000000000000000,
		1000000000000000000,
		10000000000000000000,
	}

	sDoublePowers10 = [81]float64{
		1, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9,
		1e10, 1e11, 1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19,
		1e20, 1e21, 1e22, 1e23, 1e24, 1e25, 1e26, 1e27, 1e28, 1e29,
		1e30, 1e31, 1e32, 1e33, 1e34, 1e35, 1e36, 1e37, 1e38, 1e39,
		1e40, 1e41, 1e42, 1e43, 1e44, 1e45, 1e46, 1e47, 1e48, 1e49,
		1e50, 1e51, 1e52, 1e53, 1e54, 1e55, 1e56, 1e57, 1e58, 1e59,
		1e60, 1e61, 1e62, 1e63, 1e64, 1e65, 1e66, 1e67, 1e68, 1e69,
		1e70, 1e71, 1e72, 1e73, 1e74, 1e75, 1e76, 1e77, 1e78, 1e79,
		1e80,
	}
)

// region Decimal Math Helpers

// getExponent32 ...
func getExponent32(f float32) uint32 {
	return uint32(byte(*(*uint32)(unsafe.Pointer(&f)) >> 23))
}

// getExponent64 ...
func getExponent64(f float64) uint32 {
	return uint32((*(*uint64)(unsafe.Pointer(&f)) >> 52) & 0x7FF)
}

// uInt32x32To64 ...
func uInt32x32To64(a, b uint32) uint64 { return uint64(a) * uint64(b) }

// uInt64x64To128 ...
func uInt64x64To128(a, b uint64, result *Decimal) {
	var low = uInt32x32To64(uint32(a), uint32(b))     // lo partial prod
	var mid = uInt32x32To64(uint32(a), uint32(b>>32)) // mid 1 partial prod
	var high = uInt32x32To64(uint32(a>>32), uint32(b>>32))
	high += mid >> 32
	mid <<= 32
	low += mid
	if low < mid { // test for carry
		high++
	}

	mid = uInt32x32To64(uint32(a>>32), uint32(b))
	high += mid >> 32
	mid <<= 32
	low += mid
	if low < mid { // test for carry
		high++
	}

	if high > math.MaxUint32 {
		panic("decimal overflow exception")
	}
	result.setLow64(low)
	result.high = uint32(high)
}

// div96By32 does full divide, yielding 96-bit result and 32-bit remainder.
func div96By32(bufNum *buf12, den uint32) uint32 {
	// TODO: https://github.com/dotnet/coreclr/issues/3439

	var tmp, div uint64
	if bufNum.U2 != 0 {
		tmp = bufNum.High64()
		div = tmp / uint64(den)
		bufNum.SetHigh64(div)
		tmp = ((tmp - uint64(uint32(div))*uint64(den)) << 32) | uint64(bufNum.U0)
		if tmp == 0 {
			return 0
		}
		var div32 uint32 = uint32(tmp / uint64(den))
		bufNum.U0 = div32
		return uint32(tmp) - div32*den
	}

	tmp = bufNum.Low64()
	if tmp == 0 {
		return 0
	}
	div = tmp / uint64(den)
	bufNum.SetLow64(div)
	return uint32(tmp - div*uint64(den))
}

// div96ByConst ...
func div96ByConst(high64 *uint64, low *uint32, pow uint32) bool {

	if platform.Bits64 {
		var div64 uint64 = *high64 / uint64(pow)
		var div uint32 = uint32(((((*high64 - div64*uint64(pow)) << 32) + uint64(*low)) / uint64(pow)))
		if *low == div*pow {
			*high64 = div64
			*low = div
			return true
		}
	}

	// DOTNET: 32-bit RyuJIT doesn't convert 64-bit division by constant into multiplication by reciprocal. Do half-width divisions instead.
	var num, mid32, low16, div uint32
	if *high64 <= math.MaxUint32 {
		num = uint32(*high64)
		mid32 = num / pow
		num = (num - mid32*pow) << 16

		num += *low >> 16
		low16 = num / pow
		num = (num - low16*pow) << 16

		num += uint32(uint16(*low))
		div = num / pow
		if num == div*pow {
			*high64 = uint64(mid32)
			*low = (low16 << 16) + div
			return true
		}
	} else {
		num = uint32(*high64 >> 32)
		var high32 uint32 = num / pow
		num = (num - high32*pow) << 16

		num += uint32(*high64) >> 16
		mid32 = num / pow
		num = (num - mid32*pow) << 16

		num += uint32(uint16(*high64))
		div = num / pow
		num = (num - div*pow) << 16
		mid32 = div + (mid32 << 16)

		num += *low >> 16
		low16 = num / pow
		num = (num - low16*pow) << 16

		num += uint32(uint16(*low))
		div = num / pow
		if num == div*pow {
			*high64 = (uint64(high32) << 32) | uint64(mid32)
			*low = (low16 << 16) + div
			return true
		}
	}
	return false
}

// calcUnscale normalizes (unscale) the number by trying to divide out 10^8, 10^4, 10^2, and 10^1.
// If a division by one of these powers returns a zero remainder, then we keep the quotient.
func calcUnscale(low *uint32, high64 *uint64, scale *int32) {
	// Since 10 = 2 * 5, there must be a factor of 2 for every power of 10 we can extract.
	// We use this as a quick test on whether to try a given power.
	if platform.Bits64 {
		for byte(*low) == 0 && *scale >= 8 && div96ByConst(high64, low, 100000000) {
			*scale -= 8
		}

		if (*low&0xF) == 0 && *scale > 4 && div96ByConst(high64, low, 10000) {
			*scale -= 4
		}

	} else {
		for (*low&0xF) == 0 && *scale > 4 && div96ByConst(high64, low, 10000) {
			*scale -= 4
		}
	}

	if (*low&3) == 0 && *scale > 2 && div96ByConst(high64, low, 100) {
		*scale -= 2
	}

	if (*low&1) == 0 && *scale > 1 && div96ByConst(high64, low, 10) {
		*scale--
	}
}

// div96By64 does  partial divide, yielding 32-bit result and 64-bit remainder.
// Divisor must be larger than upper 64 bits of dividend.
func div96By64(bufNum *buf12, den uint64) uint32 {

	var quo uint32
	var num uint64
	var num2 uint32 = bufNum.U2
	if num2 == 0 {
		num = bufNum.Low64()
		if num < den {
			// Result is zero. Entire dividend is remainder
			return 0
		}

		// TODO: https://github.com/dotnet/coreclr/issues/3439
		quo = uint32(num / den)
		num -= uint64(quo) * den // remainder
		bufNum.SetLow64(num)
		return quo
	}

	var denHigh32 uint32 = uint32(den >> 32)
	if num2 >= denHigh32 {
		// Divide would overflow.  Assume a quotient of 2^32, and set
		// up remainder accordingly.
		num = bufNum.Low64()
		num -= den << 32
		quo = 0

		// Remainder went negative.  Add divisor back in until it's positive,
		// a max of 2 times.
		for ok := true; ok; ok = num >= den {
			quo--
			num += den
		}

		bufNum.SetLow64(num)
		return quo
	}

	// Hardware divide won't overflow
	var num64 uint64 = bufNum.High64()
	if num64 < uint64(denHigh32) {
		// Result is zero. Entirre dividend is remainder.
		return 0
	}

	// TODO: https://github.com/dotnet/coreclr/issues/3439
	quo = uint32(num64 / uint64(denHigh32))
	num = uint64(bufNum.U0) | ((num64 - uint64(quo)*uint64(denHigh32)) << 32) // remainder

	// Compute full remainder, rem = dividend - (quo * divisor).
	var prod uint64 = uInt32x32To64(quo, uint32(den)) // quo * lo divisor
	num -= prod
	if num > ^prod {
		// Remainder went negative.  Add divisor back in until it's positive,
		// a max of 2 times.
		for ok := true; ok; ok = num >= den {
			quo--
			num += den
		}
	}

	bufNum.SetLow64(num)
	return quo
}

// div128By96 does partial divide, yielding 32-bit result and 96-bit remainder.
// Top divisor uint must be larger than top dividend uint. This is
// assured in the initial call because the divisor is normalized
// and the dividend can't be. In subsequent calls, the remainder
// is multiplied by 10^9 (max), so it can be no more than 1/4 of
// the divisor which is effectively multiplied by 2^32 (4 * 10^9).
func div128By96(bufNum *buf16, bufDen *buf12) uint32 {
	var dividend uint64 = bufNum.High64()
	var den uint32 = bufDen.U2
	if dividend < uint64(den) {
		// Result is zero.  Entire dividend is remainder.
		return 0
	}

	// TODO: https://github.com/dotnet/coreclr/issues/3439
	var quo uint32 = uint32(dividend / uint64(den))
	var remainder uint32 = uint32(dividend) - quo*den

	// Compute full remainder, rem = dividend - (quo * divisor).
	var prod1 uint64 = uInt32x32To64(quo, bufDen.U0) // quo * lo divisor
	var prod2 uint64 = uInt32x32To64(quo, bufDen.U1) // quo * mid divisor
	prod2 += prod1 >> 32
	prod1 = uint64(uint32(prod1)) | (prod2 << 32)
	prod2 >>= 32

	var num uint64 = bufNum.Low64()
	num -= prod1
	remainder -= uint32(prod2)

	// Propagate carries
	if num > ^prod1 {
		remainder--
		if remainder < ^uint32(prod2) {
			goto PostRem
		}
	} else if remainder <= ^uint32(prod2) {
		goto PostRem
	}

	// Remainder went negative.  Add divisor back in until it's positive,
	// a max of 2 times.
	prod1 = bufDen.Low64()

	for {
		quo--
		num += prod1
		remainder += den

		if num < prod1 {
			// Detected carry. Check for carry out of top
			// before adding it in.
			if remainder++; remainder < den {
				break
			}
		}
		if remainder < den {
			break // detected carry
		}
	}

PostRem:
	bufNum.SetLow64(num)
	bufNum.U2 = remainder
	return quo
}

// increaseScale multiplies the two numbers. The low 96 bits of the result overwrite
// the input. The last 32 bits of the product are the return value.
func increaseScale(bufNum *buf12, power uint32) uint32 {
	var tmp uint64 = uInt32x32To64(bufNum.U0, power)
	bufNum.U0 = uint32(tmp)
	tmp >>= 32
	tmp += uInt32x32To64(bufNum.U1, power)
	bufNum.U1 = uint32(tmp)
	tmp >>= 32
	tmp += uInt32x32To64(bufNum.U2, power)
	bufNum.U2 = uint32(tmp)
	return uint32(tmp >> 32)
}

// increaseScale64 ...
func increaseScale64(bufNum *buf12, power uint32) {
	var tmp uint64 = uInt32x32To64(bufNum.U0, power)
	bufNum.U0 = uint32(tmp)
	tmp >>= 32
	tmp += uInt32x32To64(bufNum.U1, power)
	bufNum.SetHigh64(tmp)
}

// scaleResult sees if we need to scale the result to fit it in 96 bits.
// Perform needed scaling. Adjust scale factor accordingly.
func scaleResult(bufRes *buf24, hiRes uint32, scale int32) int32 {
	var result *[6]uint32 = (*[6]uint32)(unsafe.Pointer(bufRes))

	// See if we need to scale the result.  The combined scale must
	// be <= DEC_SCALE_MAX and the upper 96 bits must be zero.
	//
	// Start by figuring a lower bound on the scaling needed to make
	// the upper 96 bits zero.  hiRes is the index into result[]
	// of the highest non-zero uint.
	var newScale int32
	if hiRes > 2 {
		newScale = int32(hiRes)*32 - 64 - 1
		newScale -= leadingZeroCount(result[hiRes])

		// Multiply bit position by log10(2) to figure it's power of 10.
		// We scale the log by 256.  log(2) = .30103, * 256 = 77.  Doing this
		// with a multiply saves a 96-byte lookup table.  The power returned
		// is <= the power of the number, so we must add one power of 10
		// to make it's integer part zero after dividing by 256.
		//
		// Note: the result of this multiplication by an approximation of
		// log10(2) have been exhaustively checked to verify it gives the
		// correct result.  (There were only 95 to check...)
		newScale = ((newScale * 77) >> 8) + 1

		// newScale = min scale factor to make high 96 bits zero, 0 - 29.
		// This reduces the scale factor of the result.  If it exceeds the
		// current scale of the result, we'll overflow.
		//
		if newScale > scale {
			panic("decimal overflow exception")
		}
	}

	// Make sure we scale by enough to bring the current scale factor
	// into valid range.
	if newScale < scale-decScaleMax {
		newScale = scale - decScaleMax
	}

	if newScale != 0 {
		// Scale by the power of 10 given by newScale.  Note that this is
		// NOT guaranteed to bring the number within 96 bits -- it could
		// be 1 power of 10 short.
		scale -= newScale
		var sticky uint32
		var quotient, remainder uint32

		for {
			sticky |= remainder // record remainder as sticky bit

			var power uint32
			// Scaling loop specialized for each power of 10 because division by constant is an order of magnitude faster (especially for 64-bit division that's actually done by 128bit DIV on x64)
			if platform.Bits64 {
				switch newScale {
				case 1:
					power, quotient, remainder = divByConst(result, hiRes, 10)
				case 2:
					power, quotient, remainder = divByConst(result, hiRes, 100)
				case 3:
					power, quotient, remainder = divByConst(result, hiRes, 1000)
				case 4:
					power, quotient, remainder = divByConst(result, hiRes, 10000)
				case 5:
					power, quotient, remainder = divByConst(result, hiRes, 100000)
				case 6:
					power, quotient, remainder = divByConst(result, hiRes, 1000000)
				case 7:
					power, quotient, remainder = divByConst(result, hiRes, 10000000)
				case 8:
					power, quotient, remainder = divByConst(result, hiRes, 100000000)
				default:
					power, quotient, remainder = divByConst(result, hiRes, tenToPowerNine)
				}
			} else {
				switch newScale {
				case 1:
					power, quotient, remainder = divByConst(result, hiRes, 10)
				case 2:
					power, quotient, remainder = divByConst(result, hiRes, 100)
				case 3:
					power, quotient, remainder = divByConst(result, hiRes, 1000)
				case 4:
					power, quotient, remainder = divByConst(result, hiRes, 10000)
				default:
					// goto case 4 <- not possible in Go just using case 4's code
					power, quotient, remainder = divByConst(result, hiRes, 10000)
				}
			}
			result[hiRes] = quotient
			// If first quotient was 0, update hiRes.
			if quotient == 0 && hiRes != 0 {
				hiRes--
			}

			if platform.Bits64 {
				newScale -= maxInt32Scale
			} else {
				newScale -= 4
			}

			if newScale > 0 {
				continue // scale some more
			}

			// If we scaled enough, hiRes would be 2 or less.  If not,
			// divide by 10 more.
			if hiRes > 2 {
				if scale == 0 {
					panic("decimal overflow exception")
				}
				newScale = 1
				scale--
				continue // scale by 10
			}

			// Round final result.  See if remainder >= 1/2 of divisor.
			// If remainder == 1/2 divisor, round up if odd or sticky bit set.
			power >>= 1 // power of 10 always even
			if power <= remainder && (power < remainder || ((result[0]&1)|sticky) != 0) {
				if result[0]++; result[0] == 0 {
					var cur uint32
					for ok := true; ok; ok = result[cur] == 0 {
						cur++
						result[cur]++
					}

					if cur > 2 {
						// The rounding caused us to carry beyond 96 bits.
						// Scale by 10 more.
						if scale == 0 {
							panic("decimal overflow exception")
						}
						hiRes = cur
						sticky = 0    // no sticky bit
						remainder = 0 // or remainder
						newScale = 1
						scale--
						continue // scale by 10
					}
				}
			}

			break
		} // for{}
	}
	return scale
}

// divByConst ...
func divByConst(result *[6]uint32, hiRes uint32, power uint32) (uint32, uint32, uint32) {
	var high uint32 = result[hiRes]
	quotient := high / power
	remainder := high - quotient*power
	for i := hiRes - 1; int32(i) >= 0; i-- {
		if platform.Bits64 {
			var num uint64 = uint64(result[i]) + (uint64(remainder) << 32)
			result[i] = uint32(num / uint64(power))
			remainder = uint32(num) - result[i]*power
		} else {
			// TODO: work on this as the C# gets a bit unsafey with pointers to 8 and 16 bit
			// DOTNET: 32-bit RyuJIT doesn't convert 64-bit division by constant into multiplication by reciprocal. Do half-width divisions instead.
			// var low16, high16 int32
			// if platform.LittleEndian {
			// 	low16 = 0
			// 	high16 = 2
			// } else {
			// 	low16 = 2
			// 	high16 = 0
			// }
			// // byte* is used here because Roslyn doesn't do constant propagation for pointer arithmetic
			// var num uint32 = uint32(uint16(*(*byte)(unsafe.Pointer(result))[i+4+high16])) + (remainder << 16)
			// var div uint32 = num / power
			// remainder = num - div*power
			// *(*uint16)((*byte)(unsafe.Pointer(result))[i+4+high16]) = uint16(div)
			panic("not implemented for 32-bit")
		}
	}
	return power, quotient, remainder
}

// leadingZeroCount ...
func leadingZeroCount(value uint32) int32 {
	var c int32 = 1
	if (value & 0xFFFF0000) == 0 {
		value <<= 16
		c += 16
	}
	if (value & 0xFF000000) == 0 {
		value <<= 8
		c += 8
	}
	if (value & 0xF0000000) == 0 {
		value <<= 4
		c += 4
	}
	if (value & 0xC0000000) == 0 {
		value <<= 2
		c += 2
	}
	return c + (int32(value) >> 31)
}

// overflowUnscale adjusts the quotient to deal with an overflow.
// We need to divide by 10, feed in the high bit to undo the overflow and then round as required.
func overflowUnscale(bufQuo *buf12, scale int32, sticky bool) int32 {
	if scale--; scale < 0 {
		panic("decimal overflow exception")
	}

	// We have overflown, so load the high bit with a one.
	const highBit uint64 = 1 << 32
	bufQuo.U2 = uint32(highBit / 10)
	var tmp uint64 = ((highBit % 10) << 32) + uint64(bufQuo.U1)
	var div uint32 = (uint32(tmp / 10))
	bufQuo.U1 = div
	tmp = ((tmp - uint64(div)*10) << 32) + uint64(bufQuo.U0)
	div = uint32(tmp / 10)
	bufQuo.U0 = div
	var remainder uint32 = uint32(tmp - uint64(div)*10)
	// The remainder is the last digit that does not fit, so we can use it to work out if we need to round up
	if remainder > 5 || remainder == 5 && (sticky || ((bufQuo.U0&1) != 0)) {
		add32To96(bufQuo, 1)
	}
	return scale
}

// searchScale determines the max power of 10, &lt;= 9, that the quotient can be scaled
// up by and still fit in 96 bits.
func searchScale(bufQuo *buf12, scale int32) int32 {
	const OvflMax9Hi uint32 = 4
	const OvflMax8Hi uint32 = 42
	const OvflMax7Hi uint32 = 429
	const OvflMax6Hi uint32 = 4294
	const OvflMax5Hi uint32 = 42949
	const OvflMax4Hi uint32 = 429496
	const OvflMax3Hi uint32 = 4294967
	const OvflMax2Hi uint32 = 42949672
	const OvflMax1Hi uint32 = 429496729
	const OvflMax9MidLo uint64 = 5441186219426131129

	var resHi uint32 = bufQuo.U2
	var resMidLo uint64 = bufQuo.Low64()
	var curScale int32
	var powerOvfl = powerOvflValues

	// Quick check to stop us from trying to scale any more.
	if resHi > OvflMax1Hi {
		goto HaveScale
	}

	if scale > decScaleMax-9 {
		// We can't scale by 10^9 without exceeding the max scale factor.
		// See if we can scale to the max.  If not, we'll fall into
		// standard search for scale factor.
		curScale = decScaleMax - scale
		if resHi < powerOvfl[curScale-1].Hi {
			goto HaveScale
		}
	} else if resHi < OvflMax9Hi || resHi == OvflMax9Hi && resMidLo <= OvflMax9MidLo {
		return 9
	}

	// Search for a power to scale by < 9.  Do a binary search.
	if resHi > OvflMax5Hi {
		if resHi > OvflMax3Hi {
			curScale = 2
			if resHi > OvflMax2Hi {
				curScale--
			}
		} else {
			curScale = 4
			if resHi > OvflMax4Hi {
				curScale--
			}
		}
	} else {
		if resHi > OvflMax7Hi {
			curScale = 6
			if resHi > OvflMax6Hi {
				curScale--
			}
		} else {
			curScale = 8
			if resHi > OvflMax8Hi {
				curScale--
			}
		}
	}

	// In all cases, we already found we could not use the power one larger.
	// So if we can use this power, it is the biggest, and we're done.  If
	// we can't use this power, the one below it is correct for all cases
	// unless it's 10^1 -- we might have to go to 10^0 (no scaling).
	if resHi == powerOvfl[curScale-1].Hi && resMidLo > powerOvfl[curScale-1].MidLo {
		curScale--
	}

HaveScale:
	// curScale = largest power of 10 we can scale by without overflow,
	// curScale < 9.  See if this is enough to make scale factor
	// positive if it isn't already.
	if curScale+scale < 0 {
		panic("decimal overflow exception")
	}
	return curScale
}

// add32To96 adds a 32-bit uint to an array of 3 uints representing a 96-bit integer.
func add32To96(bufNum *buf12, value uint32) bool {
	if bufNum.SetLow64(bufNum.Low64() + uint64(value)); bufNum.Low64() < uint64(value) {
		if bufNum.U2++; bufNum.U2 == 0 {
			return false
		}
	}
	return true
}

func xor(a, b bool) bool {
	if a == b {
		return false
	}
	return a || b
}

// decAddSub adds or subtracts two decimal values.
// On return, d1 contains the result of the operation and d2 is trashed.
func decAddSub(d1, d2 *Decimal, sign bool) {
	var low64 uint64 = d1.low64()
	var high uint32 = d1.high
	var flags uint32 = d1.flags
	var d2flags uint32 = d2.flags

	var xorflags uint32 = d2flags ^ flags
	sign = xor(sign, (xorflags&signMask) != 0)

	// Declarations here to get around "goto ... jumps over variable declaration" errors in Go
	var tmpHigh, hiProd, d1flags, power uint32
	var bufNum *buf24
	var tmp64, tmpLow uint64
	var scale int32

	if (xorflags & scaleMask) == 0 {
		// Scale factors are equal, no alignment necessary.
		goto AlignedAdd
	}

	// Scale factors are not equal.  Assume that a larger scale
	// factor (more decimal places) is likely to mean that number
	// is smaller.  Start by guessing that the right operand has
	// the larger scale factor.  The result will have the larger
	// scale factor.
	d1flags = flags
	flags = d2flags&scaleMask | flags&signMask // scale factor of "smaller",  but sign of "larger"
	scale = int32(flags-d1flags) >> scaleShift

	if scale < 0 {
		// Guessed scale factor wrong. Swap operands
		scale = -scale
		flags = d1flags
		if sign {
			flags ^= signMask
		}
		low64 = d2.low64()
		high = d2.high
		d2 = d1
	}

	// d1 will need to be multiplied by 10^scale so
	// it will have the same scale as d2.  We could be
	// extending it to up to 192 bits of precision.

	// Scan for zeros in the upper words.
	if high == 0 {
		if low64 <= math.MaxUint32 {
			if uint32(low64) == 0 {
				// Left arg is zero, return right.
				var signFlags uint32 = flags & scaleMask
				if sign {
					signFlags ^= signMask
				}
				d1 = d2
				d1.flags = (d2.flags&scaleMask | signFlags)
				return
			}

			for ok := true; ok; ok = low64 <= math.MaxUint32 {
				if scale <= maxInt32Scale {
					low64 = uInt32x32To64(uint32(low64), sPowers10[scale])
					goto AlignedAdd
				}
				scale -= maxInt32Scale
				low64 = uInt32x32To64(uint32(low64), tenToPowerNine)
			}
		}

		for ok := true; ok; ok = high == 0 {
			power = tenToPowerNine
			if scale < maxInt32Scale {
				power = sPowers10[scale]
			}
			tmpLow = uInt32x32To64(uint32(low64), power)
			tmp64 = uInt32x32To64(uint32(low64>>32), power) + (tmpLow >> 32)
			low64 = uint64(uint32(tmpLow)) + (tmp64 << 32)
			high = uint32(tmp64 >> 32)
			if scale -= maxInt32Scale; scale <= 0 {
				goto AlignedAdd
			}
		}
	}

	for {
		// Scaling won't make it larget han 4 unit32s
		power = tenToPowerNine
		if scale < maxInt32Scale {
			power = sPowers10[scale]
		}
		tmpLow = uInt32x32To64(uint32(low64), power)
		tmp64 = uInt32x32To64(uint32(low64>>32), power) + (tmpLow >> 32)
		low64 = uint64(uint32(tmpLow)) + (tmp64 << 32)
		tmp64 >>= 32
		tmp64 += uInt32x32To64(high, power)

		scale -= maxInt32Scale
		if tmp64 > math.MaxUint32 {
			break
		}

		high = uint32(tmp64)
		// Result fits in 96 bits.  Use standard aligned add.
		if scale <= 0 {
			goto AlignedAdd
		}
	}

	// Have to scale by a bunch. Move the number to a buffer where it has room to grow as it's scaled.
	bufNum = new(buf24)
	bufNum.SetLow64(low64)
	bufNum.SetMid64(tmp64)
	hiProd = 3

	// Scaling loop, up to 10^9 at a time. hiProd stays updated with index of highest non-zero uint.
	for ; scale > 0; scale -= maxInt32Scale {
		power = tenToPowerNine
		if scale < maxInt32Scale {
			power = sPowers10[scale]
		}
		tmp64 = 0
		var rgulNum *[6]uint32 = (*[6]uint32)(unsafe.Pointer(bufNum))
		var cur uint32 = 0
		for {
			tmp64 += uInt32x32To64(rgulNum[cur], power)
			rgulNum[cur] = uint32(tmp64)
			cur++
			tmp64 >>= 32
			if cur > hiProd {
				break
			}
		}

		if uint32(tmp64) != 0 {
			// We're extending the result by another uint.
			hiProd++
			rgulNum[hiProd] = uint32(tmp64)
		}
	}

	// Scaling complete, do the add.  Could be subtract if signs differ.
	tmp64 = bufNum.Low64()
	low64 = d2.low64()
	tmpHigh = bufNum.U2
	high = d2.high

	if sign {
		// Signs differ, subtract
		low64 = tmp64 - low64
		high = tmpHigh - high

		// Propagate carry
		if low64 > tmp64 {
			high--
			if high < tmpHigh {
				goto NoCarry
			}
		} else if high <= tmpHigh {
			goto NoCarry
		}

		// Carry the subtraction into the higher bits
		var number *[6]uint32 = (*[6]uint32)(unsafe.Pointer(bufNum))
		var cur uint32 = 3
		for ok := true; ok; ok = number[cur-1] == 0 {
			number[cur]--
			cur++
		}
		if number[hiProd] == 0 {
			if hiProd--; hiProd <= 2 {
				goto ReturnResult
			}
		}
	} else {

		// Signs the same, add
		low64 += tmp64
		high += tmpHigh

		// Propagate carry
		if low64 < tmp64 {
			high++
			if high > tmpHigh {
				goto NoCarry
			}
		} else if high >= tmpHigh {
			goto NoCarry
		}

		var number *[6]uint32 = (*[6]uint32)(unsafe.Pointer(bufNum))
		var cur uint32 = 3
		var ok bool
		number[cur]++
		for ok, cur = number[cur] == 0, cur+1; ok; ok = number[cur] == 0 {
			if hiProd < cur {
				number[cur] = 1
				hiProd = cur
				break
			}
			number[cur]++
			cur++
		}
	}

NoCarry:
	{
		bufNum.SetLow64(low64)
		bufNum.U2 = high
		scale = scaleResult(bufNum, hiProd, int32(byte(flags>>scaleShift)))
		flags = (flags & ^scaleMask) | uint32(scale)<<scaleShift
		low64 = bufNum.Low64()
		high = bufNum.U2
		goto ReturnResult
	}

SignFlip:
	{
		// Got negative result. Flip its sign.
		flags ^= signMask
		high = ^high
		low64 = uint64(-int64(low64))
		if low64 == 0 {
			high++
		}
		goto ReturnResult
	}

AlignedScale:
	{
		// The addition carried above 96 bits.
		// Divide the value by 10, dropping the scale factor.
		if (flags & scaleMask) == 0 {
			panic("decimal overflow exception")
		}
		flags -= 1 << scaleShift

		const den uint32 = 10
		var num uint64 = uint64(high) + (1 << 32)
		high = uint32(num / uint64(den))
		num = ((num - uint64(high)*uint64(den)) << 32) + (low64 >> 32)
		var div = uint32(num / uint64(den))
		num = ((num - uint64(div)*uint64(den)) << 32) + uint64(uint32(low64))
		low64 = uint64(div)
		low64 <<= 32
		div = uint32(num / uint64(den))
		low64 += uint64(div)
		div = uint32(num - uint64(div)*uint64(den))

		// See if we need to round up.
		if div >= 5 && (div > 5 || (low64&1) != 0) {
			if low64++; low64 == 0 {
				high++
			}
		}
		goto ReturnResult
	}

AlignedAdd:
	{
		var d1Low64 uint64 = low64
		var d1High uint32 = high
		if sign {
			// Signs differ - subtract
			low64 = d1Low64 - d2.low64()
			high = d1High - d2.high

			// Propagate carry
			if low64 > d1Low64 {
				high--
				if high >= d1High {
					goto SignFlip
				}
			} else if high > d1High {
				goto SignFlip
			}
		} else {
			// Signs are the same - add
			low64 = d1Low64 + d2.low64()
			high = d1High + d2.high

			// Propagate carry
			if low64 < d1Low64 {
				high++
				if high <= d1High {
					goto AlignedScale
				}
			} else if high < d1High {
				goto AlignedScale
			}
		}
		goto ReturnResult
	}
ReturnResult:
	d1.flags = flags
	d1.high = high
	d1.setLow64(low64)
}

// varDecCmp Decimal Compare updated to return values similar to ICompareTo
func varDecCmp(d1, d2 Decimal) int32 {
	if d2.low|d2.mid|d2.high == 0 {
		if d1.low|d1.mid|d1.high == 0 {
			return 0
		}
		return (int32(d1.flags) >> 31) | 1
	}
	if d1.low|d1.mid|d1.high == 0 {
		return -((int32(d2.flags) >> 31) | 1)
	}

	var sign int32 = (int32(d1.flags) >> 31) - (int32(d2.flags) >> 31)
	if sign != 0 {
		return sign
	}
	return varDecCmpSub(d1, d2)
}

// varDecCmpSub ...
func varDecCmpSub(d1, d2 Decimal) int32 {
	var flags int32 = int32(d2.flags)
	var sign int32 = (flags >> 31) | 1
	var scale int32 = flags - int32(d1.flags)

	var low64 uint64 = d1.low64()
	var high uint32 = d1.high

	var d2Low64 uint64 = d2.low64()
	var d2High uint32 = d2.high

	if scale != 0 {
		scale >>= scaleShift

		// Scale factors are not equal. Assume that a larger scale factor (more decimal places) is likely to mean that number is smaller.
		// Start by guessing that the right operand has the larger scale factor.
		if scale < 0 {
			// Guessed scale factor wrong. Swap operands.
			scale = -scale
			sign = -sign
			low64, d2Low64 = d2Low64, low64
			high, d2High = d2High, high
		}

		// d1 will need to be multiplied by 10^scale so it will have the same scale as d2.
		// Scaling loop, up to 10^9 at a time.
		for ok := true; ok; ok = scale > 0 {
			var power uint32
			if scale >= maxInt32Scale {
				power = tenToPowerNine
			} else {
				power = sPowers10[scale]
			}
			var tmpLow uint64 = uInt32x32To64(uint32(low64), power)
			var tmp uint64 = uInt32x32To64(uint32(low64>>32), power) + (tmpLow >> 32)
			low64 = uint64(uint32(tmpLow)) + (tmp << 32)
			tmp >>= 32
			tmp += uInt32x32To64(high, power)
			// If the scaled value has more than 96 significant bits then it's greater than d2
			if tmp > math.MaxUint32 {
				return sign
			}
			high = uint32(tmp)
			scale -= maxInt32Scale
		}
	}

	var cmpHigh uint32 = high - d2High
	if cmpHigh != 0 {
		// check for overflow
		if cmpHigh > high {
			sign = -sign
		}
		return sign
	}

	var cmpLow64 uint64 = low64 - d2Low64
	if cmpLow64 == 0 {
		sign = 0
	} else if cmpLow64 > low64 { // check for overflow
		sign = -sign
	}
	return sign
}

// varDecMul Decimal Multiply
func varDecMul(d1, d2 *Decimal) {
	var scale int32 = int32(byte((d1.flags + d2.flags) >> scaleShift))

	var tmp uint64
	var hiProd uint32
	var bufProd *buf24 = new(buf24)
	var product *[6]uint32

	if d1.high|d1.mid == 0 {
		if d2.high|d2.mid == 0 {
			// Upper 64 bits are zero
			var low64 uint64 = uInt32x32To64(d1.low, d2.low)
			if scale > decScaleMax {
				// Result scale is too big.  Divide result by power of 10 to reduce it.
				// If the amount to divide by is > 19 the result is guaranteed
				// less than 1/2.  [max value in 64 bits = 1.84E19]
				if scale > decScaleMax+maxInt64Scale {
					goto ReturnZero
				}

				scale -= decScaleMax + 1
				var power uint64 = sUlongPowers10[scale]

				// TODO: https://github.com/dotnet/coreclr/issues/3439
				tmp = low64 / power
				var remainder uint64 = low64 - tmp*power
				low64 = tmp

				// Round result.  See if remainder >= 1/2 of divisor.
				// Divisor is a power of 10, so it is always even.
				power >>= 1
				if remainder >= power && (remainder > power || (uint32(low64)&1) > 0) {
					low64++
				}
				scale = decScaleMax
			}
			d1.setLow64(low64)
			d1.flags = ((d2.flags ^ d1.flags) & signMask) | (uint32(scale) << scaleShift)
			return
		}
		// Left value is 32 bit, result fits in 4 uint32s
		tmp = uInt32x32To64(d1.low, d2.low)
		bufProd.U0 = uint32(tmp)

		tmp = uInt32x32To64(d1.low, d2.mid) + (tmp >> 32)
		bufProd.U1 = uint32(tmp)
		tmp >>= 32

		if d2.high != 0 {
			tmp += uInt32x32To64(d1.low, d2.high)
			if tmp > math.MaxUint32 {
				bufProd.SetMid64(tmp)
				hiProd = 3
				goto SkipScan
			}
		}
		if uint32(tmp) != 0 {
			bufProd.U2 = uint32(tmp)
			hiProd = 2
			goto SkipScan
		}
		hiProd = 1

	} else if (d2.high | d2.mid) == 0 {

		// Right value is 32-bit, result fits in 4 uint32s
		tmp = uInt32x32To64(d2.low, d1.low)
		bufProd.U0 = uint32(tmp)

		tmp = uInt32x32To64(d2.low, d1.mid) + (tmp >> 32)
		bufProd.U1 = uint32(tmp)
		tmp >>= 32

		if d1.high != 0 {
			tmp += uInt32x32To64(d2.low, d1.high)
			if tmp > math.MaxUint32 {
				bufProd.SetMid64(tmp)
				hiProd = 3
				goto SkipScan
			}
		}
		if uint32(tmp) != 0 {
			bufProd.U2 = uint32(tmp)
			hiProd = 2
			goto SkipScan
		}
		hiProd = 1
	} else {

		// Both operands have bits set in the upper 64 bits.
		//
		// Compute and accumulate the 9 partial products into a
		// 192-bit (24-byte) result.
		//
		//        [l-h][l-m][l-l]      left high, middle, low
		//         x    [r-h][r-m][r-l]      right high, middle, low
		// ------------------------------
		//
		//             [0-h][0-l]      l-l * r-l
		//        [1ah][1al]      l-l * r-m
		//        [1bh][1bl]      l-m * r-l
		//       [2ah][2al]          l-m * r-m
		//       [2bh][2bl]          l-l * r-h
		//       [2ch][2cl]          l-h * r-l
		//      [3ah][3al]          l-m * r-h
		//      [3bh][3bl]          l-h * r-m
		// [4-h][4-l]              l-h * r-h
		// ------------------------------
		// [p-5][p-4][p-3][p-2][p-1][p-0]      prod[] array

		tmp = uInt32x32To64(d1.low, d2.low)
		bufProd.U0 = uint32(tmp)

		var tmp2 uint64 = uInt32x32To64(d1.low, d2.mid) + (tmp >> 32)

		tmp = uInt32x32To64(d1.mid, d2.low)
		tmp += tmp2 // this could generate carry
		bufProd.U1 = uint32(tmp)
		if tmp < tmp2 { // detect carry
			tmp2 = (tmp >> 32) | (1 << 32)
		} else {
			tmp2 = tmp >> 32
		}

		tmp = uInt32x32To64(d1.mid, d2.mid) + tmp2

		if d1.high|d2.high > 0 {
			// Highest 32 bits is non-zero.     Calculate 5 more partial products.
			tmp2 = uInt32x32To64(d1.low, d2.high)
			tmp += tmp2 // this could generate carry
			var tmp3 uint32 = 0
			if tmp < tmp2 { // detect carry
				tmp3 = 1
			}

			tmp2 = uInt32x32To64(d1.high, d2.low)
			tmp += tmp2 // this could generate carry
			bufProd.U2 = uint32(tmp)
			if tmp < tmp2 { // detect carrry
				tmp3++
			}
			tmp2 = (uint64(tmp3) << 32) | (tmp >> 32)

			tmp = uInt32x32To64(d1.mid, d2.high)
			tmp += tmp2 // this could generate carry
			tmp3 = 0
			if tmp < tmp2 { // detect carry
				tmp3 = 1
			}

			tmp2 = uInt32x32To64(d1.high, d2.mid)
			tmp += tmp2 // This could generate carry
			bufProd.U3 = uint32(tmp)
			if tmp < tmp2 { // detect carry
				tmp3++
			}
			tmp = (uint64(tmp3) << 32) | (tmp >> 32)

			bufProd.SetHigh64(uInt32x32To64(d1.high, d2.high) + tmp)

			hiProd = 5
		} else if tmp != 0 {
			bufProd.SetMid64(tmp)
			hiProd = 3
		} else {
			hiProd = 1
		}
	}

	// Check for leading zero uints on the product
	product = (*[6]uint32)(unsafe.Pointer(bufProd))
	for product[int32(hiProd)] == 0 {
		if hiProd == 0 {
			goto ReturnZero
		}
		hiProd--
	}

SkipScan:
	if hiProd > 2 || scale > decScaleMax {
		scale = scaleResult(bufProd, hiProd, scale)
	}

	d1.setLow64(bufProd.Low64())
	d1.high = bufProd.U2
	d1.flags = ((d2.flags ^ d1.flags) & signMask) | (uint32(scale) << scaleShift)
	return

ReturnZero:
	*d1 = Decimal{}
}

// varDecFromR4 Converts float32 to Decimal
func varDecFromR4(input float32, result *Decimal) {
	*result = Decimal{}

	// The most we can scale by is 10^28, which is just slightly more
	// than 2^93.  So a float with an exponent of -94 could just
	// barely reach 0.5, but smaller exponents will always round to zero.
	const sngbias uint32 = 126
	var exp int32 = int32(getExponent32(input) - sngbias)
	if exp < -94 {
		return // result should be zeroed out
	}

	if exp > 96 {
		panic("decimal overflow exception")
	}

	var flags uint32
	if input < 0 {
		input = -input
		flags = signMask
	}

	// Round the input to a 7-digit integer.  The R4 format has
	// only 7 digits of precision, and we want to keep garbage digits
	// out of the Decimal were making.
	//
	// Calculate max power of 10 input value could have by multiplying
	// the exponent by log10(2).  Using scaled integer multiplcation,
	// log10(2) * 2 ^ 16 = .30103 * 65536 = 19728.3.
	var dbl float64 = float64(input)
	var power int32 = 6 - ((exp * 19728) >> 16)
	// power is between -22 and 35

	if power >= 0 {
		// We have less than 7 digits, scale input up.
		if power > decScaleMax {
			power = decScaleMax
		}
		dbl *= sDoublePowers10[power]
	} else {
		if power != -1 || dbl >= 1E7 {
			dbl /= sDoublePowers10[-power]
		} else {
			power = 0 // didn't scale it
		}

	}

	if dbl < 1E6 && power < decScaleMax {
		dbl *= 10
		power++
	}

	// Round to integer
	var mant uint32
	mant = uint32(int32(dbl))
	dbl -= float64(uint32(mant)) // difference between input & integer
	if dbl > 0.5 || dbl == 0.5 && (mant&1) != 0 {
		mant++
	}
	if mant == 0 {
		return // result should be zeroed out
	}

	if power < 0 {
		// Add -power factors of 10, -power <= (29 - 7) = 22.
		power = -power
		if power < 10 {
			result.setLow64(uInt32x32To64(mant, sPowers10[power]))
		} else {
			// Have a big power of 10.
			if power > 18 {
				var low64 uint64 = uInt32x32To64(mant, sPowers10[power-18])
				uInt64x64To128(low64, tenToPowerEighteen, result)
			} else {
				var low64 uint64 = uInt32x32To64(mant, sPowers10[power-9])
				var hi64 uint64 = uInt32x32To64(tenToPowerNine, uint32(low64>>32))
				low64 = uInt32x32To64(tenToPowerNine, uint32(low64))
				result.low = uint32(low64)
				hi64 += low64 >> 32
				result.mid = uint32(hi64)
				hi64 >>= 32
				result.high = uint32(hi64)
			}
		}
	} else {

		// Factor out powers of 10 to reduce the scale, if possible.
		// The maximum number we could factor out would be 6.  This
		// comes from the fact we have a 7-digit number, and the
		// MSD must be non-zero -- but the lower 6 digits could be
		// zero.  Note also the scale factor is never negative, so
		// we can't scale by any more than the power we used to
		// get the integer.
		var lmax int32 = power
		if lmax > 6 {
			lmax = 6
		}

		if (mant&0xF) == 0 && lmax >= 4 {
			const den uint32 = 10000
			var div uint32 = mant / den
			if mant == div*den {
				mant = div
				power -= 4
				lmax -= 4
			}
		}

		if (mant&3) == 0 && lmax >= 2 {
			const den uint32 = 100
			var div uint32 = mant / den
			if mant == div*den {
				mant = div
				power -= 2
				lmax -= 2
			}
		}

		if (mant&1) == 0 && lmax >= 1 {
			const den uint32 = 10
			var div uint32 = mant / den
			if mant == div*den {
				mant = div
				power--
			}
		}

		flags |= uint32(power) << scaleShift
		result.low = mant
	}
	result.flags = flags
}

// varDecFromR8 converts float64 to Decimal
func varDecFromR8(input float64, result *Decimal) {
	*result = Decimal{}

	// The most we can scale by is 10^28, which is just slightly more
	// than 2^93.  So a float with an exponent of -94 could just
	// barely reach 0.5, but smaller exponents will always round to zero.
	const dblbias uint32 = 1022
	var exp int32 = int32(getExponent64(input) - dblbias)
	if exp < -94 {
		return // result should be zeroed out
	}
	if exp > 96 {
		panic("decimal overflow excpetion")
	}

	var flags uint32
	if input < 0 {
		input = -input
		flags = signMask
	}

	// Round the input to a 15-digit integer.  The R8 format has
	// only 15 digits of precision, and we want to keep garbage digits
	// out of the Decimal were making.
	//
	// Calculate max power of 10 input value could have by multiplying
	// the exponent by log10(2).  Using scaled integer multiplcation,
	// log10(2) * 2 ^ 16 = .30103 * 65536 = 19728.3.
	var dbl = input
	var power int32 = 14 - ((exp * 19728) >> 16) // power is between -14 and 43

	if power >= 0 {
		// We have less than 15 digits, scale input up.
		if power > decScaleMax {
			power = decScaleMax
		}
		dbl *= sDoublePowers10[power]
	} else {
		if power != -1 || dbl >= 1E15 {
			dbl /= sDoublePowers10[-power]
		} else {
			power = 0 // didn't scale it
		}
	}

	if dbl < 1E14 && power < decScaleMax {
		dbl *= 10
		power++
	}

	// Round to int64
	var mant uint64
	mant = uint64(int64(dbl))
	dbl -= float64(int64(mant)) // difference between input & integer
	if dbl > 0.5 || dbl == 0.5 && (mant&1) != 0 {
		mant++
	}

	if mant == 0 {
		return // result should be zeroed out
	}

	if power < 0 {
		// Add -power factors of 10, -power <= (29 - 15) = 14.
		power = -power
		if power < 10 {
			var pow10 uint32 = sPowers10[power]
			var low64 uint64 = uInt32x32To64(uint32(mant), pow10)
			var hi64 uint64 = uInt32x32To64(uint32(mant>>32), pow10)
			result.low = uint32(low64)
			hi64 += low64 >> 32
			result.mid = uint32(hi64)
			hi64 >>= 32
			result.high = uint32(hi64)
		} else {
			// Have a big power of 10.
			uInt64x64To128(mant, sUlongPowers10[power-1], result)
		}
	} else {

		// Factor out powers of 10 to reduce the scale, if possible.
		// The maximum number we could factor out would be 14.  This
		// comes from the fact we have a 15-digit number, and the
		// MSD must be non-zero -- but the lower 14 digits could be
		// zero.  Note also the scale factor is never negative, so
		// we can't scale by any more than the power we used to
		// get the integer.
		var lmax int32 = power
		if lmax > 14 {
			lmax = 14
		}

		if byte(mant) == 0 && lmax >= 8 {
			const den uint32 = 100000000
			var div uint64 = mant / uint64(den)
			if uint32(mant) == uint32(div*uint64(den)) {
				mant = div
				power -= 8
				lmax -= 8
			}
		}

		if (uint32(mant)&0xF) == 0 && lmax >= 4 {
			const den uint32 = 10000
			var div uint64 = mant / uint64(den)
			if uint32(mant) == uint32(div*uint64(den)) {
				mant = div
				power -= 4
				lmax -= 4
			}
		}

		if (uint32(mant)&3) == 0 && lmax >= 2 {
			const den uint32 = 100
			var div uint64 = mant / uint64(den)
			if uint32(mant) == uint32(div*uint64(den)) {
				mant = div
				power -= 2
				lmax -= 2
			}
		}

		if (uint32(mant)&1) == 0 && lmax >= 1 {
			const den uint32 = 10
			var div uint64 = mant / uint64(den)
			if uint32(mant) == uint32(div*uint64(den)) {
				mant = div
				power--
			}
		}

		flags |= uint32(power) << scaleShift
		result.setLow64(mant)
	}

	result.flags = flags
}

// varR4FromDec converts Decimal to float32
func varR4FromDec(value Decimal) float32 { return float32(varR8FromDec(value)) }

// varR8FromDec converts Decimal to float64
func varR8FromDec(value Decimal) float64 {
	// Value taken via reverse engineering the double that corresponds to 2^64. (oleaut32 has ds2to64 = DEFDS(0, 0, DBLBIAS + 65, 0))
	const ds2to64 float64 = 1.8446744073709552e+019

	var dbl float64 = (float64(value.low64()) + float64(value.high)*ds2to64) / sDoublePowers10[value.Scale()]

	if value.IsNegative() {
		dbl = -dbl
	}
	return dbl
}

// varDecDiv divides two decimal values. On return, d1 contains the result of the operation.
func varDecDiv(d1, d2 *Decimal) {
	var bufQuo *buf12 = new(buf12)
	var power uint32
	var curScale int32

	var scale int32 = int32(int8((d1.flags - d2.flags) >> scaleShift))
	var unscale bool
	var tmp uint32
	var ok bool

	if (d2.high | d2.mid) == 0 {
		// Divisor is only 32 bits. Easy divide.
		var den uint32 = d2.low
		if den == 0 {
			panic("division by zero exception")
		}

		bufQuo.SetLow64(d1.low64())
		bufQuo.U2 = d1.high
		var remainder uint32 = div96By32(bufQuo, den)

		for {
			if remainder == 0 {
				if scale < 0 {
					if -scale < 9 {
						curScale = -scale
					} else {
						curScale = 9
					}
					goto HaveScale
				}
				break
			}

			// We need to unscale if and only if we have a non-zero remainder
			unscale = true

			// We have computed a quotient based on the natural scale
			// ( <dividend scale> - <divisor scale> ).  We have a non-zero
			// remainder, so now we should increase the scale if possible to
			// include more quotient bits.
			//
			// If it doesn't cause overflow, we'll loop scaling by 10^9 and
			// computing more quotient bits as long as the remainder stays
			// non-zero.  If scaling by that much would cause overflow, we'll
			// drop out of the loop and scale by as much as we can.
			//
			// Scaling by 10^9 will overflow if bufQuo[2].bufQuo[1] >= 2^32 / 10^9
			// = 4.294 967 296.  So the upper limit is bufQuo[2] == 4 and
			// bufQuo[1] == 0.294 967 296 * 2^32 = 1,266,874,889.7+.  Since
			// quotient bits in bufQuo[0] could be all 1's, then 1,266,874,888
			// is the largest value in bufQuo[1] (when bufQuo[2] == 4) that is
			// assured not to overflow.

			if ok = scale == decScaleMax; !ok {
				curScale = searchScale(bufQuo, scale)
				ok = curScale == 0
			}
			if ok {
				// No more scaling to be done, but remainder is non-zero.
				// Round quotient.
				tmp = remainder << 1
				if tmp < remainder || tmp >= den && (tmp > den || (bufQuo.U0&1) != 0) {
					goto RoundUp
				}
				break
			}

		HaveScale:
			power = sPowers10[curScale]
			scale += curScale

			if increaseScale(bufQuo, power) != 0 {
				goto ThrowOverflow
			}

			var num uint64 = uInt32x32To64(remainder, power)
			// TODO: https://github.com/dotnet/coreclr/issues/3439
			var div uint32 = uint32(num / uint64(den))
			remainder = uint32(num) - div*den

			if !add32To96(bufQuo, div) {
				scale = overflowUnscale(bufQuo, scale, remainder != 0)
				break
			}
		}
	} else {

		// Divisor has bits set in the upper 64 bits.
		//
		// Divisor must be fully normalized (shifted so bit 31 of the most
		// significant uint is 1).  Locate the MSB so we know how much to
		// normalize by.  The dividend will be shifted by the same amount so
		// the quotient is not changed.
		tmp = d2.high
		if tmp == 0 {
			tmp = d2.mid
		}

		curScale = leadingZeroCount(tmp)

		// Shift both dividend and divisor left by curScale.
		var bufRem *buf16 = new(buf16)
		bufRem.SetLow64(d1.low64() << curScale)
		bufRem.SetHigh64((uint64(d1.mid) + (uint64(d1.high) << 32)) >> (32 - curScale))

		var divisor uint64 = d2.low64() << curScale

		if d2.high == 0 {
			// Have a 64-bit divisor in sdlDivisor.  The remainder
			// (currently 96 bits spread over 4 uints) will be < divisor.
			bufQuo.U1 = div96By64((*buf12)(unsafe.Pointer(&bufRem.U1)), divisor)
			bufQuo.U0 = div96By64((*buf12)(unsafe.Pointer(bufRem)), divisor)

			for {
				if bufRem.Low64() == 0 {
					if scale < 0 {
						if -scale < 9 {
							curScale = -scale
						} else {
							curScale = 9
						}
						goto HaveScale64
					}
				}

				// We need to unscale if and only if we have a non-zero remainder
				unscale = true

				// Remainder is non-zero.  Scale up quotient and remainder by
				// powers of 10 so we can compute more significant bits.
				if ok = scale == decScaleMax; !ok {
					curScale = searchScale(bufQuo, scale)
					ok = curScale == 0
				}
				if ok {
					// No more scaling to be done, but remainder is non-zero.
					// Round quotient.
					var tmp64 uint64 = bufRem.Low64()
					if ok = int64(tmp64) < 0; !ok {
						tmp64 <<= 1
						if ok = tmp64 > divisor; !ok {
							ok = (tmp64 == divisor) && (bufQuo.U0&1) != 0
						}
					}
					if ok {
						goto RoundUp
					}
					break
				}

			HaveScale64:
				power = sPowers10[curScale]
				scale += curScale

				if increaseScale(bufQuo, power) != 0 {
					goto ThrowOverflow
				}

				increaseScale64((*buf12)(unsafe.Pointer(bufRem)), power)
				tmp = div96By64((*buf12)(unsafe.Pointer(bufRem)), divisor)
				if !add32To96(bufQuo, tmp) {
					scale = overflowUnscale(bufQuo, scale, bufRem.Low64() != 0)
					break
				}
			}
		} else {

			// Have a 96-bit divisor in bufDivisor.
			//
			// Start by finishing the shift left by curScale.
			var bufDivisor *buf12 = new(buf12)
			bufDivisor.SetLow64(divisor)
			bufDivisor.U2 = uint32(((uint64(d2.mid) + (uint64(d2.high) << 32)) >> (32 - curScale)))

			// The remainder (currently 96 bits spread over 4 uints) will be < divisor.
			bufQuo.SetLow64(uint64(div128By96(bufRem, bufDivisor)))

			for {
				if (bufRem.Low64() | uint64(bufRem.U2)) == 0 { // TODO Check. C# does not cast Low64
					if scale < 0 {
						if -scale < 9 {
							curScale = -scale
						} else {
							curScale = 9
						}
						goto HaveScale96
					}
					break
				}

				// We need to unscale if and only if we have a non-zero remainder
				unscale = true

				// Remainder is non-zero.  Scale up quotient and remainder by
				// powers of 10 so we can compute more significant bits.
				if ok = scale == decScaleMax; !ok {
					curScale = searchScale(bufQuo, scale)
					ok = curScale == 0
				}
				if ok {
					// No more scaling to be done, but remainder is non-zero.
					// Round quotient.
					if int32(bufRem.U2) < 0 {
						goto RoundUp
					}

					tmp = bufRem.U1 >> 31
					bufRem.SetLow64(bufRem.Low64() << 1)
					bufRem.U2 = (bufRem.U2 << 1) + tmp

					if bufRem.U2 > bufDivisor.U2 || bufRem.U2 == bufDivisor.U2 && (bufRem.Low64() > bufDivisor.Low64() || bufRem.Low64() == bufDivisor.Low64() && (bufQuo.U0&1) != 0) {
						goto RoundUp
					}
					break
				}

			HaveScale96:
				power = sPowers10[curScale]
				scale += curScale
				if increaseScale(bufQuo, power) != 0 {
					goto ThrowOverflow
				}

				bufRem.U3 = increaseScale((*buf12)(unsafe.Pointer(bufRem)), power)
				tmp = div128By96(bufRem, bufDivisor)
				if !add32To96(bufQuo, tmp) {
					scale = overflowUnscale(bufQuo, scale, (bufRem.Low64()|bufRem.High64()) != 0)
					break
				}

			}
		}
	}

Unscale:
	if unscale {
		var low uint32 = bufQuo.U0
		var high64 uint64 = bufQuo.High64()
		calcUnscale(&low, &high64, &scale)
		d1.low = low
		d1.mid = uint32(high64)
		d1.high = uint32(high64 >> 32)
	} else {
		d1.setLow64(bufQuo.Low64())
		d1.high = bufQuo.U2
	}
	d1.flags = ((d1.flags ^ d2.flags) & signMask) | (uint32(scale) << scaleShift)
	return

RoundUp:
	{
		if bufQuo.SetLow64(bufQuo.Low64() + 1); bufQuo.Low64() == 0 {
			if bufQuo.U2++; bufQuo.U2 == 0 {
				scale = overflowUnscale(bufQuo, scale, true)
			}
		}
		goto Unscale
	}

ThrowOverflow:
	panic("decimal overflow exception") // TODO: move this to the actual panic location for better tracing
}

// varDecMod computes the rmeainder between two decimals. On return, d1 contains the result of the
// operation and d2 is trashed.
func varDecMod(d1, d2 *Decimal) {
	if (d2.low | d2.mid | d2.high) == 0 {
		panic("divide by zero exception")
	}

	if (d1.low | d1.mid | d1.high) == 0 {
		return
	}

	// In the operation x % y the sign of y does not matter. Result will have the sign of x.
	d2.flags = (d2.flags & ^signMask) | (d1.flags & signMask)

	var cmp int32 = varDecCmpSub(*d1, *d2)
	if cmp == 0 {
		d1.low = 0
		d1.mid = 0
		d1.high = 0
		if d2.flags > d1.flags {
			d1.flags = d2.flags
		}
		return
	}

	if (cmp ^ int32(d1.flags&signMask)) < 0 {
		return
	}

	// The divisor is smaller than the dividend and both are non-zero. Calculate the integer remainder using the larger scaling factor.

	var scale int32 = int32(int8((d1.flags - d2.flags) >> scaleShift))
	if scale > 0 {
		for ok := true; ok; ok = scale > 0 {
			var power uint32
			if scale >= maxInt32Scale {
				power = tenToPowerNine
			} else {
				power = sPowers10[scale]
			}
			var tmp uint64 = uInt32x32To64(d2.low, power)
			d2.low = uint32(tmp)
			tmp >>= 32
			tmp += (uint64(d2.mid) + (uint64(d2.high) << 32)) * uint64(power)
			d2.mid = uint32(tmp)
			d2.high = uint32(tmp >> 32)
			scale -= maxInt32Scale
		}
		scale = 0
	}

	for ok := true; ok; ok = scale < 0 {
		if scale < 0 {
			d1.flags = d2.flags
			// Try to scale up dividend to match divisor.
			var bufQuo *buf12 = new(buf12)
			bufQuo.SetLow64(d1.low64())
			bufQuo.U2 = d1.high

			for ok2 := true; ok2; ok2 = scale < 0 {
				var iCurScale int32 = searchScale(bufQuo, decScaleMax+scale)
				if iCurScale == 0 {
					break
				}
				var power uint32
				if iCurScale >= maxInt32Scale {
					power = tenToPowerNine
				} else {
					power = sPowers10[iCurScale]
				}
				scale += iCurScale
				var tmp uint64 = uInt32x32To64(bufQuo.U0, power)
				bufQuo.U0 = uint32(tmp)
				tmp >>= 32
				bufQuo.SetHigh64(tmp + bufQuo.High64()*uint64(power))
				if power != tenToPowerNine {
					break
				}
			}
			d1.setLow64(bufQuo.Low64())
			d1.high = bufQuo.U2
		}

		if d1.high == 0 {
			d1.setLow64(d1.low64() % d2.low64())
			return
		} else if (d2.high | d2.mid) == 0 {
			var den uint32 = d2.low
			var tmp uint64 = (uint64(d1.high) << 32) | uint64(d1.mid)
			tmp = ((tmp % uint64(den)) << 32) | uint64(d1.low)
			d1.setLow64(tmp % uint64(den))
			d1.high = 0
		} else {
			varDecModFull(d1, d2, scale)
			return
		}
	}
}

// varDecModFull ...
func varDecModFull(d1, d2 *Decimal, scale int32) {
	// Divisor has bits set in the upper 64 bits.
	//
	// Divisor must be fully normalized (shifted so bit 31 of the most significant uint is 1).
	// Locate the MSB so we know how much to normalize by.
	// The dividend will be shifted by the same amount so the quotient is not changed.
	var tmp uint32 = d2.high
	if tmp == 0 {
		tmp = d2.mid
	}
	var shift int32 = leadingZeroCount(tmp)

	var b *buf28 = new(buf28)
	b.buf24.SetLow64(d1.low64() << shift)
	b.buf24.SetMid64((uint64(d1.mid) + (uint64(d1.high) << 32)) >> (32 - shift))

	// The dividend might need to be scaled up to 221 significant bits.
	// Maximum scaling is required when the divisor is 2^64 with scale 28 and is left shifted 31 bits
	// and the dividend is decimal.MaxValue: (2^96 - 1) * 10^28 << 31 = 221 bits.
	var high uint32 = 3
	for scale < 0 {
		var power uint32
		if scale <= -maxInt32Scale {
			power = tenToPowerNine
		} else {
			power = sPowers10[-scale]
		}
		var buf *[7]uint32 = (*[7]uint32)(unsafe.Pointer(b))
		var tmp64 uint64 = uInt32x32To64(b.buf24.U0, power)
		b.buf24.U0 = uint32(tmp64)
		var i uint32
		for i = 1; i <= high; i++ {
			tmp64 >>= 32
			tmp64 += uInt32x32To64(buf[i], power)
			buf[i] = uint32(tmp64)
		}
		// The high bit of the dividend must not be set.
		if tmp64 > math.MaxInt32 {
			high++
			buf[high] = uint32(tmp64 >> 32)
		}

		scale += maxInt32Scale
	}

	if d2.high == 0 {
		var divisor uint64 = d2.low64() << shift
		switch high {
		case 6:
			div96By64((*buf12)(unsafe.Pointer(&b.buf24.U4)), divisor)
			div96By64((*buf12)(unsafe.Pointer(&b.buf24.U3)), divisor) // In c# goto case 5
			div96By64((*buf12)(unsafe.Pointer(&b.buf24.U2)), divisor) // In c# goto case 4
		case 5:
			div96By64((*buf12)(unsafe.Pointer(&b.buf24.U3)), divisor)
			div96By64((*buf12)(unsafe.Pointer(&b.buf24.U2)), divisor) // In c# goto case 4
		case 4:
			div96By64((*buf12)(unsafe.Pointer(&b.buf24.U2)), divisor)
		}
		div96By64((*buf12)(unsafe.Pointer(&b.buf24.U1)), divisor)
		div96By64((*buf12)(unsafe.Pointer(b)), divisor)

		d1.setLow64(b.buf24.Low64() >> shift)
		d1.high = 0
	} else {
		var bufDivisor *buf12 = new(buf12)
		bufDivisor.SetLow64(d2.low64() << shift)
		bufDivisor.U2 = uint32(((uint64(d2.mid) + (uint64(d2.high) << 32)) >> (32 - shift)))

		switch high {
		case 6:
			div128By96((*buf16)(unsafe.Pointer(&b.buf24.U3)), bufDivisor)
			div128By96((*buf16)(unsafe.Pointer(&b.buf24.U2)), bufDivisor) // In c# goto case 5
			div128By96((*buf16)(unsafe.Pointer(&b.buf24.U1)), bufDivisor) // In c# goto case 4
		case 5:
			div128By96((*buf16)(unsafe.Pointer(&b.buf24.U2)), bufDivisor)
			div128By96((*buf16)(unsafe.Pointer(&b.buf24.U1)), bufDivisor) // In c# goto case 4
		case 4:
			div128By96((*buf16)(unsafe.Pointer(&b.buf24.U1)), bufDivisor)
		}
		div128By96((*buf16)(unsafe.Pointer(b)), bufDivisor)

		d1.setLow64((b.buf24.Low64() >> shift) + (uint64(b.buf24.U2) << (32 - shift) << 32))
		d1.high = b.buf24.U2 >> shift
	}
}

// internalRound does an in-place round by the specified value.
func internalRound(d *Decimal, scale uint32, mode RoundingMode) {
	// the scale becomes the desired decimal count
	d.flags -= scale << scaleShift

	var remainder, sticky, power uint32
	// First divide the value by constant 10^9 up to three times
	for int32(scale) >= maxInt32Scale {
		scale -= uint32(maxInt32Scale)

		const divisor uint32 = tenToPowerNine
		var n uint32 = d.high
		if n == 0 {
			var tmp uint64 = d.low64()
			var div uint64 = tmp / uint64(divisor)
			d.setLow64(div)
			remainder = uint32(tmp - div*uint64(divisor))
		} else {
			var q uint32 = n / divisor
			d.high = q
			remainder = n - q*divisor
			n = d.mid
			if (n | remainder) != 0 {
				q = uint32(((uint64(remainder) << 32) | uint64(n)) / uint64(divisor))
				d.mid = q
				remainder = n - q*divisor
			}
			n = d.low
			if (n | remainder) != 0 {
				q = uint32(((uint64(remainder) << 32) | uint64(n)) / uint64(divisor))
				d.low = q
				remainder = n - q*divisor
			}
		}
		power = divisor
		if scale == 0 {
			goto CheckRemainder
		}
		sticky |= remainder
	}

	{
		power = sPowers10[scale]
		// TODO: https://github.com/dotnet/coreclr/issues/3439
		var n uint32 = d.high
		if n == 0 {
			var tmp uint64 = d.low64()
			if tmp == 0 {
				if mode <= Truncate {
					goto Done
				}
				remainder = 0
				goto CheckRemainder
			}
			var div uint64 = tmp / uint64(power)
			d.setLow64(div)
			remainder = uint32(tmp - div*uint64(power))
		} else {
			var q uint32 = n / power
			d.high = q
			remainder = n - q*power
			n = d.mid
			if (n | remainder) != 0 {
				q = uint32(((uint64(remainder) << 32) | uint64(n)) / uint64(power))
				d.mid = q
				remainder = n - q*power
			}
			n = d.low
			if (n | remainder) != 0 {
				q = uint32(((uint64(remainder) << 32) | uint64(n)) / uint64(power))
				d.low = q
				remainder = n - q*power
			}
		}
	}

CheckRemainder:
	switch mode {
	case Truncate:
		goto Done
	case ToEven:
		// To do IEEE rounding, we add LSB of result to sticky bits so either causes round up if remainder * 2 == last divisor.
		remainder <<= 1
		if (sticky | d.low&1) != 0 {
			remainder++
		}
		if power >= remainder {
			goto Done
		}
	case AwayFromZero:
		// Round away from zero at the mid point.
		remainder <<= 1
		if power > remainder {
			goto Done
		}
	case Floor:
		// Round toward -infinity if we have chopped off a non-zero amount from a negative value.
		if (remainder|sticky) == 0 || !d.IsNegative() {
			goto Done
		}
	case Ceiling:
		// Round toward infinity if we have chopped off a non-zero amount from a positive value.
		if (remainder|sticky) == 0 || d.IsNegative() {
			goto Done
		}
	default:
		panic("unknown rounding mode")
	}
	if d.setLow64(d.low64() + 1); d.low64() == 0 {
		d.high++
	}

Done:
	return // replace goto Done with return above.
}

// decDivMod1E9 ...
func decDivMod1E9(value *Decimal) uint32 {
	var high64 uint64 = (uint64(value.high) << 32) + uint64(value.mid)
	var div64 uint64 = high64 / uint64(tenToPowerNine)
	value.high = uint32(div64 >> 32)
	value.mid = uint32(div64)

	var num uint64 = ((high64 - uint64(uint32(div64))*uint64(tenToPowerNine)) << 32) + uint64(value.low)
	var div uint32 = uint32(num / uint64(tenToPowerNine))
	value.low = div
	return uint32(num) - div*tenToPowerNine
}
