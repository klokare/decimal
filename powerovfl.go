package decimal

// powerOvfl ...
type powerOvfl struct {
	Hi    uint32
	MidLo uint64
}

// newPowerOvfl ...
func newPowerOvfl(hi, mid, lo uint32) powerOvfl {
	return powerOvfl{
		Hi:    hi,
		MidLo: (uint64(mid) << 32) + uint64(lo),
	}
}

// powerOvflValues is a table of the largest values that can be in the upper two
// uints of a 96-bit number that will not overflow when multiplied
// by a given power.  For the upper word, this is a table of
// 2^32 / 10^n for 1 <= n <= 8.  For the lower word, this is the
// remaining fraction part * 2^32.  2^32 = 4294967296.
var powerOvflValues = [8]powerOvfl{
	newPowerOvfl(429496729, 2576980377, 2576980377), // 10^1 remainder 0.6
	newPowerOvfl(42949672, 4123168604, 687194767),   // 10^2 remainder 0.16
	newPowerOvfl(4294967, 1271310319, 2645699854),   // 10^3 remainder 0.616
	newPowerOvfl(429496, 3133608139, 694066715),     // 10^4 remainder 0.1616
	newPowerOvfl(42949, 2890341191, 2216890319),     // 10^5 remainder 0.51616
	newPowerOvfl(4294, 4154504685, 2369172679),      // 10^6 remainder 0.551616
	newPowerOvfl(429, 2133437386, 4102387834),       // 10^7 remainder 0.9551616
	newPowerOvfl(42, 4078814305, 410238783),         // 10^8 remainder 0.09991616
}
