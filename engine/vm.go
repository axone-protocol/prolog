package engine

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"strings"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// HookFunc is a type for a hook function that is triggered before the VM executes a specific instruction.
// If the hook function returns an error, the VM halts execution and returns the error.
type HookFunc func(opcode Opcode, operand Term, env *Env) error

// DebugHookFn is a function that returns a hook function that prints the executed instruction.
func DebugHookFn(w io.Writer) HookFunc {
	return func(opcode Opcode, operand Term, _ *Env) error {
		_, _ = io.WriteString(w, opcode.String())

		if operand != nil {
			_, _ = io.WriteString(w, "(")
			_ = operand.WriteTerm(w, &defaultWriteOptions, nil)
			_, _ = io.WriteString(w, ")")
		}
		_, _ = io.WriteString(w, "\n")

		return nil
	}
}

// CompositeHook returns a hook function that chains multiple hooks together.
// The hooks are executed sequentially, and if any hook returns an error, the execution stops.
func CompositeHook(fs ...HookFunc) HookFunc {
	return func(opcode Opcode, operand Term, env *Env) error {
		for _, f := range fs {
			if err := f(opcode, operand, env); err != nil {
				return err
			}
		}
		return nil
	}
}

type bytecode []instruction

type instruction struct {
	opcode  Opcode
	operand Term
}

func (i instruction) String() string {
	var sb strings.Builder
	if i.operand != nil {
		_ = i.operand.WriteTerm(&sb, &defaultWriteOptions, nil)
	}
	return fmt.Sprintf("%s(%s)", i.opcode, sb.String())
}

type Opcode byte

const (
	OpEnter Opcode = iota
	OpCall
	OpExit
	OpGetConst
	OpPutConst
	OpGetVar
	OpPutVar
	OpGetFunctor
	OpPutFunctor
	OpPop

	OpCut
	OpGetList
	OpPutList
	OpGetPartial
	OpPutPartial
	OpGetDict
	OpPutDict
)

func (op Opcode) String() string {
	opcodeStrings := [...]string{
		OpEnter:      "enter",
		OpCall:       "call",
		OpExit:       "exit",
		OpGetConst:   "get_const",
		OpPutConst:   "put_const",
		OpGetVar:     "get_var",
		OpPutVar:     "put_var",
		OpGetFunctor: "get_functor",
		OpPutFunctor: "put_functor",
		OpPop:        "pop",
		OpCut:        "cut",
		OpGetList:    "get_list",
		OpPutList:    "put_list",
		OpGetPartial: "get_partial",
		OpPutPartial: "put_partial",
		OpGetDict:    "get_dict",
		OpPutDict:    "put_dict",
	}

	if int(op) < 0 || int(op) >= len(opcodeStrings) {
		return fmt.Sprintf("(%d)", op)
	}
	return opcodeStrings[op]
}

// Success is a continuation that leads to true.
func Success(*Env) *Promise {
	return Bool(true)
}

// Failure is a continuation that leads to false.
func Failure(*Env) *Promise {
	return Bool(false)
}

// VM is the core of a Prolog interpreter. The zero value for VM is a valid VM without any builtin predicates.
type VM struct {
	// Unknown is a callback that is triggered when the VM reaches to an unknown predicate while current_prolog_flag(unknown, warning).
	Unknown func(name Atom, args []Term, env *Env)

	procedures *orderedmap.OrderedMap[procedureIndicator, procedure]
	unknown    unknownAction

	// FS is a file system that is referenced when the VM loads Prolog texts e.g. ensure_loaded/1.
	// It has no effect on open/4 nor open/3 which always access the actual file system.
	FS     fs.FS
	loaded map[string]struct{}

	// Internal/external expression
	_operators      *operators
	charConversions map[rune]rune
	charConvEnabled bool
	doubleQuotes    doubleQuotes

	// I/O
	streams       streams
	input, output *Stream

	// Limits
	maxVariables uint64

	// Hook
	hook HookFunc

	// Misc
	debug bool
}

// Register0 registers a predicate of arity 0.
func (vm *VM) Register0(name Atom, p Predicate0) {
	vm.setProcedure(procedureIndicator{name: name, arity: 0}, p)
}

// Register1 registers a predicate of arity 1.
func (vm *VM) Register1(name Atom, p Predicate1) {
	vm.setProcedure(procedureIndicator{name: name, arity: 1}, p)
}

// Register2 registers a predicate of arity 2.
func (vm *VM) Register2(name Atom, p Predicate2) {
	vm.setProcedure(procedureIndicator{name: name, arity: 2}, p)
}

