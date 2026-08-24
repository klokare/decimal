package decimal

import (
	"testing"
	"unsafe"

	"github.com/klokare/decimal/v2/internal/assert"
)

// The arithmetic core reinterprets its scratch buffers as flat [N]uint32 arrays,
// exactly as the .NET reference does. That is sound only while every buffer is a
// run of adjacent, unpadded uint32 words with array alignment. These assertions
// fail loudly if a field is ever reordered, retyped or inserted.
func TestBufferLayout(t *testing.T) {
	t.Run("sizes", func(t *testing.T) {
		assert.Equal(t, uintptr(12), unsafe.Sizeof(buf12{}), "buf12")
		assert.Equal(t, uintptr(16), unsafe.Sizeof(buf16{}), "buf16")
		assert.Equal(t, uintptr(24), unsafe.Sizeof(buf24{}), "buf24")
		assert.Equal(t, uintptr(28), unsafe.Sizeof(buf28{}), "buf28")
		assert.Equal(t, uintptr(16), unsafe.Sizeof(Decimal{}), "Decimal")
	})

	t.Run("alignment", func(t *testing.T) {
		assert.Equal(t, unsafe.Alignof([6]uint32{}), unsafe.Alignof(buf24{}), "buf24 vs [6]uint32")
		assert.Equal(t, unsafe.Alignof([7]uint32{}), unsafe.Alignof(buf28{}), "buf28 vs [7]uint32")
		assert.Equal(t, unsafe.Alignof(buf12{}), unsafe.Alignof(buf16{}), "buf12 vs buf16")
	})

	t.Run("no padding in buf24", func(t *testing.T) {
		var b buf24
		base := uintptr(unsafe.Pointer(&b))
		for i, got := range []uintptr{
			uintptr(unsafe.Pointer(&b.U0)) - base,
			uintptr(unsafe.Pointer(&b.U1)) - base,
			uintptr(unsafe.Pointer(&b.U2)) - base,
			uintptr(unsafe.Pointer(&b.U3)) - base,
			uintptr(unsafe.Pointer(&b.U4)) - base,
			uintptr(unsafe.Pointer(&b.U5)) - base,
		} {
			assert.Equal(t, uintptr(i*4), got, "buf24.U%d offset", i)
		}
	})

	// A buf24 viewed as [6]uint32 must observe the same words in the same order.
	t.Run("array view aliases the fields", func(t *testing.T) {
		var b buf24
		b.U0, b.U1, b.U2, b.U3, b.U4, b.U5 = 10, 11, 12, 13, 14, 15
		view := (*[6]uint32)(unsafe.Pointer(&b))
		for i := range view {
			assert.Equal(t, uint32(10+i), view[i], "view[%d]", i)
		}
		view[3] = 99
		assert.Equal(t, uint32(99), b.U3, "write through the view")
	})

	// buf12 views taken at a word offset into a larger buffer must line up too.
	t.Run("offset buf12 view", func(t *testing.T) {
		var b buf16
		b.U0, b.U1, b.U2, b.U3 = 1, 2, 3, 4
		v := (*buf12)(unsafe.Pointer(&b.U1))
		assert.Equal(t, uint32(2), v.U0, "U0")
		assert.Equal(t, uint32(3), v.U1, "U1")
		assert.Equal(t, uint32(4), v.U2, "U2")
	})
}

// The 64-bit accessors are plain shifts rather than a reinterpreted pointer, so
// they behave identically on every architecture. Pin the word order.
func TestWordAccessors(t *testing.T) {
	t.Run("buf12", func(t *testing.T) {
		var b buf12
		b.SetLow64(0x1122334455667788)
		assert.Equal(t, uint32(0x55667788), b.U0, "U0")
		assert.Equal(t, uint32(0x11223344), b.U1, "U1")
		assert.Equal(t, uint64(0x1122334455667788), b.Low64(), "Low64 round-trip")

		b.SetHigh64(0x99aabbccddeeff00)
		assert.Equal(t, uint32(0xddeeff00), b.U1, "U1 after SetHigh64")
		assert.Equal(t, uint32(0x99aabbcc), b.U2, "U2 after SetHigh64")
		assert.Equal(t, uint64(0x99aabbccddeeff00), b.High64(), "High64 round-trip")
	})

	t.Run("decimal", func(t *testing.T) {
		var d Decimal
		d.setLow64(0x00000002ffffffff)
		assert.Equal(t, uint32(0xffffffff), d.low, "low")
		assert.Equal(t, uint32(2), d.mid, "mid")
		assert.Equal(t, uint64(0x00000002ffffffff), d.low64(), "low64 round-trip")
	})
}
