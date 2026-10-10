package engine

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVM_Register0(t *testing.T) {
	var vm VM
	vm.Register0(NewAtom("foo"), func(_ *VM, k Cont, env *Env) *Promise {
		return k(env)
	})
	p, _ := vm.procedures.Get(procedureIndicator{name: NewAtom("foo"), arity: 0})

	t.Run("ok", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{}, Success, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("wrong number of arguments", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a")}, Success, nil).Force(context.Background())
		assert.Error(t, err)
		assert.False(t, ok)

		assert.Equal(t, "wrong number of arguments: expected=0, actual=[a]", err.Error())
	})
}

func TestVM_Register1(t *testing.T) {
	var vm VM
	vm.Register1(NewAtom("foo"), func(_ *VM, a Term, k Cont, env *Env) *Promise {
		return k(env)
	})
	p, _ := vm.procedures.Get(procedureIndicator{name: NewAtom("foo"), arity: 1})

	t.Run("ok", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a")}, Success, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("wrong number of arguments", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a"), NewAtom("b")}, Success, nil).Force(context.Background())
		assert.Error(t, err)
		assert.False(t, ok)
	})
}

func TestVM_Register2(t *testing.T) {
	var vm VM
	vm.Register2(NewAtom("foo"), func(_ *VM, a, b Term, k Cont, env *Env) *Promise {
		return k(env)
	})
	p, _ := vm.procedures.Get(procedureIndicator{name: NewAtom("foo"), arity: 2})

	t.Run("ok", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a"), NewAtom("b")}, Success, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("wrong number of arguments", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a"), NewAtom("b"), NewAtom("c")}, Success, nil).Force(context.Background())
		assert.Error(t, err)
		assert.False(t, ok)
	})
}

func TestVM_Register3(t *testing.T) {
	var vm VM
	vm.Register3(NewAtom("foo"), func(_ *VM, a, b, c Term, k Cont, env *Env) *Promise {
		return k(env)
	})
	p, _ := vm.procedures.Get(procedureIndicator{name: NewAtom("foo"), arity: 3})

	t.Run("ok", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a"), NewAtom("b"), NewAtom("c")}, Success, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("wrong number of arguments", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a"), NewAtom("b"), NewAtom("c"), NewAtom("d")}, Success, nil).Force(context.Background())
		assert.Error(t, err)
		assert.False(t, ok)
	})
}

func TestVM_Register4(t *testing.T) {
	var vm VM
	vm.Register4(NewAtom("foo"), func(_ *VM, a, b, c, d Term, k Cont, env *Env) *Promise {
		return k(env)
	})
	p, _ := vm.procedures.Get(procedureIndicator{name: NewAtom("foo"), arity: 4})

	t.Run("ok", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a"), NewAtom("b"), NewAtom("c"), NewAtom("d")}, Success, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("wrong number of arguments", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a"), NewAtom("b"), NewAtom("c"), NewAtom("d"), NewAtom("e")}, Success, nil).Force(context.Background())
		assert.Error(t, err)
		assert.False(t, ok)
	})
}

func TestVM_Register5(t *testing.T) {
	var vm VM
	vm.Register5(NewAtom("foo"), func(_ *VM, a, b, c, d, e Term, k Cont, env *Env) *Promise {
		return k(env)
	})
	p, _ := vm.procedures.Get(procedureIndicator{name: NewAtom("foo"), arity: 5})

	t.Run("ok", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a"), NewAtom("b"), NewAtom("c"), NewAtom("d"), NewAtom("e")}, Success, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("wrong number of arguments", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a"), NewAtom("b"), NewAtom("c"), NewAtom("d"), NewAtom("e"), NewAtom("f")}, Success, nil).Force(context.Background())
		assert.Error(t, err)
		assert.False(t, ok)
	})
}

func TestVM_Register6(t *testing.T) {
	var vm VM
	vm.Register6(NewAtom("foo"), func(_ *VM, a, b, c, d, e, f Term, k Cont, env *Env) *Promise {
		return k(env)
	})
	p, _ := vm.procedures.Get(procedureIndicator{name: NewAtom("foo"), arity: 6})

	t.Run("ok", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a"), NewAtom("b"), NewAtom("c"), NewAtom("d"), NewAtom("e"), NewAtom("f")}, Success, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("wrong number of arguments", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a"), NewAtom("b"), NewAtom("c"), NewAtom("d"), NewAtom("e"), NewAtom("f"), NewAtom("g")}, Success, nil).Force(context.Background())
		assert.Error(t, err)
		assert.False(t, ok)
	})
}