// Register3 registers a predicate of arity 3.
func (vm *VM) Register3(name Atom, p Predicate3) {
	vm.setProcedure(procedureIndicator{name: name, arity: 3}, p)
}

// Register4 registers a predicate of arity 4.
func (vm *VM) Register4(name Atom, p Predicate4) {
	vm.setProcedure(procedureIndicator{name: name, arity: 4}, p)
}

// Register5 registers a predicate of arity 5.
func (vm *VM) Register5(name Atom, p Predicate5) {
	vm.setProcedure(procedureIndicator{name: name, arity: 5}, p)
}

// Register6 registers a predicate of arity 6.
func (vm *VM) Register6(name Atom, p Predicate6) {
	vm.setProcedure(procedureIndicator{name: name, arity: 6}, p)
}

// Register7 registers a predicate of arity 7.
func (vm *VM) Register7(name Atom, p Predicate7) {
	vm.setProcedure(procedureIndicator{name: name, arity: 7}, p)
}

// Register8 registers a predicate of arity 8.
func (vm *VM) Register8(name Atom, p Predicate8) {
	vm.setProcedure(procedureIndicator{name: name, arity: 8}, p)
}

type unknownAction int

const (
	unknownError unknownAction = iota
	unknownFail
	unknownWarning
)

func (u unknownAction) String() string {
	return [...]string{
		unknownError:   "error",
		unknownFail:    "fail",
		unknownWarning: "warning",
	}[u]
}

type procedure interface {
	call(*VM, []Term, Cont, *Env) *Promise
}

// Cont is a continuation.
type Cont func(*Env) *Promise

// Arrive is the entry point of the VM.
func (vm *VM) Arrive(name Atom, args []Term, k Cont, env *Env) (promise *Promise) {
	defer ensurePromise(&promise)

	if vm.Unknown == nil {
		vm.Unknown = func(Atom, []Term, *Env) {}
	}

	pi := procedureIndicator{name: name, arity: Integer(len(args))}
	p, ok := vm.getProcedure(pi)
	if !ok {
		switch vm.unknown {
		case unknownWarning:
			vm.Unknown(name, args, env)
			fallthrough
		case unknownFail:
			return Bool(false)
		default:
			return Error(existenceError(objectTypeProcedure, pi.Term(), env))
		}
	}

	// bind the special variable to inform the predicate about the context.
	env = env.bind(varContext, pi.Term())

	return p.call(vm, args, k, env)
}

func (vm *VM) exec(pc bytecode, vars []Variable, cont Cont, args []Term, astack [][]Term, env *Env, cutParent *Promise) *Promise {
	var (
		ok  = true
		op  instruction
		arg Term
	)
	for ok {
		op, pc = pc[0], pc[1:]
		if vm.hook != nil {
			if err := vm.hook(op.opcode, op.operand, env); err != nil {
				return Error(err)
			}
		}

		switch opcode, operand := op.opcode, op.operand; opcode {
		case OpGetConst:
			arg, args = args[0], args[1:]
			env, ok = env.Unify(arg, operand)
		case OpPutConst:
			args = append(args, operand)
		case OpGetVar:
			v := vars[operand.(Integer)]
			arg, args = args[0], args[1:]
			env, ok = env.Unify(arg, v)
		case OpPutVar:
			v := vars[operand.(Integer)]
			args = append(args, v)
		case OpGetFunctor:
			pi := operand.(procedureIndicator)
			arg, astack = env.Resolve(args[0]), append(astack, args[1:])
			args = make([]Term, int(pi.arity))
			for i := range args {
				args[i] = NewVariable()
			}
			env, ok = env.Unify(arg, pi.name.Apply(args...))
		case OpPutFunctor:
			pi := operand.(procedureIndicator)
			vs := make([]Term, int(pi.arity))
			arg = pi.name.Apply(vs...)
			args = append(args, arg)
			astack = append(astack, args)
			args = vs[:0]
		case OpPop:
			args, astack = astack[len(astack)-1], astack[:len(astack)-1]
		case OpEnter:
			break
		case OpCall:
			pi := operand.(procedureIndicator)
			return vm.Arrive(pi.name, args, func(env *Env) *Promise {
				return vm.exec(pc, vars, cont, nil, nil, env, cutParent)
			}, env)
		case OpExit:
			return cont(env)
		case OpCut:
			return cut(cutParent, func(context.Context) *Promise {
				return vm.exec(pc, vars, cont, args, astack, env, cutParent)
			})
		case OpGetList:
			l := operand.(Integer)
			arg, astack = args[0], append(astack, args[1:])
			args = make([]Term, int(l))
			for i := range args {
				args[i] = NewVariable()
			}
			env, ok = env.Unify(arg, list(args))
		case OpPutList:
			l := operand.(Integer)
			vs := make([]Term, int(l))
			arg = list(vs)
			args = append(args, arg)
			astack = append(astack, args)
			args = vs[:0]
		case OpGetDict:
			l := operand.(Integer)
			arg, astack = args[0], append(astack, args[1:])
			args = make([]Term, int(l))
			for i := range args {
				args[i] = NewVariable()
			}
			env, ok = env.Unify(arg, newDict(args))
		case OpPutDict:
			l := operand.(Integer)
			vs := make([]Term, int(l))
			arg = &dict{compound: compound{functor: atomDict, args: vs}}
			args = append(args, arg)
			astack = append(astack, args)
			args = vs[:0]
		case OpGetPartial:
			l := operand.(Integer)
			arg, astack = args[0], append(astack, args[1:])
			args = make([]Term, int(l+1))
			for i := range args {
				args[i] = NewVariable()
			}
			env, ok = env.Unify(arg, PartialList(args[0], args[1:]...))
		case OpPutPartial:
			l := operand.(Integer)
			vs := make([]Term, int(l+1))
			arg = &partial{
				Compound: list(vs[1:]),
				tail:     &vs[0],
			}
			args = append(args, arg)
			astack = append(astack, args)
			args = vs[:0]
		}
	}

	return Bool(false)
}

