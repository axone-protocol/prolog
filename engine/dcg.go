package engine

import (
	"context"
	"errors"
)

// based on: https://www.complang.tuwien.ac.at/ulrich/iso-prolog/dcgs/dcgsdin150408.pdf

// Phrase succeeds if the difference list of s0-s satisfies the grammar rule of grBody.
func Phrase(vm *VM, grBody, s0, s Term, k Cont, env *Env) (promise *Promise) {
	defer ensurePromise(&promise)

	env = vm.ownedEnv(env)
	goal, err := dcgBody(vm, grBody, s0, s, env)
	if err != nil {
		return Error(err)
	}
	return Delay(func(context.Context) *Promise {
		return Call(vm, goal, k, env)
	})
}

var errDCGNotApplicable = errors.New("not applicable")

// dcgMeterError lets expand retain its ordinary DCG fallback while propagating
// a host meter rejection.
type dcgMeterError struct {
	Exception
}

func expandDCG(vm *VM, term Term, env *Env) (Term, error) {
	env = vm.ownedEnv(env)
	var (
		expanded Term
		err      error
		meterErr error
	)
	func() {
		defer recoverMeterError(&meterErr)
		expanded, err = expandDCGBody(vm, term, env)
	}()
	if meterErr != nil {
		return nil, dcgMeterError{Exception: meterErr.(Exception)}
	}
	return expanded, err
}

func expandDCGBody(vm *VM, term Term, env *Env) (Term, error) {
	rule, ok := env.Resolve(term).(Compound)
	if !ok || rule.Functor() != atomArrow || rule.Arity() != 2 {
		return nil, errDCGNotApplicable
	}

	s0, s1, s := vm.NewVariable(), vm.NewVariable(), vm.NewVariable()
	if c, ok := env.Resolve(rule.Arg(0)).(Compound); ok && c.Functor() == atomComma && c.Arity() == 2 {
		head, err := dcgNonTerminal(c.Arg(0), s0, s, env)
		if err != nil {
			return nil, err
		}
		goal1, err := dcgBody(vm, rule.Arg(1), s0, s1, env)
		if err != nil {
			return nil, err
		}
		goal2, err := dcgTerminals(c.Arg(1), s, s1, env)
		if err != nil {
			return nil, err
		}
		chargeTermCells(2, env)
		body := atomComma.Apply(goal1, goal2)
		chargeTermCells(2, env)
		return atomIf.Apply(head, body), nil
	}

	head, err := dcgNonTerminal(rule.Arg(0), s0, s, env)
	if err != nil {
		return nil, err
	}
	body, err := dcgBody(vm, rule.Arg(1), s0, s, env)
	if err != nil {
		return nil, err
	}
	chargeTermCells(2, env)
	return atomIf.Apply(head, body), nil
}

func dcgNonTerminal(nonTerminal, list, rest Term, env *Env) (Term, error) {
	pi, arg, err := piArg(nonTerminal, env)
	if err != nil {
		return nil, err
	}
	args := makeTerms(addTermCells(int64(pi.arity), 2, env), env)
	for i := range args[:len(args)-2] {
		args[i] = arg(i)
	}
	args[len(args)-2], args[len(args)-1] = list, rest
	return pi.name.Apply(args...), nil
}

func dcgTerminals(terminals, list, rest Term, env *Env) (Term, error) {
	var elems []Term
	iter := ListIterator{List: terminals, Env: env}
	for iter.Next() {
		elems = appendTerms(elems, env, iter.Current())
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}
	chargeTermCells(3, env)
	return atomEqual.Apply(list, PartialList(rest, elems...)), nil
}

var dcgConstr map[procedureIndicator]func(vm *VM, args []Term, list, rest Term, env *Env) (Term, error)

