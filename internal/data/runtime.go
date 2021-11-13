package data

import (
	"fmt"
	"strconv"
)

// Runtime type for movie runtime
type Runtime int32

// MarshalJSON converts movie runtime to string
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)

	quotedJSONValue := strconv.Quote(jsonValue)

	return []byte(quotedJSONValue), nil
}
