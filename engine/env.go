package engine

func (vm *VM) varContext() Variable {
	if vm.variableContext == nil {
		*vm.variableContext = vm.NewVariable()
	}
	return *vm.variableContext
}

var rootContext = NewAtom("root")

type envKey int64

func newEnvKey(v Variable) envKey {
	// A new Variable is always bigger than the previous ones.
	// So, if we used the Variable itself as the key, insertions to the Env tree would be skewed to the right.
	k := envKey(v)
	if k/2 != 0 {
		k *= -1
	}
	return k
}

type color uint8

const (
	red color = iota
	black
)

// Env is a mapping from variables to terms.
type Env struct {
	// basically, this is Red-Black tree from Purely Functional Data Structures by Okazaki.
	color       color
	left, right *Env
	binding
}

type binding struct {
	key   envKey
	value Term
	// attributes?
}

func (vm *VM) rootEnv() *Env {
	return &Env{
		binding: binding{
			key:   newEnvKey(vm.varContext()),
			value: rootContext,
		},
	}
}

// NewEnv creates an empty environment.
func NewEnv() *Env {
	return nil
}

// lookup returns a term that the given variable is bound to.
func (e *Env) lookup(vm *VM, v Variable) (Term, bool) {
	k := newEnvKey(v)

	node := e
	if node == nil {
		node = vm.rootEnv()
	}
	for {
		if node == nil {
			return nil, false
		}
		switch {
		case k < node.key:
			node = node.left
		case k > node.key:
			node = node.right
		default:
			return node.value, true
		}
	}
}

// bind adds a new entry to the environment.
func (e *Env) bind(vm *VM, v Variable, t Term) *Env {
	k := newEnvKey(v)

	node := e
	if node == nil {
		node = vm.rootEnv()
	}
	ret := *node.insert(k, t)
	ret.color = black
	return &ret
}

func (e *Env) insert(k envKey, v Term) *Env {
	if e == nil {
		return &Env{color: red, binding: binding{key: k, value: v}}
	}
	switch {
	case k < e.key:
		ret := *e
		ret.left = e.left.insert(k, v)
		ret.balance()
		return &ret
	case k > e.key:
		ret := *e
		ret.right = e.right.insert(k, v)
		ret.balance()
		return &ret
	default:
		ret := *e
		ret.value = v
		return &ret
	}
}

func (e *Env) balance() {
	var (
		a, b, c, d *Env
		x, y, z    binding
	)
	switch {
	case e.left != nil && e.left.color == red:
		switch {
		case e.left.left != nil && e.left.left.color == red:
			a = e.left.left.left
			b = e.left.left.right
			c = e.left.right
			d = e.right
			x = e.left.left.binding
			y = e.left.binding
			z = e.binding
		case e.left.right != nil && e.left.right.color == red:
			a = e.left.left
			b = e.left.right.left
			c = e.left.right.right
			d = e.right
			x = e.left.binding
			y = e.left.right.binding
			z = e.binding
		default:
			return
		}
	case e.right != nil && e.right.color == red:
		switch {
		case e.right.left != nil && e.right.left.color == red:
			a = e.left
			b = e.right.left.left
			c = e.right.left.right
			d = e.right.right
			x = e.binding
			y = e.right.left.binding
			z = e.right.binding
		case e.right.right != nil && e.right.right.color == red:
			a = e.left
			b = e.right.left
			c = e.right.right.left
			d = e.right.right.right
			x = e.binding
			y = e.right.binding
			z = e.right.right.binding
		default:
			return
		}
	default:
		return
	}
	*e = Env{
		color:   red,
		left:    &Env{color: black, left: a, right: b, binding: x},
		right:   &Env{color: black, left: c, right: d, binding: z},
		binding: y,
	}
}

// Resolve follows the variable chain and returns the first non-variable term or the last free variable.
func (e *Env) Resolve(vm *VM, t Term) Term {
	var stop []Variable
	for t != nil {
		switch v := t.(type) {
		case Variable:
			for _, s := range stop {
				if v == s {
					return v
				}
			}
			ref, ok := e.lookup(vm, v)
			if !ok {
				return v
			}
			stop = append(stop, v)
			t = ref
		default:
			return v
		}
	}
	return nil
}