func init() {
	dcgConstr = map[procedureIndicator]func(vm *VM, args []Term, list, rest Term, env *Env) (Term, error){
		{name: atomEmptyList, arity: 0}: func(_ *VM, _ []Term, list, rest Term, env *Env) (Term, error) {
			chargeTermCells(2, env)
			return atomEqual.Apply(list, rest), nil
		},
		{name: atomDot, arity: 2}: func(_ *VM, args []Term, list, rest Term, env *Env) (Term, error) {
			chargeTermCells(int64(len(args)), env)
			return dcgTerminals(atomDot.Apply(args...), list, rest, env)
		},
		{name: atomComma, arity: 2}: func(vm *VM, args []Term, list, rest Term, env *Env) (Term, error) {
			v := vm.NewVariable()
			first, err := dcgBody(vm, args[0], list, v, env)
			if err != nil {
				return nil, err
			}
			second, err := dcgBody(vm, args[1], v, rest, env)
			if err != nil {
				return nil, err
			}
			chargeTermCells(2, env)
			return atomComma.Apply(first, second), nil
		},
		{name: atomSemiColon, arity: 2}: func(vm *VM, args []Term, list, rest Term, env *Env) (Term, error) {
			body := dcgBody
			if t, ok := env.Resolve(args[0]).(Compound); ok && t.Functor() == atomThen && t.Arity() == 2 {
				body = dcgCBody
			}
			either, err := body(vm, args[0], list, rest, env)
			if err != nil {
				return nil, err
			}
			or, err := dcgBody(vm, args[1], list, rest, env)
			if err != nil {
				return nil, err
			}
			chargeTermCells(2, env)
			return atomSemiColon.Apply(either, or), nil
		},
		{name: atomBar, arity: 2}: func(vm *VM, args []Term, list, rest Term, env *Env) (Term, error) {
			either, err := dcgBody(vm, args[0], list, rest, env)
			if err != nil {
				return nil, err
			}
			or, err := dcgBody(vm, args[1], list, rest, env)
			if err != nil {
				return nil, err
			}
			chargeTermCells(2, env)
			return atomSemiColon.Apply(either, or), nil
		},
		{name: atomEmptyBlock, arity: 1}: func(_ *VM, args []Term, list, rest Term, env *Env) (Term, error) {
			chargeTermCells(2, env)
			equal := atomEqual.Apply(list, rest)
			chargeTermCells(2, env)
			return atomComma.Apply(args[0], equal), nil
		},
		{name: atomCall, arity: 1}: func(_ *VM, args []Term, list, rest Term, env *Env) (Term, error) {
			chargeTermCells(3, env)
			return atomCall.Apply(args[0], list, rest), nil
		},
		{name: atomPhrase, arity: 1}: func(_ *VM, args []Term, list, rest Term, env *Env) (Term, error) {
			chargeTermCells(3, env)
			return atomPhrase.Apply(args[0], list, rest), nil
		},
		{name: atomCut, arity: 0}: func(_ *VM, _ []Term, list, rest Term, env *Env) (Term, error) {
			chargeTermCells(2, env)
			equal := atomEqual.Apply(list, rest)
			chargeTermCells(2, env)
			return atomComma.Apply(atomCut, equal), nil
		},
		{name: atomNegation, arity: 1}: func(vm *VM, args []Term, list, rest Term, env *Env) (Term, error) {
			v := vm.NewVariable()
			g, err := dcgBody(vm, args[0], list, v, env)
			if err != nil {
				return nil, err
			}
			chargeTermCells(1, env)
			negated := atomNegation.Apply(g)
			chargeTermCells(2, env)
			equal := atomEqual.Apply(list, rest)
			chargeTermCells(2, env)
			return atomComma.Apply(negated, equal), nil
		},
		{name: atomThen, arity: 2}: func(vm *VM, args []Term, list, rest Term, env *Env) (Term, error) {
			v := vm.NewVariable()
			cond, err := dcgBody(vm, args[0], list, v, env)
			if err != nil {
				return nil, err
			}
			then, err := dcgBody(vm, args[1], v, rest, env)
			if err != nil {
				return nil, err
			}
			chargeTermCells(2, env)
			return atomThen.Apply(cond, then), nil
		},
	}
}

func dcgBody(vm *VM, term, list, rest Term, env *Env) (Term, error) {
	term = env.Resolve(term)
	if t, ok := term.(Variable); ok {
		chargeTermCells(3, env)
		return atomPhrase.Apply(t, list, rest), nil
	}

	t, err := dcgCBody(vm, term, list, rest, env)
	if errors.Is(err, errDCGNotApplicable) {
		return dcgNonTerminal(term, list, rest, env)
	}
	return t, err
}

func dcgCBody(vm *VM, term, list, rest Term, env *Env) (Term, error) {
	pi, arg, err := piArg(term, env)
	if err != nil {
		return nil, err
	}
	if c, ok := dcgConstr[pi]; ok {
		var args [2]Term
		for i := range args[:int(pi.arity)] {
			args[i] = arg(i)
		}
		return c(vm, args[:int(pi.arity)], list, rest, env)
	}
	return nil, errDCGNotApplicable
}
