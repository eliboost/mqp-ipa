// Package random contains random helper
package random

import (
	"crypto/rand"
	"encoding/base64"
	"math"
)

// New returns a new random string with the expected length
func New(length int) string {
	buff := make([]byte, int(math.Ceil(float64(length)/float64(1.33333333333))))

	_, _ = rand.Read(buff)

	str := base64.RawURLEncoding.EncodeToString(buff)

	return str[:length] // strip 1 extra character we get from odd length results
}
