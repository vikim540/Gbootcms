package parser

import (
	"strconv"
)

func parseIntSafe(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func parseFloatSafe(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}
