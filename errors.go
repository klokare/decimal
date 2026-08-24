package decimal

import "errors"

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
