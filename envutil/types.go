package envutil

import (
	"errors"
)

// errUnsetValue is a special error used internally to unset a Reader's value
// during Map operations. When returned from a Map function, the resulting
// Reader will have present=false.
var errUnsetValue = errors.New("unset value")
