package engine

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVariable_WriteTerm(t *testing.T) {
	var vm VM
	x := vm.NewVariable()

	tests := []struct {
		title  string
		v      Variable
		w      io.StringWriter
		opts   WriteOptions
		output string
	}{
		{title: "unnamed", v: x, output: fmt.Sprintf("_%d", x.index)},
		{title: "variable_names", v: x, opts: WriteOptions{variableNames: map[Variable]Atom{x: NewAtom("Foo")}}, output: `Foo`},
		{title: "following a letter-digit operator", v: x, opts: WriteOptions{left: operator{name: NewAtom("is")}}, output: fmt.Sprintf(" _%d", x.index)},
		{title: "followed by a letter-digit operator", v: x, opts: WriteOptions{right: operator{name: NewAtom("is")}}, output: fmt.Sprintf("_%d ", x.index)},
	}

	var buf bytes.Buffer
	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			buf.Reset()
			assert.NoError(t, tt.v.WriteTerm(&buf, &tt.opts, nil))
			assert.Equal(t, tt.output, buf.String())
		})
	}
}

func TestVariable_Compare(t *testing.T) {
	var vm VM
	w, x, y := vm.NewVariable(), vm.NewVariable(), vm.NewVariable()

	tests := []struct {
		title string
		v     Variable
		t     Term
		o     int
	}{
		{title: `X > W`, v: x, t: w, o: 1},
		{title: `X = X`, v: x, t: x, o: 0},
		{title: `X < Y`, v: x, t: y, o: -1},
		{title: `X < 0.0`, v: x, t: NewFloatFromInt64(0), o: -1},
		{title: `X < 0`, v: x, t: Integer(0), o: -1},
		{title: `X < a`, v: x, t: NewAtom("a"), o: -1},
		{title: `X < f(a)`, v: x, t: NewAtom("f").Apply(NewAtom("a")), o: -1},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			assert.Equal(t, tt.o, tt.v.Compare(tt.t, nil))
		})
	}
}

func Test_variableSet(t *testing.T) {
	var vm VM
	f := NewAtom("f")
	x, y := vm.NewVariable(), vm.NewVariable()

	tests := []struct {
		term Term
		s    variableSet
	}{
		{term: f.Apply(x, y), s: map[Variable]int{
			x: 1,
			y: 1,
		}},
		{term: f.Apply(y, x), s: map[Variable]int{
			x: 1,
			y: 1,
		}},
		{term: atomPlus.Apply(x, y), s: map[Variable]int{
			x: 1,
			y: 1,
		}},
		{term: atomMinus.Apply(y, atomMinus.Apply(x, x)), s: map[Variable]int{
			x: 2,
			y: 1,
		}},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.s, newVariableSet(tt.term, nil))
	}
}

func Test_existentialVariableSet(t *testing.T) {
	var vm VM
	f := NewAtom("f")
	x, y, z := vm.NewVariable(), vm.NewVariable(), vm.NewVariable()

	tests := []struct {
		term Term
		ev   variableSet
	}{
		{term: atomCaret.Apply(x, atomCaret.Apply(y, f.Apply(x, y, z))), ev: variableSet{
			x: 1,
			y: 1,
		}},
		{term: atomCaret.Apply(atomComma.Apply(x, y), f.Apply(z, y, x)), ev: variableSet{
			x: 1,
			y: 1,
		}},
		{term: atomCaret.Apply(atomPlus.Apply(x, y), Integer(3)), ev: variableSet{
			x: 1,
			y: 1,
		}},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.ev, newExistentialVariablesSet(tt.term, nil))
	}
}

func Test_freeVariablesSet(t *testing.T) {
	var vm VM
	f := NewAtom("f")
	x, y, z := vm.NewVariable(), vm.NewVariable(), vm.NewVariable()
	a := vm.NewVariable()

	tests := []struct {
		t, v Term
		fv   variableSet
	}{
		{t: atomPlus.Apply(x, atomPlus.Apply(y, z)), v: f.Apply(z), fv: variableSet{
			x: 1,
			y: 1,
		}},
		{t: atomCaret.Apply(z, atomPlus.Apply(a, atomPlus.Apply(x, atomPlus.Apply(y, z)))), v: a, fv: variableSet{
			x: 1,
			y: 1,
		}},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.fv, newFreeVariablesSet(tt.t, tt.v, nil))
	}
}

func TestVM_NewVariableLimits(t *testing.T) {
	var a, b VM
	a.SetMaxVariables(2)
	a.NewVariable()
	a.NewVariable()
	assert.PanicsWithValue(t, ErrMaxVariables, func() { a.NewVariable() })

	b.SetMaxVariables(0)
	for range 4 {
		b.NewVariable()
	}
	assert.PanicsWithValue(t, ErrMaxVariables, func() { a.NewVariable() })

	a.SetMaxVariables(0)
	assert.Equal(t, int64(3), a.NewVariable().index)
	a.SetMaxVariables(3)
	assert.PanicsWithValue(t, ErrMaxVariables, func() { a.NewVariable() })
	assert.Equal(t, int64(5), b.NewVariable().index)
}

func TestVariable_ScopeIsolation(t *testing.T) {
	var a, b VM
	x, y := a.NewVariable(), b.NewVariable()
	assert.False(t, x == y)
	env := a.NewEnv().bind(x, NewAtom("a"))
	assert.Equal(t, NewAtom("a"), env.Resolve(x))
	assert.PanicsWithValue(t, ErrVariableScope, func() { env.Resolve(y) })
	assert.PanicsWithValue(t, ErrVariableScope, func() { x.Compare(y, nil) })
	assert.PanicsWithValue(t, ErrVariableScope, func() {
		var empty *Env
		empty.Unify(NewAtom("f").Apply(x), NewAtom("f").Apply(y))
	})
	assert.PanicsWithValue(t, ErrVariableScope, func() {
		env.Unify(x, NewAtom("f").Apply(y))
	})
}

func TestEnv_UnifyRejectsForeignOrUnownedStreams(t *testing.T) {
	var vm, other VM
	x := vm.NewVariable()

	tests := []struct {
		title  string
		stream *Stream
	}{
		{title: "foreign", stream: other.NewInputTextStream(nil)},
		{title: "unowned", stream: &Stream{}},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			assert.PanicsWithValue(t, ErrStreamScope, func() {
				vm.NewEnv().Unify(
					NewAtom("outer").Apply(x),
					NewAtom("outer").Apply(NewAtom("nested").Apply(tt.stream)),
				)
			})
		})
	}
}

func TestVariable_DeterministicRenderingAndOrdering(t *testing.T) {
	for noise := range 8 {
		var vm, other VM
		x := vm.NewVariable()
		for range noise {
			other.NewVariable()
		}
		y := vm.NewVariable()
		var out bytes.Buffer
		assert.NoError(t, NewAtom("pair").Apply(x, y).WriteTerm(&out, &defaultWriteOptions, vm.NewEnv()))
		assert.Equal(t, "pair(_1,_2)", out.String())
		assert.Equal(t, -1, x.Compare(y, vm.NewEnv()))
	}
}

func TestVariable_ScopeValidationHandlesCycles(t *testing.T) {
	var vm, other VM
	x := vm.NewVariable()
	c := &compound{functor: NewAtom("cycle"), args: []Term{x, nil}}
	c.args[1] = c
	env, ok := vm.NewEnv().Unify(x, c)
	assert.True(t, ok)
	assert.Equal(t, c, env.Resolve(x))
	assert.PanicsWithValue(t, ErrVariableScope, func() {
		other.NewEnv().Unify(other.NewVariable(), c)
	})
}
