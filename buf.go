package decimal

import (
	"unsafe"

	"github.com/klokare/decimal/internal/platform"
)

// buf12 ...
type buf12 struct {
	U0 uint32
	U1 uint32
	U2 uint32
}

func (b buf12) ulo64LE() uint64          { return *(*uint64)(unsafe.Pointer(&b.U0)) }
func (b *buf12) setUlo64LE(value uint64) { *(*uint64)(unsafe.Pointer(&b.U0)) = value }

func (b buf12) uhigh64LE() uint64          { return *(*uint64)(unsafe.Pointer(&b.U1)) }
func (b *buf12) setUhigh64LE(value uint64) { *(*uint64)(unsafe.Pointer(&b.U1)) = value }

// Low64 ...
func (b buf12) Low64() uint64 {
	if platform.LittleEndian {
		return b.ulo64LE()
	}
	return uint64(b.U1)<<32 | uint64(b.U0)
}

// SetLow64 ...
func (b *buf12) SetLow64(value uint64) {
	if platform.LittleEndian {
		b.setUlo64LE(value)
	} else {
		b.U1 = uint32(value >> 32)
		b.U0 = uint32(value)
	}
}

// High64 ... U1-U2 combined (overlaps with Low64)
func (b buf12) High64() uint64 {
	if platform.LittleEndian {
		return b.uhigh64LE()
	}
	return uint64(b.U2)<<32 | uint64(b.U1)
}

// SetHigh64 ... U1-U2 combined (overlaps with Low64)
func (b *buf12) SetHigh64(value uint64) {
	if platform.LittleEndian {
		b.setUhigh64LE(value)
	} else {
		b.U2 = uint32(value >> 32)
		b.U1 = uint32(value)
	}
}

// buf16 ...
type buf16 struct {
	U0 uint32
	U1 uint32
	U2 uint32
	U3 uint32
}

func (b buf16) ulo64LE() uint64          { return *(*uint64)(unsafe.Pointer(&b.U0)) }
func (b *buf16) setUlo64LE(value uint64) { *(*uint64)(unsafe.Pointer(&b.U0)) = value }

func (b buf16) uhigh64LE() uint64          { return *(*uint64)(unsafe.Pointer(&b.U2)) }
func (b *buf16) setUhigh64LE(value uint64) { *(*uint64)(unsafe.Pointer(&b.U2)) = value }

// Low64 ...
func (b buf16) Low64() uint64 {
	if platform.LittleEndian {
		return b.ulo64LE()
	}
	return uint64(b.U1)<<32 | uint64(b.U0)
}

// SetLow64 ...
func (b *buf16) SetLow64(value uint64) {
	if platform.LittleEndian {
		b.setUlo64LE(value)
	} else {
		b.U1 = uint32(value >> 32)
		b.U0 = uint32(value)
	}
}

// High64 ...
func (b buf16) High64() uint64 {
	if platform.LittleEndian {
		return b.uhigh64LE()
	}
	return uint64(b.U3)<<32 | uint64(b.U2)
}

// SetHigh64 ...
func (b *buf16) SetHigh64(value uint64) {
	if platform.LittleEndian {
		b.setUhigh64LE(value)
	} else {
		b.U3 = uint32(value >> 32)
		b.U2 = uint32(value)
	}
}

// buf24 ...
type buf24 struct {
	U0 uint32
	U1 uint32
	U2 uint32
	U3 uint32
	U4 uint32
	U5 uint32
}

func (b buf24) ulo64LE() uint64          { return *(*uint64)(unsafe.Pointer(&b.U0)) }
func (b *buf24) setUlo64LE(value uint64) { *(*uint64)(unsafe.Pointer(&b.U0)) = value }

func (b buf24) umid64LE() uint64          { return *(*uint64)(unsafe.Pointer(&b.U2)) }
func (b *buf24) setUmid64LE(value uint64) { *(*uint64)(unsafe.Pointer(&b.U2)) = value }

func (b buf24) uhigh64LE() uint64          { return *(*uint64)(unsafe.Pointer(&b.U4)) }
func (b *buf24) setUhigh64LE(value uint64) { *(*uint64)(unsafe.Pointer(&b.U4)) = value }

// Low64 ...
func (b buf24) Low64() uint64 {
	if platform.LittleEndian {
		return b.ulo64LE()
	}
	return uint64(b.U1)<<32 | uint64(b.U0)
}

// SetLow64 ...
func (b *buf24) SetLow64(value uint64) {
	if platform.LittleEndian {
		b.setUlo64LE(value)
	} else {
		b.U1 = uint32(value >> 32)
		b.U0 = uint32(value)
	}
}

// Mid64 ...
func (b buf24) Mid64() uint64 {
	if platform.LittleEndian {
		return b.umid64LE()
	}
	return uint64(b.U3)<<32 | uint64(b.U2)
}

// SetMid64 ...
func (b *buf24) SetMid64(value uint64) {
	if platform.LittleEndian {
		b.setUmid64LE(value)
	} else {
		b.U3 = uint32(value >> 32)
		b.U2 = uint32(value)
	}
}

// High64 ...
func (b buf24) High64() uint64 {
	if platform.LittleEndian {
		return b.uhigh64LE()
	}
	return uint64(b.U5)<<32 | uint64(b.U4)
}

// SetHigh64 ...
func (b *buf24) SetHigh64(value uint64) {
	if platform.LittleEndian {
		b.setUhigh64LE(value)
	} else {
		b.U5 = uint32(value >> 32)
		b.U4 = uint32(value)
	}
}

// Length ...
func (b buf24) Length() int32 { return 6 }

// buf28 ...
type buf28 struct {
	buf24
	U6 uint32
}

// Length ...
func (b buf28) Length() int32 { return 7 }
