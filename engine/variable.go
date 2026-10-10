package engine

import (
	"errors"
	"fmt"
	"io"
	"math"
)

var ErrMaxVariables = errors.New("maximum number of variables reached")

// ErrVariableScope reports variables or environments from a different VM.
var ErrVariableScope = errors.New("variable belongs to a different VM")

// variableScope must not be zero-sized: distinct live VMs need distinct tokens.
type variableScope struct{ _ byte }

// Variable identifies a variable within an opaque VM scope.
// Only its local index participates in rendering and same-VM ordering.
type Variable struct {
	scope *variableScope
	index int64
}

func (vm *VM) scope() *variableScope {
	if vm.variableScope == nil {
		vm.variableScope = &variableScope{}
	}
	return vm.variableScope
}

// NewVariable creates an anonymous variable owned by vm.
// It panics with ErrMaxVariables if the VM's variable limit is reached.
func (vm *VM) NewVariable() Variable {
	if vm.variableCount == math.MaxInt64 || vm.maxVariables != 0 && vm.variableCount >= vm.maxVariables {
		panic(ErrMaxVariables)
	}
	vm.variableCount++
	return Variable{scope: vm.scope(), index: int64(vm.variableCount)}
}

func (v Variable) WriteTerm(w io.Writer, opts *WriteOptions, env *Env) error {
	x := env.Resolve(v)
	v, ok := x.(Variable)
	if !ok {
		return x.WriteTerm(w, opts, env)
	}

	ew := errWriter{w: w}

	if letterDigit(opts.left.name) {
		_, _ = ew.Write([]byte(" "))
	}
	if a, ok := opts.variableNames[v]; ok {
		_ = a.WriteTerm(&ew, opts.withQuoted(false).withLeft(operator{}).withRight(operator{}), env)
	} else {
		_, _ = fmt.Fprintf(&ew, "_%d", v.index)
	}
	if letterDigit(opts.right.name) {
		_, _ = ew.Write([]byte(" "))
	}

	return ew.err
}

func (v Variable) Compare(t Term, env *Env) int {
	env.charge(MeterCompareStep, 1)
	w := env.Resolve(v)
	v, ok := w.(Variable)
	if !ok {
		return w.Compare(t, env)
	}

	switch t := env.Resolve(t).(type) {
	case Variable:
		if v.scope != t.scope {
			panic(ErrVariableScope)
		}
		switch {
		case v.index > t.index:
			return 1
		case v.index < t.index:
			return -1
		default:
			return 0
		}
	default:
		return -1
	}
}

// checkTermScope validates a term without resolving it. The visited set
// makes validation safe for shared and cyclic compounds.
func checkTermScope(t Term, scope *variableScope, seen map[termID]struct{}) *variableScope {
	switch t := t.(type) {
	case Variable:
		if t.scope == nil || scope != nil && scope != t.scope {
			panic(ErrVariableScope)
		}
		return t.scope
	case *Stream:
		if t == nil || !t.ownedBy(t.vm) {
			panic(ErrStreamScope)
		}
		owner := t.vm.scope()
		if scope != nil && scope != owner {
			panic(ErrStreamScope)
		}
		return owner
	case Compound:
		if seen == nil {
			seen = make(map[termID]struct{})
		}
		key := id(t)
		if _, ok := seen[key]; ok {
			return scope
		}
		seen[key] = struct{}{}
		for i := range t.Arity() {
			scope = checkTermScope(t.Arg(i), scope, seen)
		}
	}
	return scope
}

// variableSet is a set of variables. The key is the variable and the value is the number of occurrences.
// So if you look at the value it's a multi set of variable occurrences and if you ignore the value it's a set of occurrences (witness).
type variableSet map[Variable]int

func newVariableSet(t Term, env *Env) variableSet {
	s := variableSet{}
	for terms := []Term{t}; len(terms) > 0; terms, t = terms[:len(terms)-1], terms[len(terms)-1] {
		switch t := env.Resolve(t).(type) {
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

func newExistentialVariablesSet(t Term, env *Env) variableSet {
	ev := variableSet{}
	for terms := []Term{t}; len(terms) > 0; terms, t = terms[:len(terms)-1], terms[len(terms)-1] {
		if c, ok := env.Resolve(t).(Compound); ok && c.Functor() == atomCaret && c.Arity() == 2 {
			for v, o := range newVariableSet(c.Arg(0), env) {
				ev[v] = o
			}
			terms = append(terms, c.Arg(1))
		}
	}
	return ev
}

func newFreeVariablesSet(t, v Term, env *Env) variableSet {
	fv := variableSet{}
	s := newVariableSet(t, env)

	bv := newVariableSet(v, env)
	for v := range newExistentialVariablesSet(t, env) {
		bv[v] += 1
	}

	for v, n := range s {
		if m, ok := bv[v]; !ok {
			fv[v] = n + m
		}
	}

	return fv
}
