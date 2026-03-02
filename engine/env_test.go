package engine

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnv_Bind(t *testing.T) {
	a := NewVariable()

	var env *Env
	assert.Equal(t, &Env{
		color: black,
		left: &Env{
			binding: binding{
				key:   newEnvKey(a),
				value: NewAtom("a"),
			},
		},
		binding: binding{
			key:   newEnvKey(varContext),
			value: NewAtom("root"),
		},
	}, env.bind(a, NewAtom("a")))
}

func TestEnv_Lookup(t *testing.T) {
	vars := make([]Variable, 1000)
	for i := range vars {
		vars[i] = NewVariable()
	}

	rand.Shuffle(len(vars), func(i, j int) {
		vars[i], vars[j] = vars[j], vars[i]
	})

	var env *Env
	for _, v := range vars {
		env = env.bind(v, v)
	}

	rand.Shuffle(len(vars), func(i, j int) {
		vars[i], vars[j] = vars[j], vars[i]
	})

	for _, v := range vars {
		t.Run(fmt.Sprintf("_%d", v), func(t *testing.T) {
			w, ok := env.lookup(v)
			assert.True(t, ok)
			assert.Equal(t, v, w)
		})
	}
}

func TestEnv_BindPreservesMeter(t *testing.T) {
	m := func(kind MeterKind, units uint64) Term {
		return nil
	}

	env := NewEnv().withMeter(m)
	for i := 0; i < 8; i++ {
		v := NewVariable()
		env = env.bind(v, v)
	}

	var walk func(*Env)
	walk = func(e *Env) {
		if e == nil {
			return
		}
		assert.Equal(t, reflect.ValueOf(m).Pointer(), reflect.ValueOf(e.meter).Pointer())
		walk(e.left)
		walk(e.right)
	}

	walk(env)
}

func TestEnv_WithMeter(t *testing.T) {
	t.Run("nil env and nil meter", func(t *testing.T) {
		var env *Env
		assert.Nil(t, env.withMeter(nil))
	})

	t.Run("nil env and meter", func(t *testing.T) {
		m := func(kind MeterKind, units uint64) Term {
			return nil
		}

		var env *Env
		metered := env.withMeter(m)
		if assert.NotNil(t, metered) {
			assert.Equal(t, rootEnv.key, metered.key)
			assert.Equal(t, rootEnv.value, metered.value)
			assert.Equal(t, reflect.ValueOf(m).Pointer(), reflect.ValueOf(metered.meter).Pointer())
		}
	})
}

func TestEnv_WithoutMeter(t *testing.T) {
	t.Run("nil env", func(t *testing.T) {
		var env *Env
		assert.Nil(t, env.withoutMeter())
	})

	t.Run("clears meter", func(t *testing.T) {
		m := func(kind MeterKind, units uint64) Term {
			return nil
		}

		env := NewEnv().withMeter(m)
		cleared := env.withoutMeter()
		if assert.NotNil(t, cleared) {
			assert.Nil(t, cleared.meter)
			assert.Equal(t, env.key, cleared.key)
			assert.Equal(t, env.value, cleared.value)
		}
	})
}

func TestEnv_Simplify(t *testing.T) {
	// L = [a, b|L] ==> [a, b, a, b, ...]
	l := NewVariable()
	p := PartialList(l, NewAtom("a"), NewAtom("b"))
	env := NewEnv().bind(l, p)
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

func TestContains(t *testing.T) {
	var env *Env
	assert.True(t, contains(NewAtom("a"), NewAtom("a"), env))
	assert.False(t, contains(NewVariable(), NewAtom("a"), env))
	v := NewVariable()
	env = env.bind(v, NewAtom("a"))
	assert.True(t, contains(v, NewAtom("a"), env))
	assert.True(t, contains(&compound{functor: NewAtom("a")}, NewAtom("a"), env))
	assert.True(t, contains(&compound{functor: NewAtom("f"), args: []Term{NewAtom("a")}}, NewAtom("a"), env))
	assert.False(t, contains(&compound{functor: NewAtom("f")}, NewAtom("a"), env))
}