func TestVM_Register7(t *testing.T) {
	var vm VM
	vm.Register7(NewAtom("foo"), func(_ *VM, a, b, c, d, e, f, g Term, k Cont, env *Env) *Promise {
		return k(env)
	})
	p, _ := vm.procedures.Get(procedureIndicator{name: NewAtom("foo"), arity: 7})

	t.Run("ok", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a"), NewAtom("b"), NewAtom("c"), NewAtom("d"), NewAtom("e"), NewAtom("f"), NewAtom("g")}, Success, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("wrong number of arguments", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a"), NewAtom("b"), NewAtom("c"), NewAtom("d"), NewAtom("e"), NewAtom("f"), NewAtom("g"), NewAtom("h")}, Success, nil).Force(context.Background())
		assert.Error(t, err)
		assert.False(t, ok)
	})
}

func TestVM_Register8(t *testing.T) {
	var vm VM
	vm.Register8(NewAtom("foo"), func(_ *VM, a, b, c, d, e, f, g, h Term, k Cont, env *Env) *Promise {
		return k(env)
	})
	p, _ := vm.procedures.Get(procedureIndicator{name: NewAtom("foo"), arity: 8})

	t.Run("ok", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a"), NewAtom("b"), NewAtom("c"), NewAtom("d"), NewAtom("e"), NewAtom("f"), NewAtom("g"), NewAtom("h")}, Success, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("wrong number of arguments", func(t *testing.T) {
		ok, err := p.call(&vm, []Term{NewAtom("a"), NewAtom("b"), NewAtom("c"), NewAtom("d"), NewAtom("e"), NewAtom("f"), NewAtom("g"), NewAtom("h"), NewAtom("i")}, Success, nil).Force(context.Background())
		assert.Error(t, err)
		assert.False(t, ok)
	})
}

func TestVM_Arrive(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		vm := VM{
			procedures: buildOrderedMap(
				procedurePair{
					Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
					Value: Predicate1(func(_ *VM, t Term, k Cont, env *Env) *Promise {
						return k(env)
					}),
				},
			),
		}
		ok, err := vm.Arrive(NewAtom("foo"), []Term{NewAtom("a")}, Success, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("unknown procedure", func(t *testing.T) {
		t.Run("error", func(t *testing.T) {
			vm := VM{
				unknown: unknownError,
			}
			ok, err := vm.Arrive(NewAtom("foo"), []Term{NewAtom("a")}, Success, nil).Force(context.Background())
			assert.Equal(t, existenceError(objectTypeProcedure, &compound{
				functor: atomSlash,
				args:    []Term{NewAtom("foo"), Integer(1)},
			}, nil), err)
			assert.False(t, ok)
		})

		t.Run("warning", func(t *testing.T) {
			var warned bool
			vm := VM{
				unknown: unknownWarning,
				Unknown: func(name Atom, args []Term, env *Env) {
					assert.Equal(t, NewAtom("foo"), name)
					assert.Equal(t, []Term{NewAtom("a")}, args)
					warned = true
				},
			}
			ok, err := vm.Arrive(NewAtom("foo"), []Term{NewAtom("a")}, Success, nil).Force(context.Background())
			assert.NoError(t, err)
			assert.False(t, ok)
			assert.True(t, warned)
		})

		t.Run("fail", func(t *testing.T) {
			vm := VM{
				unknown: unknownFail,
			}
			ok, err := vm.Arrive(NewAtom("foo"), []Term{NewAtom("a")}, Success, nil).Force(context.Background())
			assert.NoError(t, err)
			assert.False(t, ok)
		})
	})
}

func TestVM_open_nilFS(t *testing.T) {
	var vm VM
	env := vm.NewEnv()
	_, _, err := vm.open(NewAtom("foo"), env)
	assert.Equal(t, permissionError(operationOpen, permissionTypeSourceSink, NewAtom("foo"), env), err)
}

func TestVM_LoadedSources(t *testing.T) {
	vm := VM{FS: testdata}
	assert.Nil(t, vm.LoadedSources())

	ok, err := Consult(&vm, List(
		NewAtom("testdata/foo"),
		NewAtom("testdata/empty.txt"),
		NewAtom("testdata/empty.txt"),
	), Success, nil).Force(context.Background())
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, []string{"testdata/foo.pl", "testdata/empty.txt"}, vm.LoadedSources())

	loaded := vm.LoadedSources()
	loaded[0] = "mutated"
	assert.Equal(t, []string{"testdata/foo.pl", "testdata/empty.txt"}, vm.LoadedSources())
}

