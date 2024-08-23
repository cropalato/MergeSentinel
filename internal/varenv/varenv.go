//
// varenv.go
// Copyright (C) 2024 rmelo <Ricardo Melo <rmelo@ludia.com>>
//
// Distributed under terms of the MIT license.
//

package varenv

import (
	"os"
	"strconv"
)

// LookupEnvOrString returns the value from env variable key is exists or defaultVal as string
func LookupEnvOrString(key string, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	return defaultVal
}

// LookupEnvOrInt returns the value from env variable key is exists or defaultVal as string
func LookupEnvOrInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		newVal, err := strconv.Atoi(val)
		if err == nil {
			return newVal
		}
	}
	return defaultVal
}

// LookupEnvOrBool returns the value from env variable key is exists or defaultVal as string
func LookupEnvOrBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		newVal, err := strconv.ParseBool(val)
		if err == nil {
			return newVal
		}
	}
	return defaultVal
}
