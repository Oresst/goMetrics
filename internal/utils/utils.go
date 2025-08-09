package utils

import (
	"strconv"
)

func BetterFormat(num float64) string {
	return strconv.FormatFloat(num, 'f', -1, 64)
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
