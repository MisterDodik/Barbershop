package env

import (
	"os"
	"strconv"
)

func GetString(key, fallback string) string {
	result, err := os.LookupEnv(key)

	if !err {
		return fallback
	}
	return result
}
func GetInt(key string, fallback int) int {
	result, err := os.LookupEnv(key)

	if !err {
		return fallback
	}
	num, convErr := strconv.Atoi(result)
	if convErr != nil {
		return fallback
	}
	return num
}
