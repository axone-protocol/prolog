package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeSlice(t *testing.T) {
	s, err := makeSlice(-1)

	assert.Nil(t, s)
	assert.ErrorIs(t, err, errOutOfMemory)
}
