package engine

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnv_Bind(t *testing.T) {
	var vm VM
	a := vm.NewVariable()
	env := vm.prepareEnv(nil)
	bound := env.bind(a, NewAtom("a"))
	assert.Equal(t, a, env.Resolve(a))
	assert.Equal(t, NewAtom("a"), bound.Resolve(a))
	assert.Equal(t, rootContext, bound.Resolve(varContext))
}

func TestEnv_Lookup(t *testing.T) {
	var vm VM
	vars := make([]Variable, 1000)
	for i := range vars {
		vars[i] = vm.NewVariable()
	}

	random := rand.New(rand.NewSource(0))
	random.Shuffle(len(vars), func(i, j int) {
		vars[i], vars[j] = vars[j], vars[i]
	})

	env := vm.NewEnv()
	for _, v := range vars {
		env = env.bind(v, v)
	}

	random.Shuffle(len(vars), func(i, j int) {
		vars[i], vars[j] = vars[j], vars[i]
	})

	for _, v := range vars {
		t.Run(fmt.Sprintf("_%d", v.index), func(t *testing.T) {
			w, ok := env.lookup(v)
			assert.True(t, ok)
			assert.Equal(t, v, w)
			assert.True(t, env.Resolve(v) == v)
		})
	}
}

func TestEnv_Simplify(t *testing.T) {
	var vm VM
	// L = [a, b|L] ==> [a, b, a, b, ...]
	l := vm.NewVariable()
	p := PartialList(l, NewAtom("a"), NewAtom("b"))
	env := vm.NewEnv().bind(l, p)
	c := env.simplify(l)
	iter := ListIterator{List: c, Env: env}
	assert.True(t, iter.Next())
	assert.Equal(t, NewAtom("a"), iter.Current())
	assert.True(t, iter.Next())
	assert.Equal(t, NewAtom("b"), iter.Current())
	assert.False(t, iter.Next())
	suffix, ok := iter.Suffix().(*partial)
	assert.True(t, ok)
	assert.Equal(t, atomDot, suffix.Functor())
	assert.Equal(t, 2, suffix.Arity())
}

func TestEnv_Simplify_MetersPartialListTail(t *testing.T) {
	var vm VM
	var charged uint64
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		if kind == MeterTermCell {
			charged += units
		}
		return nil
	})

	tail := vm.NewVariable()
	env := vm.NewEnv()
	_ = env.simplify(PartialList(tail, NewAtom("a"), NewAtom("b")))

	assert.Equal(t, uint64(3), charged)
}

func TestContains(t *testing.T) {
	var vm VM
	var env *Env
	assert.True(t, contains(NewAtom("a"), NewAtom("a"), env))
	assert.False(t, contains(vm.NewVariable(), NewAtom("a"), env))
	v := vm.NewVariable()
	env = vm.NewEnv().bind(v, NewAtom("a"))
	assert.True(t, contains(v, NewAtom("a"), env))
	assert.True(t, contains(&compound{functor: NewAtom("a")}, NewAtom("a"), env))
	assert.True(t, contains(&compound{functor: NewAtom("f"), args: []Term{NewAtom("a")}}, NewAtom("a"), env))
	assert.False(t, contains(&compound{functor: NewAtom("f")}, NewAtom("a"), env))
}

func TestEnv_MeteredBaseAdoptsScopeWithoutMutation(t *testing.T) {
	var vm, other VM
	x, y := vm.NewVariable(), other.NewVariable()
	var charged uint64
	var empty *Env
	base := empty.withMeter(func(kind MeterKind, units uint64) Term {
		if kind == MeterUnifyStep {
			charged += units
		}
		return nil
	})

	bound := base.bind(x, NewAtom("a"))
	fork, ok := base.Unify(y, NewAtom("b"))
	assert.True(t, ok)
	assert.Equal(t, NewAtom("a"), bound.Resolve(x))
	assert.Equal(t, NewAtom("b"), fork.Resolve(y))
	assert.Equal(t, x, base.Resolve(x))
	assert.Equal(t, y, base.Resolve(y))
	assert.Equal(t, rootContext, bound.Resolve(varContext))
	assert.Equal(t, uint64(1), charged)

	_, ok = bound.Unify(x, NewAtom("a"))
	assert.True(t, ok)
	assert.Equal(t, uint64(2), charged)
	assert.PanicsWithValue(t, ErrVariableScope, func() { bound.Resolve(y) })
	assert.PanicsWithValue(t, ErrVariableScope, func() { bound.bind(y, NewAtom("wrong")) })
	assert.Equal(t, NewAtom("a"), bound.Resolve(x))

	unmetered := bound.withoutMeter()
	_, ok = unmetered.Unify(x, NewAtom("a"))
	assert.True(t, ok)
	assert.Equal(t, uint64(2), charged)
	_, ok = bound.Unify(x, NewAtom("a"))
	assert.True(t, ok)
	assert.Equal(t, uint64(3), charged)
}

func TestEnv_UnifyDoesNotCoerceAtomsToIntegers(t *testing.T) {
	var env *Env
	_, ok := env.Unify(NewAtom("1"), Integer(1))
	assert.False(t, ok)
}
