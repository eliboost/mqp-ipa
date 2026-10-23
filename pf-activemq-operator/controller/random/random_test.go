package random

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	a := New(5)
	b := New(5)

	assert.Len(t, a, 5)
	assert.Len(t, b, 5)
	assert.NotEqualValues(t, a, b)
}
