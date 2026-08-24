package decimal

import (
	"errors"
	"fmt"
)

// Errors reported by this package. Every failure -- whether returned or
// delivered as a panic -- wraps one of these, so errors.Is identifies the cause
// in both styles:
//
//	v, err := a.TryDiv(b)
//	if errors.Is(err, decimal.ErrDivideByZero) { ... }
//
//	defer func() {
//		if r := recover(); r != nil {
//			if err, ok := r.(error); ok && errors.Is(err, decimal.ErrOverflow) { ... }
//		}
//	}()
var (
	// ErrOverflow means the result is outside the range a Decimal can hold.
	ErrOverflow = errors.New("decimal: overflow")

	// ErrDivideByZero means the divisor was zero.
	ErrDivideByZero = errors.New("decimal: divide by zero")

	// ErrScaleRange means a scale or digit count fell outside 0 to 28.
	ErrScaleRange = errors.New("decimal: scale out of range")

	// ErrSyntax means a string could not be parsed as a Decimal.
	ErrSyntax = errors.New("decimal: invalid syntax")

	// ErrFormat means a format string was not understood.
	ErrFormat = errors.New("decimal: invalid format")
)

// wrapf builds an error that wraps one of the sentinels above with context.
func wrapf(sentinel error, format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{sentinel}, args...)...)
}

// recovered converts a panic raised by the arithmetic core into an error, for
// the Try and E variants. It is used as
//
//	defer func() { err = recovered(recover(), &result) }()
//
// and zeroes the result on failure so a caller who ignores the error does not
// see a partially computed value.
//
// A panic that is not one of this package's errors is re-raised: it is a bug
// here or a runtime fault, and swallowing it would hide both.
func recovered[T any](r any, result *T) error {
	if r == nil {
		return nil
	}
	err, ok := r.(error)
	if !ok || !isDecimalError(err) {
		panic(r)
	}
	var zero T
	*result = zero
	return err
}

// isDecimalError reports whether err is one of this package's sentinels.
func isDecimalError(err error) bool {
	for _, sentinel := range []error{
		ErrOverflow, ErrDivideByZero, ErrScaleRange, ErrSyntax, ErrFormat,
	} {
		if errors.Is(err, sentinel) {
			return true
		}
	}
	return false
}
