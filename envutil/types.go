package envutil

import (
	"errors"
)

// ErrUnsetValue is a special error used internally to unset a Reader's value
// during Map operations. When returned from a Map function, the resulting
// Reader will have present=false.
var ErrUnsetValue = errors.New("unset value")
