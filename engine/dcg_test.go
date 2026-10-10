package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type oversizedDCGCompound struct {
	Compound
}

func (oversizedDCGCompound) Arity() int {
	return int(^uint(0) >> 1)
}

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

func TestPhraseRejectsMeteredTermAllocation(t *testing.T) {
	var vm VM
	var charged uint64
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		if kind != MeterTermCell {
			return nil
		}
		charged += units
		return atomResourceError.Apply(resourceMemory.Term())
	})

	ok, err := Phrase(
		&vm,
		NewAtom("rule").Apply(atomTrue),
		atomEmptyList,
		atomEmptyList,
		Success,
		nil,
	).Force(context.Background())

	assert.False(t, ok)
	assert.Equal(t, resourceError(resourceMemory, nil), err)
	assert.Equal(t, uint64(3), charged)
}

func TestDCGTerminals_MetersPartialListTail(t *testing.T) {
	var vm VM
	var charged uint64
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		if kind == MeterTermCell {
			charged += units
		}
		return nil
	})

	_, err := dcgTerminals(List(NewAtom("a"), NewAtom("b")), vm.NewVariable(), vm.NewVariable(), vm.NewEnv())

	assert.NoError(t, err)
	assert.Equal(t, uint64(5), charged)
}

func TestExpandDCGRejectsOversizedNonterminal(t *testing.T) {
	var vm VM
	rule := atomArrow.Apply(
		oversizedDCGCompound{Compound: &compound{functor: NewAtom("too_large")}},
		atomEmptyList,
	)

	_, err := expandDCG(&vm, rule, nil)
	meterErr, ok := err.(dcgMeterError)
	assert.True(t, ok)
	if ok {
		assert.Equal(t, resourceError(resourceMemory, nil), meterErr.Exception)
	}
}
