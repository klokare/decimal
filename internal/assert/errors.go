package assert

import "errors"

// errorIs is a thin wrapper so the assertion helpers do not import errors at
// every call site.
func errorIs(err, target error) bool { return errors.Is(err, target) }
