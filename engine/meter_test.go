package engine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVM_MeterInstruction(t *testing.T) {
	var vm VM
	vm.Register0(NewAtom("foo"), func(_ *VM, k Cont, env *Env) *Promise {
		return k(env)
	})

	counts := map[MeterKind]uint64{}
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		counts[kind] += units
		return nil
	})

	ok, err := Call(&vm, NewAtom("foo"), Success, nil).Force(context.Background())
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, uint64(3), counts[MeterInstruction])
}

func TestVM_MeterUnifyStep(t *testing.T) {
	var vm VM
	vm.Register2(atomEqual, Unify)

	counts := map[MeterKind]uint64{}
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		counts[kind] += units
		return nil
	})

	goal := atomEqual.Apply(
		NewAtom("f").Apply(NewAtom("a"), NewAtom("b")),
		NewAtom("f").Apply(NewAtom("a"), NewAtom("b")),
	)

	ok, err := Call(&vm, goal, Success, nil).Force(context.Background())
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, uint64(3), counts[MeterUnifyStep])
}

func TestVM_MeterUnifyStep_PreservedAcrossEnvRewrites(t *testing.T) {
	var vm VM
	vm.Register2(atomEqual, Unify)

	const n = 64

	left := make([]Term, n)
	right := make([]Term, n)
	for i := 0; i < n; i++ {
		left[i] = NewVariable()
		right[i] = Integer(i)
	}

	counts := map[MeterKind]uint64{}
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		counts[kind] += units
		return nil
	})

	ok, err := Call(&vm, atomEqual.Apply(List(left...), List(right...)), Success, nil).Force(context.Background())
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, uint64(2*n+1), counts[MeterUnifyStep])
}

func TestVM_MeterListCell(t *testing.T) {
	var vm VM
	vm.Register2(NewAtom("length"), Length)

	counts := map[MeterKind]uint64{}
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		counts[kind] += units
		return nil
	})

	goal := NewAtom("length").Apply(List(NewAtom("a"), NewAtom("b"), NewAtom("c")), Integer(3))

	ok, err := Call(&vm, goal, Success, nil).Force(context.Background())
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, uint64(3), counts[MeterListCell])
}

func TestVM_MeterCopyNode(t *testing.T) {
	var vm VM
	vm.Register2(NewAtom("copy_term"), CopyTerm)

	counts := map[MeterKind]uint64{}
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		counts[kind] += units
		return nil
	})

	x := NewVariable()
	goal := NewAtom("copy_term").Apply(
		NewAtom("f").Apply(x, List(NewAtom("a"))),
		NewVariable(),
	)

	ok, err := Call(&vm, goal, Success, nil).Force(context.Background())
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, uint64(3), counts[MeterCopyNode])
}

func TestVM_MeterArithNode(t *testing.T) {
	var vm VM
	vm.Register2(NewAtom("is"), Is)

	counts := map[MeterKind]uint64{}
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		counts[kind] += units
		return nil
	})

	goal := NewAtom("is").Apply(
		NewVariable(),
		atomPlus.Apply(Integer(1), atomAsterisk.Apply(Integer(2), Integer(3))),
	)

	ok, err := Call(&vm, goal, Success, nil).Force(context.Background())
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, uint64(5), counts[MeterArithNode])
}

func TestVM_MeterCompareStep(t *testing.T) {
	var vm VM
	vm.Register3(NewAtom("compare"), Compare)

	counts := map[MeterKind]uint64{}
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		counts[kind] += units
		return nil
	})

	goal := NewAtom("compare").Apply(
		NewVariable(),
		NewAtom("f").Apply(NewAtom("a")),
		NewAtom("f").Apply(NewAtom("b")),
	)

	ok, err := Call(&vm, goal, Success, nil).Force(context.Background())
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, uint64(3), counts[MeterCompareStep])
}

func TestVM_MeterException(t *testing.T) {
	var vm VM
	vm.Register2(atomEqual, Unify)

	want := NewAtom("resource_error").Apply(NewAtom("gas"))
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		if kind == MeterUnifyStep {
			return want
		}
		return nil
	})

	ok, err := Call(&vm, atomEqual.Apply(NewVariable(), Integer(1)), Success, nil).Force(context.Background())
	assert.False(t, ok)
	ex, okCast := err.(Exception)
	assert.True(t, okCast)
	pattern := atomError.Apply(
		NewAtom("resource_error").Apply(NewAtom("gas")),
		atomSlash.Apply(atomEqual, Integer(2)),
	)
	_, matched := NewEnv().Unify(pattern, ex.Term())
	assert.True(t, matched)
}
