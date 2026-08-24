package decimal

import "math/rand"

// newTestRand returns a deterministic generator, so a failing case can be
// reproduced from its seed alone.
func newTestRand(seed int64) *rand.Rand { return rand.New(rand.NewSource(seed)) }
