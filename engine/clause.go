package engine

import (
	"context"
	"errors"
)

type userDefined struct {
	public        bool
	dynamic       bool
	multifile     bool
	discontiguous bool

	// 7.4.3 says "If no clauses are defined for a procedure indicated by a directive ... then the procedure shall exist but have no clauses."
	clauses
}

type clauses []clause

func (cs clauses) call(vm *VM, args []Term, k Cont, env *Env) *Promise {
	env = vm.ownedEnv(env)
	var p *Promise
	ks := make([]func(context.Context) *Promise, len(cs))
	for i := range cs {
		i, c := i, cs[i]
		ks[i] = func(context.Context) *Promise {
			chargeTermCells(int64(len(c.vars)), env)
			vars := make([]Variable, len(c.vars))
			for i := range vars {
				vars[i] = vm.NewVariable()
			}
			return vm.exec(c.bytecode, vars, k, args, nil, env, p)
		}
	}
	p = Delay(ks...)
	return p
}

func compile(vm *VM, t Term, env *Env) (clauses, error) {
	env = vm.ownedEnv(env)
	t = env.Resolve(t)
	if t, ok := t.(Compound); ok && t.Functor() == atomIf && t.Arity() == 2 {
		var cs clauses
		head, body := t.Arg(0), t.Arg(1)
		iter := altIterator{Alt: body, Env: env}
		for iter.Next() {
			c, err := compileClause(vm, head, iter.Current(), env)
			if err != nil {
				return nil, typeError(validTypeCallable, body, env)
			}
			c.raw = t
			cs = append(cs, c)
		}
		return cs, nil
	}

	c, err := compileClause(vm, t, nil, env)
	c.raw = env.simplify(t)
	return []clause{c}, err
}

type clause struct {
	pi       procedureIndicator
	raw      Term
	vars     []Variable
	bytecode bytecode
}

func compileClause(vm *VM, head Term, body Term, env *Env) (clause, error) {
	head, preds := desugarHead(vm, head, env)
	body = desugarBody(vm, body, env)

	if len(preds) > 0 {
		predSeq := meteredSeq(atomComma, preds, env)
		if body == nil {
			body = predSeq
		} else {
			args := makeTerms(2, env)
			args[0], args[1] = body, predSeq
			body = atomComma.Apply(args...)
		}
	}

	var c clause
	c.compileHead(head, env)

	if body != nil {
		if err := c.compileBody(body, env); err != nil {
			return c, typeError(validTypeCallable, body, env)
		}
	}

	c.emit(instruction{opcode: OpExit})

	return c, nil
}

func desugarHead(vm *VM, head Term, env *Env) (Term, []Term) {
	if head, ok := env.Resolve(head).(Compound); ok {
		return desugarPred(vm, head, nil, env)
	}
	return head, nil
}

func desugarBody(vm *VM, body Term, env *Env) Term {
	if body == nil {
		return body
	}

	var items []Term
	iter := seqIterator{Seq: body, Env: env}
	for iter.Next() {
		t, preds := desugarPred(vm, iter.Current(), nil, env)
		if len(preds) > 0 {
			items = appendTerms(items, env, preds...)
		}
		items = appendTerms(items, env, t)
	}

	return meteredSeq(atomComma, items, env)
}

func meteredSeq(sep Atom, ts []Term, env *Env) Term {
	s, ts := ts[len(ts)-1], ts[:len(ts)-1]
	for i := len(ts) - 1; i >= 0; i-- {
		args := makeTerms(2, env)
		args[0], args[1] = ts[i], s
		s = sep.Apply(args...)
	}
	return s
}

func desugarPred(vm *VM, term Term, acc []Term, env *Env) (Term, []Term) {
	switch t := env.Resolve(term).(type) {
	case charList, codeList:
		return t, acc
	case list:
		l := list(makeTerms(int64(len(t)), env))
		for i, e := range t {
			l[i], acc = desugarPred(vm, e, acc, env)
		}
		return l, acc
	case *partial:
		chargeTermCells(1, env)
		c, acc := desugarPred(vm, t.Compound, acc, env)
		tail, acc := desugarPred(vm, *t.tail, acc, env)
		return &partial{
			Compound: c.(Compound),
			tail:     &tail,
		}, acc
	case Compound:
		if t.Functor() == atomSpecialDot && t.Arity() == 2 {
			tempV := vm.NewVariable()
			lhs, acc := desugarPred(vm, t.Arg(0), acc, env)
			rhs, acc := desugarPred(vm, t.Arg(1), acc, env)
			args := makeTerms(3, env)
			args[0], args[1], args[2] = lhs, rhs, tempV

			return tempV, appendTerms(acc, env, atomDot.Apply(args...))
		}

		c := compound{
			functor: t.Functor(),
			args:    makeTerms(int64(t.Arity()), env),
		}
		for i := 0; i < t.Arity(); i++ {
			c.args[i], acc = desugarPred(vm, t.Arg(i), acc, env)
		}

		if _, ok := t.(Dict); ok {
			return &dict{c}, acc
		}

		return &c, acc
	default:
		return t, acc
	}
}

func (c *clause) emit(i instruction) {
	c.bytecode = append(c.bytecode, i)
}