func TestVM_SetUserInput(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		var vm VM
		vm.SetUserInput(vm.NewInputTextStream(os.Stdin))

		s, ok := vm.streams.lookup(atomUserInput)
		assert.True(t, ok)
		assert.Equal(t, os.Stdin, s.source)
	})
}

func TestVM_SetUserOutput(t *testing.T) {
	t.Run("file", func(t *testing.T) {
		var vm VM
		vm.SetUserOutput(vm.NewOutputTextStream(os.Stdout))

		s, ok := vm.streams.lookup(atomUserOutput)
		assert.True(t, ok)
		assert.Equal(t, os.Stdout, s.sink)
	})
}

func TestVM_SetUserStreamRejectsForeignOrUnowned(t *testing.T) {
	tests := []struct {
		title     string
		alias     Atom
		newStream func(*VM) *Stream
		set       func(*VM, *Stream)
		current   func(*VM) *Stream
	}{
		{
			title:     "input",
			alias:     atomUserInput,
			newStream: func(vm *VM) *Stream { return vm.NewInputTextStream(nil) },
			set:       func(vm *VM, s *Stream) { vm.SetUserInput(s) },
			current:   func(vm *VM) *Stream { return vm.input },
		},
		{
			title:     "output",
			alias:     atomUserOutput,
			newStream: func(vm *VM) *Stream { return vm.NewOutputTextStream(nil) },
			set:       func(vm *VM, s *Stream) { vm.SetUserOutput(s) },
			current:   func(vm *VM) *Stream { return vm.output },
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			var vm, other VM
			current := tt.newStream(&vm)
			tt.set(&vm, current)
			otherCurrent := tt.newStream(&other)
			tt.set(&other, otherCurrent)

			foreignAlias := NewAtom("foreign_" + tt.title)
			foreign := tt.newStream(&other)
			foreign.alias = foreignAlias
			other.streams.add(foreign)

			assertUnchanged := func() {
				assert.Same(t, current, tt.current(&vm))
				assert.Same(t, otherCurrent, tt.current(&other))
				assert.True(t, current.ownedBy(&vm))
				assert.True(t, otherCurrent.ownedBy(&other))
				assert.Len(t, vm.streams.elems, 1)
				assert.Len(t, other.streams.elems, 2)

				s, ok := vm.streams.lookup(tt.alias)
				assert.True(t, ok)
				assert.Same(t, current, s)
				s, ok = other.streams.lookup(tt.alias)
				assert.True(t, ok)
				assert.Same(t, otherCurrent, s)
				s, ok = other.streams.lookup(foreignAlias)
				assert.True(t, ok)
				assert.Same(t, foreign, s)
			}

			assert.PanicsWithValue(t, ErrStreamScope, func() {
				tt.set(&vm, foreign)
			})
			assertUnchanged()
			assert.True(t, foreign.ownedBy(&other))
			assert.Equal(t, foreignAlias, foreign.alias)

			unownedAlias := NewAtom("unowned_" + tt.title)
			unowned := &Stream{alias: unownedAlias}
			assert.PanicsWithValue(t, ErrStreamScope, func() {
				tt.set(&vm, unowned)
			})
			assertUnchanged()
			assert.False(t, unowned.ownedBy(&vm))
			assert.False(t, unowned.ownedBy(&other))
			assert.Equal(t, unownedAlias, unowned.alias)
			_, ok := vm.streams.lookup(unownedAlias)
			assert.False(t, ok)
		})
	}
}

func TestProcedureIndicator_Apply(t *testing.T) {
	t.Run("ok", func(t *testing.T) {
		c, err := procedureIndicator{name: NewAtom("foo"), arity: 2}.Apply(NewAtom("a"), NewAtom("b"))
		assert.NoError(t, err)
		assert.Equal(t, &compound{
			functor: NewAtom("foo"),
			args:    []Term{NewAtom("a"), NewAtom("b")},
		}, c)
	})

	t.Run("wrong number of arguments", func(t *testing.T) {
		c, err := procedureIndicator{name: NewAtom("foo"), arity: 2}.Apply(NewAtom("a"), NewAtom("b"), NewAtom("c"))
		assert.Error(t, err)
		assert.Nil(t, c)
	})
}

func TestVM_EnvironmentIsolation(t *testing.T) {
	var a, b VM
	x, y := a.NewVariable(), b.NewVariable()
	envA := a.NewEnv().bind(x, NewAtom("a"))
	envB := b.NewEnv().bind(y, NewAtom("b"))
	assert.False(t, x == y)
	assert.Equal(t, NewAtom("a"), envA.Resolve(x))
	assert.Equal(t, NewAtom("b"), envB.Resolve(y))
	assert.PanicsWithValue(t, ErrVariableScope, func() { envA.Resolve(y) })
	assert.PanicsWithValue(t, ErrVariableScope, func() { envB.Resolve(x) })
	assert.PanicsWithValue(t, ErrVariableScope, func() { a.ownedEnv(envB) })
}

