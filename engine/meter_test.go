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
		left[i] = vm.NewVariable()
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

	x := vm.NewVariable()
	goal := NewAtom("copy_term").Apply(
		NewAtom("f").Apply(x, List(NewAtom("a"))),
		vm.NewVariable(),
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
		vm.NewVariable(),
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
		vm.NewVariable(),
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

	ok, err := Call(&vm, atomEqual.Apply(vm.NewVariable(), Integer(1)), Success, nil).Force(context.Background())
	assert.False(t, ok)
	ex, okCast := err.(Exception)
	assert.True(t, okCast)
	pattern := atomError.Apply(
		NewAtom("resource_error").Apply(NewAtom("gas")),
		atomSlash.Apply(atomEqual, Integer(2)),
	)
	_, matched := vm.NewEnv().Unify(pattern, ex.Term())
	assert.True(t, matched)
}

func TestRecoverMeterError_PreservesNonMeterPanics(t *testing.T) {
	want := ErrVariableScope
	var err error
	assert.PanicsWithValue(t, want, func() {
		func() {
			defer recoverMeterError(&err)
			panic(want)
		}()
	})
	assert.NoError(t, err)
}

func TestDesugarPred_MetersPartialListTail(t *testing.T) {
	var vm VM
	var charged uint64
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		if kind == MeterTermCell {
			charged += units
		}
		return nil
	})

	_, _ = desugarPred(&vm, PartialList(NewAtom("tail"), NewAtom("a"), NewAtom("b")), nil, vm.NewEnv())

	assert.Equal(t, uint64(3), charged)
}

func TestVMExec_MetersCallerSpareCapacity(t *testing.T) {
	var vm VM
	var charged uint64
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		if kind == MeterTermCell {
			charged += units
		}
		return nil
	})

	args := make([]Term, 0, 1)
	ok, err := vm.exec(
		bytecode{
			{opcode: OpPutConst, operand: NewAtom("a")},
			{opcode: OpExit},
		},
		nil,
		Success,
		args,
		nil,
		nil,
		nil,
	).Force(context.Background())

	assert.True(t, ok)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), charged)
}

func TestVM_MeterTermCell_CompiledTerms(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   Term
		cells  uint64
	}{
		{"get functor", "p(f(a,b,c)).", NewAtom("f").Apply(NewAtom("a"), NewAtom("b"), NewAtom("c")), 3},
		{"get list", "p([a,b,c]).", List(NewAtom("a"), NewAtom("b"), NewAtom("c")), 3},
		{"get dict", "p(tag{k:v}).", newDict([]Term{NewAtom("tag"), NewAtom("k"), NewAtom("v")}), 3},
		{"get partial", "p([a,b|z]).", PartialList(NewAtom("z"), NewAtom("a"), NewAtom("b")), 3},
		{"put functor", "p(X) :- X = f(a,b,c).", NewAtom("f").Apply(NewAtom("a"), NewAtom("b"), NewAtom("c")), 6},
		{"put list", "p(X) :- X = [a,b,c].", List(NewAtom("a"), NewAtom("b"), NewAtom("c")), 6},
		{"put dict", "p(X) :- X = tag{k:v}.", newDict([]Term{NewAtom("tag"), NewAtom("k"), NewAtom("v")}), 6},
		{"put partial", "p(X) :- X = [a,b|z].", PartialList(NewAtom("z"), NewAtom("a"), NewAtom("b")), 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, enough := range []bool{false, true} {
				var vm VM
				vm.Register2(atomEqual, Unify)
				vm.getOperators().define(1200, operatorSpecifierXFX, atomIf)
				vm.getOperators().define(700, operatorSpecifierXFX, atomEqual)
				if !assert.NoError(t, vm.Compile(context.Background(), tt.source)) {
					return
				}
				remaining := tt.cells
				if !enough {
					remaining--
				}
				vm.InstallMeter(func(kind MeterKind, units uint64) Term {
					if kind == MeterTermCell {
						if units > remaining {
							return atomResourceError.Apply(atomMemory)
						}
						remaining -= units
					}
					return nil
				})
				x := vm.NewVariable()
				matched := false
				ok, err := vm.Arrive(NewAtom("p"), []Term{x}, func(env *Env) *Promise {
					matched = tt.want.Compare(x, env) == 0
					return Bool(matched)
				}, nil).Force(context.Background())
				if enough {
					assert.NoError(t, err)
					assert.True(t, ok)
					assert.True(t, matched)
					assert.Zero(t, remaining)
				} else {
					assert.False(t, ok)
					assert.False(t, matched)
					assert.Equal(t, Exception{term: atomError.Apply(atomResourceError.Apply(atomMemory), atomSlash.Apply(NewAtom("p"), Integer(1)))}, err)
				}
			}
		})
	}
}

func TestVM_MeterTermCell_IndependentQuotas(t *testing.T) {
	newVM := func() *VM {
		vm := &VM{}
		if !assert.NoError(t, vm.Compile(context.Background(), "p([a,b,c]).")) {
			t.FailNow()
		}
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
		return vm
	}
	call := func(vm *VM) (bool, error) {
		x := vm.NewVariable()
		return vm.Arrive(NewAtom("p"), []Term{x}, func(env *Env) *Promise {
			return Bool(List(NewAtom("a"), NewAtom("b"), NewAtom("c")).Compare(x, env) == 0)
		}, nil).Force(context.Background())
	}
	a, b := newVM(), newVM()
	ok, err := call(a)
	assert.NoError(t, err)
	assert.True(t, ok)
	ok, err = call(a)
	assert.False(t, ok)
	assert.Equal(t, Exception{term: atomError.Apply(atomResourceError.Apply(atomMemory), atomSlash.Apply(NewAtom("p"), Integer(1)))}, err)
	ok, err = call(b)
	assert.NoError(t, err)
	assert.True(t, ok)
}