func (c *clause) compileHead(head Term, env *Env) {
	switch head := env.Resolve(head).(type) {
	case Atom:
		c.pi = procedureIndicator{name: head, arity: 0}
	case Compound:
		c.pi = procedureIndicator{name: head.Functor(), arity: Integer(head.Arity())}
		for i := 0; i < head.Arity(); i++ {
			c.compileHeadArg(head.Arg(i), env)
		}
	}
}

func (c *clause) compileBody(body Term, env *Env) error {
	c.emit(instruction{opcode: OpEnter})
	iter := seqIterator{Seq: body, Env: env}
	for iter.Next() {
		if err := c.compilePred(iter.Current(), env); err != nil {
			return err
		}
	}
	return nil
}

var errNotCallable = errors.New("not callable")

func (c *clause) compilePred(p Term, env *Env) error {
	switch p := env.Resolve(p).(type) {
	case Variable:
		args := makeTerms(1, env)
		args[0] = p
		return c.compilePred(atomCall.Apply(args...), env)
	case Atom:
		switch p {
		case atomCut:
			c.emit(instruction{opcode: OpCut})
			return nil
		}
		c.emit(instruction{opcode: OpCall, operand: procedureIndicator{name: p, arity: 0}})
		return nil
	case Compound:
		for i := 0; i < p.Arity(); i++ {
			c.compileBodyArg(p.Arg(i), env)
		}
		c.emit(instruction{opcode: OpCall, operand: procedureIndicator{name: p.Functor(), arity: Integer(p.Arity())}})
		return nil
	default:
		return errNotCallable
	}
}

func (c *clause) compileHeadArg(a Term, env *Env) {
	switch a := env.Resolve(a).(type) {
	case Variable:
		c.emit(instruction{opcode: OpGetVar, operand: c.varOffset(a, env)})
	case charList, codeList: // Treat them as if they're atomic.
		c.emit(instruction{opcode: OpGetConst, operand: a})
	case list:
		c.emit(instruction{opcode: OpGetList, operand: Integer(len(a))})
		for _, arg := range a {
			c.compileHeadArg(arg, env)
		}
		c.emit(instruction{opcode: OpPop})
	case *partial:
		prefix := a.Compound.(list)
		c.emit(instruction{opcode: OpGetPartial, operand: Integer(len(prefix))})
		c.compileHeadArg(*a.tail, env)
		for _, arg := range prefix {
			c.compileHeadArg(arg, env)
		}
		c.emit(instruction{opcode: OpPop})
	case Compound:
		switch a.(type) {
		case Dict:
			c.emit(instruction{opcode: OpGetDict, operand: Integer(a.Arity())})
		default:
			c.emit(instruction{opcode: OpGetFunctor, operand: procedureIndicator{name: a.Functor(), arity: Integer(a.Arity())}})
		}

		for i := 0; i < a.Arity(); i++ {
			c.compileHeadArg(a.Arg(i), env)
		}
		c.emit(instruction{opcode: OpPop})
	default:
		c.emit(instruction{opcode: OpGetConst, operand: a})
	}
}

func (c *clause) compileBodyArg(a Term, env *Env) {
	switch a := env.Resolve(a).(type) {
	case Variable:
		c.emit(instruction{opcode: OpPutVar, operand: c.varOffset(a, env)})
	case charList, codeList: // Treat them as if they're atomic.
		c.emit(instruction{opcode: OpPutConst, operand: a})
	case list:
		c.emit(instruction{opcode: OpPutList, operand: Integer(len(a))})
		for _, arg := range a {
			c.compileBodyArg(arg, env)
		}
		c.emit(instruction{opcode: OpPop})
	case Dict:
		c.emit(instruction{opcode: OpPutDict, operand: Integer(a.Arity())})
		for i := 0; i < a.Arity(); i++ {
			c.compileBodyArg(a.Arg(i), env)
		}
		c.emit(instruction{opcode: OpPop})
	case *partial:
		var l int
		iter := ListIterator{List: a.Compound}
		for iter.Next() {
			l++
		}
		c.emit(instruction{opcode: OpPutPartial, operand: Integer(l)})
		c.compileBodyArg(*a.tail, env)
		iter = ListIterator{List: a.Compound}
		for iter.Next() {
			c.compileBodyArg(iter.Current(), env)
		}
		c.emit(instruction{opcode: OpPop})
	case Compound:
		switch a.(type) {
		case Dict:
			c.emit(instruction{opcode: OpPutDict, operand: Integer(a.Arity())})
		default:
			c.emit(instruction{opcode: OpPutFunctor, operand: procedureIndicator{name: a.Functor(), arity: Integer(a.Arity())}})
		}
		for i := 0; i < a.Arity(); i++ {
			c.compileBodyArg(a.Arg(i), env)
		}
		c.emit(instruction{opcode: OpPop})
	default:
		c.emit(instruction{opcode: OpPutConst, operand: a})
	}
}

func (c *clause) varOffset(o Variable, env *Env) Integer {
	for i, v := range c.vars {
		if v == o {
			return Integer(i)
		}
	}
	addTermCells(int64(len(c.vars)), 1, env)
	chargeTermCells(1, env)
	c.vars = append(c.vars, o)
	return Integer(len(c.vars) - 1)
}
