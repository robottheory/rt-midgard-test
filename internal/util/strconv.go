package util

import "strconv"

func IntStr(v int64) string {
	return strconv.FormatInt(v, 10)
}