// SetUserInput sets the given stream as user_input.
func (vm *VM) SetUserInput(s *Stream) {
	s.vm = vm
	s.alias = atomUserInput
	vm.streams.add(s)
	vm.input = s
}

// SetUserOutput sets the given stream as user_output.
func (vm *VM) SetUserOutput(s *Stream) {
	s.vm = vm
	s.alias = atomUserOutput
	vm.streams.add(s)
	vm.output = s
}

// SetMaxVariables sets the maximum number of variables that the VM can create.
// Zero value mean no limits
func (vm *VM) SetMaxVariables(n uint64) {
	vm.maxVariables = n
	maxVariables = n
}

// InstallHook sets the given hook function in the VM.
func (vm *VM) InstallHook(f HookFunc) {
	vm.hook = f
}

// ClearHook removes the installed hook function from the VM.
func (vm *VM) ClearHook() {
	vm.hook = nil
}

// ResetEnv is used to reset all global variable
func (vm *VM) ResetEnv() {
	varCounter.count = 0
	varContext = NewVariable()
	rootContext = NewAtom("root")
	rootEnv = &Env{
		binding: binding{
			key:   newEnvKey(varContext),
			value: rootContext,
		},
	}
	maxVariables = vm.maxVariables
}

func (vm *VM) getProcedure(p procedureIndicator) (procedure, bool) {
	if vm.procedures == nil {
		return nil, false
	}
	return vm.procedures.Get(p)
}

func (vm *VM) setProcedure(key procedureIndicator, val procedure) (procedure, bool) {
	if vm.procedures == nil {
		vm.procedures = orderedmap.New[procedureIndicator, procedure]()
	}
	return vm.procedures.Set(key, val)
}

func (vm *VM) getOperators() *operators {
	if vm._operators == nil {
		vm._operators = newOperators()
	}
	return vm._operators
}

// Predicate0 is a predicate of arity 0.
type Predicate0 func(*VM, Cont, *Env) *Promise

func (p Predicate0) call(vm *VM, args []Term, k Cont, env *Env) *Promise {
	if len(args) != 0 {
		return Error(&wrongNumberOfArgumentsError{expected: 0, actual: args})
	}

	return p(vm, k, env)
}

// Predicate1 is a predicate of arity 1.
type Predicate1 func(*VM, Term, Cont, *Env) *Promise

func (p Predicate1) call(vm *VM, args []Term, k Cont, env *Env) *Promise {
	if len(args) != 1 {
		return Error(&wrongNumberOfArgumentsError{expected: 1, actual: args})
	}

	return p(vm, args[0], k, env)
}

// Predicate2 is a predicate of arity 2.
type Predicate2 func(*VM, Term, Term, Cont, *Env) *Promise

func (p Predicate2) call(vm *VM, args []Term, k Cont, env *Env) *Promise {
	if len(args) != 2 {
		return Error(&wrongNumberOfArgumentsError{expected: 2, actual: args})
	}

	return p(vm, args[0], args[1], k, env)
}

// Predicate3 is a predicate of arity 3.
type Predicate3 func(*VM, Term, Term, Term, Cont, *Env) *Promise

