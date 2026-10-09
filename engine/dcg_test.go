package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVM_Phrase(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		var called bool
		vm := VM{
			procedures: buildOrderedMap(
				procedurePair{
					Key: procedureIndicator{name: NewAtom("a"), arity: 2},
					Value: Predicate2(func(_ *VM, s0, s Term, k Cont, env *Env) *Promise {
						called = true
						return k(env)
					}),
				},
			),
		}

		s0, s := vm.NewVariable(), vm.NewVariable()
		ok, err := Phrase(&vm, NewAtom("a"), s0, s, Success, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.True(t, ok)

		assert.True(t, called)
	})

	t.Run("failed", func(t *testing.T) {
		var vm VM
		s0, s := vm.NewVariable(), vm.NewVariable()
		_, err := Phrase(&vm, Integer(0), s0, s, Success, nil).Force(context.Background())
		assert.Error(t, err)
	})
}
