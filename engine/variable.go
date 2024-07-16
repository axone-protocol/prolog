package engine

import (
	"fmt"
	"io"
	"sync/atomic"
)

func (vm *VM) lastVariable() Variable {
	return Variable(vm.varCounter)
}

// Variable is a prolog variable.
type Variable int64

// NewVariable creates a new anonymous variable.
func (vm *VM) NewVariable() Variable {
	n := atomic.AddInt64(&vm.varCounter, 1)
	return Variable(n)
}

func (v Variable) WriteTerm(vm *VM, w io.Writer, opts *WriteOptions, env *Env) error {
	x := env.Resolve(vm, v)
	v, ok := x.(Variable)
	if !ok {
		return x.WriteTerm(vm, w, opts, env)
	}

	ew := errWriter{w: w}

	if letterDigit(opts.left.name) {
		_, _ = ew.Write([]byte(" "))
	}
	if a, ok := opts.variableNames[v]; ok {
		_ = a.WriteTerm(vm, &ew, opts.withQuoted(false).withLeft(operator{}).withRight(operator{}), env)
	} else {
		_, _ = ew.Write([]byte(fmt.Sprintf("_%d", v)))
	}
	if letterDigit(opts.right.name) {
		_, _ = ew.Write([]byte(" "))
	}

	return ew.err
}

func (v Variable) Compare(vm *VM, t Term, env *Env) int {
	w := env.Resolve(vm, v)
	v, ok := w.(Variable)
	if !ok {
		return w.Compare(vm, t, env)
	}

	switch t := env.Resolve(vm, t).(type) {
	case Variable:
		switch {
		case v > t:
			return 1
		case v < t:
			return -1
		default:
			return 0
		}
	default:
		return -1
	}
}

// variableSet is a set of variables. The key is the variable and the value is the number of occurrences.
// So if you look at the value it's a multi set of variable occurrences and if you ignore the value it's a set of occurrences (witness).
type variableSet map[Variable]int

func newVariableSet(vm *VM, t Term, env *Env) variableSet {
	s := variableSet{}
	for terms := []Term{t}; len(terms) > 0; terms, t = terms[:len(terms)-1], terms[len(terms)-1] {
		switch t := env.Resolve(vm, t).(type) {
		case Variable:
			s[t] += 1
		case Compound:
			for i := 0; i < t.Arity(); i++ {
				terms = append(terms, t.Arg(i))
			}
		}
	}
	return s
}

func newExistentialVariablesSet(vm *VM, t Term, env *Env) variableSet {
	ev := variableSet{}
	for terms := []Term{t}; len(terms) > 0; terms, t = terms[:len(terms)-1], terms[len(terms)-1] {
		if c, ok := env.Resolve(vm, t).(Compound); ok && c.Functor() == atomCaret && c.Arity() == 2 {
			for v, o := range newVariableSet(vm, c.Arg(0), env) {
				ev[v] = o
			}
			terms = append(terms, c.Arg(1))
		}
	}
	return ev
}

func newFreeVariablesSet(vm *VM, t, v Term, env *Env) variableSet {
	fv := variableSet{}
	s := newVariableSet(vm, t, env)

	bv := newVariableSet(vm, v, env)
	for v := range newExistentialVariablesSet(vm, t, env) {
		bv[v] += 1
	}

	for v, n := range s {
		if m, ok := bv[v]; !ok {
			fv[v] = n + m
		}
	}

	return fv
}
