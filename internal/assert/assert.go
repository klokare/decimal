// Package assert provides the handful of test assertions this module needs.
//
// It exists so that go.mod can stay free of dependencies, including test-only
// ones. The signatures deliberately mirror the subset of github.com/stretchr/testify
// that the test suite used before, so test bodies read the same.
package assert

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// msg renders the trailing variadic context shared by every assertion. The first
// element, if present, is either a preformatted string or a format string for the
// rest.
func msg(args []any) string {
	if len(args) == 0 {
		return ""
	}
	format, ok := args[0].(string)
	if !ok {
		return ": " + fmt.Sprint(args...)
	}
	if len(args) == 1 {
		return ": " + format
	}
	return ": " + fmt.Sprintf(format, args[1:]...)
}

// Equal reports whether want and got are deeply equal.
func Equal(t testing.TB, want, got any, args ...any) bool {
	t.Helper()
	if objectsAreEqual(want, got) {
		return true
	}
	t.Errorf("not equal%s\n  want: %v\n   got: %v", msg(args), want, got)
	return false
}

// NotEqual reports whether want and got differ.
func NotEqual(t testing.TB, want, got any, args ...any) bool {
	t.Helper()
	if !objectsAreEqual(want, got) {
		return true
	}
	t.Errorf("expected values to differ%s\n  both: %v", msg(args), want)
	return false
}

func objectsAreEqual(want, got any) bool {
	if want == nil || got == nil {
		return want == got
	}
	if wb, ok := want.([]byte); ok {
		gb, ok := got.([]byte)
		if !ok {
			return false
		}
		return string(wb) == string(gb)
	}
	return reflect.DeepEqual(want, got)
}

// True reports whether value is true.
func True(t testing.TB, value bool, args ...any) bool {
	t.Helper()
	if value {
		return true
	}
	t.Errorf("expected true, got false%s", msg(args))
	return false
}

// False reports whether value is false.
func False(t testing.TB, value bool, args ...any) bool {
	t.Helper()
	if !value {
		return true
	}
	t.Errorf("expected false, got true%s", msg(args))
	return false
}

// NoError reports whether err is nil.
func NoError(t testing.TB, err error, args ...any) bool {
	t.Helper()
	if err == nil {
		return true
	}
	t.Errorf("unexpected error%s: %v", msg(args), err)
	return false
}

// Error reports whether err is non-nil.
func Error(t testing.TB, err error, args ...any) bool {
	t.Helper()
	if err != nil {
		return true
	}
	t.Errorf("expected an error, got nil%s", msg(args))
	return false
}

// ErrorIs reports whether err matches target under errors.Is.
func ErrorIs(t testing.TB, err, target error, args ...any) bool {
	t.Helper()
	if errorIs(err, target) {
		return true
	}
	t.Errorf("error does not match target%s\n  want: %v\n   got: %v", msg(args), target, err)
	return false
}

// Less reports whether a < b, for the ordered types the suite uses.
func Less(t testing.TB, a, b any, args ...any) bool {
	t.Helper()
	c, ok := compare(a, b)
	if !ok {
		t.Errorf("cannot order %T against %T%s", a, b, msg(args))
		return false
	}
	if c < 0 {
		return true
	}
	t.Errorf("expected %v < %v%s", a, b, msg(args))
	return false
}

// Greater reports whether a > b, for the ordered types the suite uses.
func Greater(t testing.TB, a, b any, args ...any) bool {
	t.Helper()
	c, ok := compare(a, b)
	if !ok {
		t.Errorf("cannot order %T against %T%s", a, b, msg(args))
		return false
	}
	if c > 0 {
		return true
	}
	t.Errorf("expected %v > %v%s", a, b, msg(args))
	return false
}

func compare(a, b any) (int, bool) {
	av, bv := reflect.ValueOf(a), reflect.ValueOf(b)
	if av.Kind() != bv.Kind() {
		return 0, false
	}
	switch av.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return cmp(av.Int(), bv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return cmp(av.Uint(), bv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return cmp(av.Float(), bv.Float()), true
	case reflect.String:
		return strings.Compare(av.String(), bv.String()), true
	default:
		return 0, false
	}
}

func cmp[T int64 | uint64 | float64](a, b T) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// Panics reports whether fn panics. The recovered value, if any, is returned so
// callers can inspect it.
func Panics(t testing.TB, fn func(), args ...any) (recovered any, ok bool) {
	t.Helper()
	recovered, panicked := didPanic(fn)
	if panicked {
		return recovered, true
	}
	t.Errorf("expected a panic, none occurred%s", msg(args))
	return nil, false
}

// NotPanics reports whether fn returns without panicking.
func NotPanics(t testing.TB, fn func(), args ...any) bool {
	t.Helper()
	recovered, panicked := didPanic(fn)
	if !panicked {
		return true
	}
	t.Errorf("unexpected panic%s: %v", msg(args), recovered)
	return false
}

func didPanic(fn func()) (recovered any, panicked bool) {
	defer func() {
		if r := recover(); r != nil {
			recovered, panicked = r, true
		}
	}()
	fn()
	return nil, false
}