func TestVM_ExceptionVariablesUseEnvironmentOwner(t *testing.T) {
	var vm VM
	vm.SetMaxVariables(10)
	env := vm.NewEnv()
	for n := range 8 {
		env = env.bind(vm.NewVariable(), Integer(n))
	}
	x := vm.NewVariable()
	ex := NewException(NewAtom("f").Apply(x, x), env)
	c := ex.Term().(Compound)
	assert.NotEqual(t, x, c.Arg(0))
	assert.Equal(t, c.Arg(0), c.Arg(1))
	assert.Equal(t, x, env.Resolve(x))
	assert.PanicsWithValue(t, ErrMaxVariables, func() { vm.NewVariable() })
}

func TestVM_DebugHook(t *testing.T) {
	var vm VM
	vm.Register0(NewAtom("foo"), func(_ *VM, k Cont, env *Env) *Promise {
		return k(env)
	})

	buf := &bytes.Buffer{}
	vm.InstallHook(DebugHookFn(buf))

	var env Env
	ok, err := Call(&vm, NewAtom("foo"), Success, &env).Force(context.Background())
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "enter\ncall(foo/0)\nexit\n", buf.String())
}

func TestVM_Meter(t *testing.T) {
	t.Run("install meter counts instructions", func(t *testing.T) {
		var vm VM
		vm.Register0(NewAtom("foo"), func(_ *VM, k Cont, env *Env) *Promise {
			return k(env)
		})

		var count uint64
		vm.InstallMeter(func(kind MeterKind, units uint64) Term {
			if kind == MeterInstruction {
				count += units
			}
			return nil
		})

		ok, err := Call(&vm, NewAtom("foo"), Success, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, uint64(3), count)
		assert.NotNil(t, vm.meter)
	})

	t.Run("clear meter disables charging", func(t *testing.T) {
		var vm VM
		vm.Register0(NewAtom("foo"), func(_ *VM, k Cont, env *Env) *Promise {
			return k(env)
		})

		var count uint64
		vm.InstallMeter(func(kind MeterKind, units uint64) Term {
			if kind == MeterInstruction {
				count += units
			}
			return nil
		})
		vm.ClearMeter()

		ok, err := Call(&vm, NewAtom("foo"), Success, nil).Force(context.Background())
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Zero(t, count)
		assert.Nil(t, vm.meter)
	})
}

func TestInstruction_String(t *testing.T) {
	t.Run("with operand", func(t *testing.T) {
		instr := instruction{
			opcode:  OpCall,
			operand: NewAtom("foo"),
		}
		expected := "call(foo)"
		assert.Equal(t, expected, instr.String())
	})

	t.Run("without operand", func(t *testing.T) {
		instr := instruction{
			opcode: OpExit,
		}
		expected := "exit()"
		assert.Equal(t, expected, instr.String())
	})
}

func TestVM_RejectsForeignScopeAtExecutionBoundaries(t *testing.T) {
	var vm, other VM
	called := false
	vm.Register1(NewAtom("ignore"), func(_ *VM, _ Term, k Cont, env *Env) *Promise {
		called = true
		return k(env)
	})

	tests := []struct {
		title string
		arg   Term
		err   error
	}{
		{
			title: "variable",
			arg:   NewAtom("nested").Apply(other.NewVariable()),
			err:   ErrVariableScope,
		},
		{
			title: "foreign stream",
			arg:   NewAtom("nested").Apply(other.NewInputTextStream(nil)),
			err:   ErrStreamScope,
		},
		{
			title: "unowned stream",
			arg:   NewAtom("nested").Apply(&Stream{}),
			err:   ErrStreamScope,
		},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			ok, err := vm.Arrive(NewAtom("ignore"), []Term{tt.arg}, Success, nil).Force(context.Background())
			assert.False(t, ok)
			assert.ErrorIs(t, err, tt.err)
			ok, err = Call(&vm, NewAtom("ignore").Apply(tt.arg), Success, nil).Force(context.Background())
			assert.False(t, ok)
			assert.ErrorIs(t, err, tt.err)
			assert.False(t, called)
		})
	}

	ok, err := vm.Arrive(NewAtom("ignore"), []Term{NewAtom("ground")}, Success, other.NewEnv()).Force(context.Background())
	assert.False(t, ok)
	assert.ErrorIs(t, err, ErrVariableScope)
	assert.False(t, called)
}
