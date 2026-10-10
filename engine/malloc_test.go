package engine

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeTerms_InvalidSize(t *testing.T) {
	for _, n := range []int64{-1, maxTermCells + 1} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			var vm VM
			charged := false
			vm.InstallMeter(func(MeterKind, uint64) Term {
				charged = true
				return nil
			})
			var err error
			func() {
				defer recoverMeterError(&err)
				makeTerms(n, vm.prepareEnv(nil))
			}()
			assert.Equal(t, resourceError(resourceMemory, vm.NewEnv()), err)
			assert.False(t, charged)
		})
	}
}

func TestMakeTerms_QuotaBeforeAllocation(t *testing.T) {
	var vm VM
	n := maxTermCells
	if maxVariableCells < n {
		n = maxVariableCells
	}
	calls := 0
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		if kind == MeterTermCell {
			calls++
			assert.Equal(t, uint64(n), units)
			return atomResourceError.Apply(atomMemory)
		}
		return nil
	})
	var err error
	func() {
		defer recoverMeterError(&err)
		makeTerms(n, vm.NewEnv())
	}()
	assert.Equal(t, resourceError(resourceMemory, vm.NewEnv()), err)
	assert.Equal(t, 1, calls)
}

func TestAddTermCells_Overflow(t *testing.T) {
	var err error
	func() {
		defer recoverMeterError(&err)
		addTermCells(maxTermCells, 1, nil)
	}()
	assert.Equal(t, resourceError(resourceMemory, nil), err)
}

func TestAppendTerms_QuotaIsCumulative(t *testing.T) {
	var vm VM
	remaining := uint64(3)
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		if kind == MeterTermCell {
			if units > remaining {
				return atomResourceError.Apply(atomMemory)
			}
			remaining -= units
		}
		return nil
	})
	env := vm.prepareEnv(nil)
	var terms []Term
	terms = appendTerms(terms, env, Integer(1), Integer(2))
	terms = appendTerms(terms, env, Integer(3))
	assert.Equal(t, []Term{Integer(1), Integer(2), Integer(3)}, terms)
	var err error
	func() {
		defer recoverMeterError(&err)
		terms = appendTerms(terms[:0], env, Integer(4))
	}()
	assert.Equal(t, resourceError(resourceMemory, env), err)
	assert.Equal(t, []Term{Integer(1), Integer(2), Integer(3)}, terms)
}