func (p Predicate3) call(vm *VM, args []Term, k Cont, env *Env) *Promise {
	if len(args) != 3 {
		return Error(&wrongNumberOfArgumentsError{expected: 3, actual: args})
	}

	return p(vm, args[0], args[1], args[2], k, env)
}

// Predicate4 is a predicate of arity 4.
type Predicate4 func(*VM, Term, Term, Term, Term, Cont, *Env) *Promise

func (p Predicate4) call(vm *VM, args []Term, k Cont, env *Env) *Promise {
	if len(args) != 4 {
		return Error(&wrongNumberOfArgumentsError{expected: 4, actual: args})
	}

	return p(vm, args[0], args[1], args[2], args[3], k, env)
}

// Predicate5 is a predicate of arity 5.
type Predicate5 func(*VM, Term, Term, Term, Term, Term, Cont, *Env) *Promise

func (p Predicate5) call(vm *VM, args []Term, k Cont, env *Env) *Promise {
	if len(args) != 5 {
		return Error(&wrongNumberOfArgumentsError{expected: 5, actual: args})
	}

	return p(vm, args[0], args[1], args[2], args[3], args[4], k, env)
}

// Predicate6 is a predicate of arity 6.
type Predicate6 func(*VM, Term, Term, Term, Term, Term, Term, Cont, *Env) *Promise

func (p Predicate6) call(vm *VM, args []Term, k Cont, env *Env) *Promise {
	if len(args) != 6 {
		return Error(&wrongNumberOfArgumentsError{expected: 6, actual: args})
	}

	return p(vm, args[0], args[1], args[2], args[3], args[4], args[5], k, env)
}

// Predicate7 is a predicate of arity 7.
type Predicate7 func(*VM, Term, Term, Term, Term, Term, Term, Term, Cont, *Env) *Promise

func (p Predicate7) call(vm *VM, args []Term, k Cont, env *Env) *Promise {
	if len(args) != 7 {
		return Error(&wrongNumberOfArgumentsError{expected: 7, actual: args})
	}

	return p(vm, args[0], args[1], args[2], args[3], args[4], args[5], args[6], k, env)
}

// Predicate8 is a predicate of arity 8.
type Predicate8 func(*VM, Term, Term, Term, Term, Term, Term, Term, Term, Cont, *Env) *Promise

func (p Predicate8) call(vm *VM, args []Term, k Cont, env *Env) *Promise {
	if len(args) != 8 {
		return Error(&wrongNumberOfArgumentsError{expected: 8, actual: args})
	}

	return p(vm, args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], k, env)
}

// procedureIndicator identifies a procedure e.g. (=)/2.
type procedureIndicator struct {
	name  Atom
	arity Integer
}

func (p procedureIndicator) WriteTerm(w io.Writer, opts *WriteOptions, env *Env) error {
	return WriteCompound(w, p, opts, env)
}

func (p procedureIndicator) Compare(t Term, env *Env) int {
	return CompareCompound(p, t, env)
}

func (p procedureIndicator) Functor() Atom {
	return atomSlash
}

func (p procedureIndicator) Arity() int {
	return 2
}

func (p procedureIndicator) Arg(n int) Term {
	if n == 0 {
		return p.name
	}
	return p.arity
}

func (p procedureIndicator) String() string {
	var sb strings.Builder
	_ = p.name.WriteTerm(&sb, &WriteOptions{
		quoted: true,
	}, nil)
	_, _ = fmt.Fprintf(&sb, "/%d", p.arity)
	return sb.String()
}

// Term returns p as term.
func (p procedureIndicator) Term() Term {
	return atomSlash.Apply(p.name, p.arity)
}

// Apply applies p to args.
func (p procedureIndicator) Apply(args ...Term) (Term, error) {
	if p.arity != Integer(len(args)) {
		return nil, &wrongNumberOfArgumentsError{expected: int(p.arity), actual: args}
	}
	return p.name.Apply(args...), nil
}

func piArg(t Term, env *Env) (procedureIndicator, func(int) Term, error) {
	switch f := env.Resolve(t).(type) {
	case Variable:
		return procedureIndicator{}, nil, InstantiationError(env)
	case Atom:
		return procedureIndicator{name: f, arity: 0}, nil, nil
	case Compound:
		return procedureIndicator{name: f.Functor(), arity: Integer(f.Arity())}, f.Arg, nil
	default:
		return procedureIndicator{}, nil, typeError(validTypeCallable, f, env)
	}
}

type wrongNumberOfArgumentsError struct {
	expected int
	actual   []Term
}

func (e *wrongNumberOfArgumentsError) Error() string {
	return fmt.Sprintf("wrong number of arguments: expected=%d, actual=%s", e.expected, e.actual)
}
