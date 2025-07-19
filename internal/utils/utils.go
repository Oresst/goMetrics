package utils

import (
	"fmt"
	"strconv"
	"strings"
)

func BetterFormat(num float64) string {
	s := fmt.Sprintf("%f", num)
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
