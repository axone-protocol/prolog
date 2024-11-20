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
	var p *Promise
	ks := make([]func(context.Context) *Promise, len(cs))
	for i := range cs {
		i, c := i, cs[i]
		ks[i] = func(context.Context) *Promise {
			vars := make([]Variable, len(c.vars))
			for i := range vars {
				vars[i] = NewVariable()
			}
			return vm.exec(c.bytecode, vars, k, args, nil, env, p)
		}
	}
	p = Delay(ks...)
	return p
}

func compile(t Term, env *Env) (clauses, error) {
	t = env.Resolve(t)
	if t, ok := t.(Compound); ok && t.Functor() == atomIf && t.Arity() == 2 {
		var cs clauses
		head, body := t.Arg(0), t.Arg(1)
		iter := altIterator{Alt: body, Env: env}
		for iter.Next() {
			c, err := compileClause(head, iter.Current(), env)
			if err != nil {
				return nil, typeError(validTypeCallable, body, env)
			}
			c.raw = t
			cs = append(cs, c)
		}
		return cs, nil
	}

	c, err := compileClause(t, nil, env)
	c.raw = env.simplify(t)
	return []clause{c}, err
}

type clause struct {
	pi       procedureIndicator
	raw      Term
	vars     []Variable
	bytecode bytecode
}

func compileClause(head Term, body Term, env *Env) (clause, error) {
	var c clause
	var goals []Term

	head, goals = desugar(head, goals)
	body, goals = desugar(body, goals)

	if body != nil {
		goals = append(goals, body)
	}
	if len(goals) > 0 {
		body = seq(atomComma, goals...)
	}

	c.compileHead(head, env)

	if body != nil {
		if err := c.compileBody(body, env); err != nil {
			return c, typeError(validTypeCallable, body, env)
		}
	}

	c.emit(instruction{opcode: OpExit})
	return c, nil
}

func desugar(term Term, acc []Term) (Term, []Term) {
	switch t := term.(type) {
	case charList, codeList:
		return t, acc
	case list:
		l := make(list, len(t))
		for i, e := range t {
			l[i], acc = desugar(e, acc)
		}
		return l, acc
	case *partial:
		c, acc := desugar(t.Compound, acc)
		tail, acc := desugar(*t.tail, acc)
		return &partial{
			Compound: c.(Compound),
			tail:     &tail,
		}, acc
	case Compound:
		if t.Functor() == atomSpecialDot && t.Arity() == 2 {
			tempV := NewVariable()
			lhs, acc := desugar(t.Arg(0), acc)
			rhs, acc := desugar(t.Arg(1), acc)

			return tempV, append(acc, atomDot.Apply(lhs, rhs, tempV))
		}

		c := compound{
			functor: t.Functor(),
			args:    make([]Term, t.Arity()),
		}
		for i := 0; i < t.Arity(); i++ {
			c.args[i], acc = desugar(t.Arg(i), acc)
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
		return c.compilePred(atomCall.Apply(p), env)
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
		c.emit(instruction{opcode: OpGetVar, operand: c.varOffset(a)})
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
		c.emit(instruction{opcode: OpPutVar, operand: c.varOffset(a)})
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

func (c *clause) varOffset(o Variable) Integer {
	for i, v := range c.vars {
		if v == o {
			return Integer(i)
		}
	}
	c.vars = append(c.vars, o)
	return Integer(len(c.vars) - 1)
}
