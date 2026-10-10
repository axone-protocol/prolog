package engine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewException(t *testing.T) {
	assert.Equal(t, Exception{term: NewAtom("foo").Apply(NewAtom("bar"))}, NewException(NewAtom("foo").Apply(NewAtom("bar")), nil))

	assert.Equal(t,
		Exception{
			term: NewAtom("foo").Apply(
				newDict([]Term{NewAtom("point"), NewAtom("x"), Integer(0), NewAtom("y"), Integer(1)}),
			),
		},
		NewException(NewAtom("foo").Apply(newDict([]Term{NewAtom("point"), NewAtom("x"), Integer(0), NewAtom("y"), Integer(1)})), nil))
}

func TestNewException_QuotaRefusal(t *testing.T) {
	var vm VM
	calls := 0
	vm.InstallMeter(func(kind MeterKind, units uint64) Term {
		if kind == MeterTermCell {
			calls++
			return atomResourceError.Apply(atomMemory)
		}
		return nil
	})
	exception := NewException(NewAtom("f").Apply(NewAtom("a")), vm.prepareEnv(nil))
	assert.Equal(t, Exception{term: atomError.Apply(atomResourceError.Apply(atomMemory), rootContext)}, exception)
	assert.Equal(t, 1, calls)
}

func TestNewException_PreservesForeignVariablePanic(t *testing.T) {
	var owner, foreign VM
	assert.PanicsWithValue(t, ErrVariableScope, func() {
		_ = NewException(foreign.NewVariable(), owner.NewEnv())
	})
}

func TestException_Error(t *testing.T) {
	e := Exception{term: NewAtom("foo")}
	assert.Equal(t, "foo", e.Error())
}

func TestInstantiationError(t *testing.T) {
	assert.Equal(t, Exception{
		term: atomError.Apply(atomInstantiationError, rootContext),
	}, InstantiationError(nil))
}

func TestDomainError(t *testing.T) {
	assert.Equal(t, Exception{
		term: atomError.Apply(
			atomDomainError.Apply(atomNotLessThanZero, Integer(-1)),
			rootContext,
		),
	}, DomainError(atomNotLessThanZero, Integer(-1), nil))
}

func TestTypeError(t *testing.T) {
	assert.Equal(t, Exception{
		term: atomError.Apply(
			atomTypeError.Apply(atomAtom, Integer(0)),
			rootContext,
		),
	}, TypeError(atomAtom, Integer(0), nil))
}

func TestExceptionalValue_Error(t *testing.T) {
	assert.Equal(t, "int_overflow", exceptionalValueIntOverflow.Error())
}
