package config

import (
	"os"
	"strings"
)

const prefix = "$"

// ValueOf extracts the environment variable name given or the plain string given
//
// e.g. foo -> foo
//      $DATABASE_URL -> http://foo.bar:8083
func ValueOf(s string) string {
	if strings.HasPrefix(s, prefix) && len(s) > 1 {
		return os.Getenv(s[1:])
	}
	return s
}
