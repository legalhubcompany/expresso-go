package utils

import (
	"strconv"
)

// StringToInt64 converts string to int64 safely
func StringToInt64(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return i
}
