package handlers

import (
	"strconv"
)

// parseBool parses a boolean string into a bool. If the string fails to parse into a string, the default ('def') boolean value is returned.
func parseBool(boolS string, def bool) bool {
	parsed, err := strconv.ParseBool(boolS)
	if err != nil {
		return def
	}
	return parsed
}
