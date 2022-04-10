package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestException_Error(t *testing.T) {
	e := Exception{term: Atom("foo")}
	assert.Equal(t, "foo", e.Error())
}
