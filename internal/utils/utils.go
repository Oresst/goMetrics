package utils

import (
	"strconv"
	"strings"
)

func BetterFormat(num float64) string {
	s := strconv.FormatFloat(num, 'f', -1, 64)
	return strings.TrimRight(strings.TrimRight(s, "0"), ".")
}

func StrToInt(value string, defaultValue int) int {
	newValue, err := strconv.Atoi(value)

	if err != nil {
		return defaultValue
	}

	return newValue
}

func PointFloat64(value float64) *float64 {
	return &value
}

func PointInt64(value int64) *int64 {
	return &value
}
