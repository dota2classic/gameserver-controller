package util

import (
	"os"
	"strconv"
)

func GetEnvInt(key string, defaultVal int) int {
	valStr, ok := os.LookupEnv(key)
	if !ok || valStr == "" {
		return defaultVal
	}

	val, err := strconv.Atoi(valStr)
	if err != nil {
		// fallback to default if parsing fails
		return defaultVal
	}
	return val
}