// simplify trys to remove as many variables as possible from term t.
func (e *Env) simplify(vm *VM, t Term) Term {
	return simplify(vm, t, nil, e)
}

func simplify(vm *VM, t Term, simplified map[termID]Compound, env *Env) Term {
	if simplified == nil {
		simplified = map[termID]Compound{}
	}
	t = env.Resolve(vm, t)
	if c, ok := simplified[id(t)]; ok {
		return c
	}
	switch t := t.(type) {
	case charList, codeList:
		return t
	case list:
		l := make(list, len(t))
		simplified[id(t)] = l
		for i, e := range t {
			l[i] = simplify(vm, e, simplified, env)
		}
		return l
	case *partial:
		var p partial
		simplified[id(t)] = &p
		p.Compound = simplify(vm, t.Compound, simplified, env).(Compound)
		tail := simplify(vm, *t.tail, simplified, env)
		p.tail = &tail
		return &p
	case Compound:
		c := compound{
			functor: t.Functor(),
			args:    make([]Term, t.Arity()),
		}
		simplified[id(t)] = &c
		for i := 0; i < t.Arity(); i++ {
			c.args[i] = simplify(vm, t.Arg(i), simplified, env)
		}
		return &c
	default:
		return t
	}
}

type variables []Variable

// freeVariables extracts variables in the given Term.
func (e *Env) freeVariables(vm *VM, t Term) []Variable {
	return e.appendFreeVariables(vm, nil, t)
}

func (e *Env) appendFreeVariables(vm *VM, fvs variables, t Term) variables {
	switch t := e.Resolve(vm, t).(type) {
	case Variable:
		for _, v := range fvs {
			if v == t {
				return fvs
			}
		}
		return append(fvs, t)
	case Compound:
		for i := 0; i < t.Arity(); i++ {
			fvs = e.appendFreeVariables(vm, fvs, t.Arg(i))
		}
	}
	return fvs
}

// Unify unifies 2 terms.
func (e *Env) Unify(vm *VM, x, y Term) (*Env, bool) {
	return e.unify(vm, x, y, false)
}

func (e *Env) unifyWithOccursCheck(vm *VM, x, y Term) (*Env, bool) {
	return e.unify(vm, x, y, true)
}

func (e *Env) unify(vm *VM, x, y Term, occursCheck bool) (*Env, bool) {
	x, y = e.Resolve(vm, x), e.Resolve(vm, y)
	switch x := x.(type) {
	case Variable:
		switch {
		case x == y:
			return e, true
		case occursCheck && contains(vm, y, x, e):
			return e, false
		default:
			return e.bind(vm, x, y), true
		}
	case Compound:
		switch y := y.(type) {
		case Variable:
			return e.unify(vm, y, x, occursCheck)
		case Compound:
			if x.Functor() != y.Functor() {
				return e, false
			}
			if x.Arity() != y.Arity() {
				return e, false
			}
			var ok bool
			for i := 0; i < x.Arity(); i++ {
				e, ok = e.unify(vm, x.Arg(i), y.Arg(i), occursCheck)
				if !ok {
					return e, false
				}
			}
			return e, true
		default:
			return e, false
		}
	default: // atomic
		switch y := y.(type) {
		case Variable:
			return e.unify(vm, y, x, occursCheck)
		case Float:
			if x, ok := x.(Float); ok {
				return e, y.Eq(x)
			}
			return e, false
		case Integer:
			if x, ok := x.(Integer); ok {
				return e, y == x
			}
			return e, false
		default:
			return e, x == y
		}
	}
}

func contains(vm *VM, t, s Term, env *Env) bool {
	switch t := t.(type) {
	case Variable:
		if t == s {
			return true
		}
		ref, ok := env.lookup(vm, t)
		if !ok {
			return false
		}
		return contains(vm, ref, s, env)
	case Compound:
		if s, ok := s.(Atom); ok && t.Functor() == s {
			return true
		}
		for i := 0; i < t.Arity(); i++ {
			if contains(vm, t.Arg(i), s, env) {
				return true
			}
		}
		return false
	default:
		return t == s
	}
}
