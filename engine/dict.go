package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"sort"
)

var (
	errInvalidDict = errors.New("invalid dict")
	errKeyExpected = errors.New("key expected")
)

var atomColon = NewAtom(":")

// dictFunction is the type for (predefined) functions that can be called on a Dict.
type dictFunction func(*VM, []Term, Term, Term, Cont, *Env) *Promise

// predefinedFuncs are the predefined (reserved) functions that can be called on a Dict.
var predefinedFuncs = map[Atom]map[int]dictFunction{
	"get": {
		1: func(vm *VM, args []Term, dict Term, result Term, cont Cont, env *Env) *Promise {
			return GetDict3(vm, args[0], dict, result, cont, env)
		},
		2: func(vm *VM, args []Term, dict Term, result Term, cont Cont, env *Env) *Promise {
			return GetDict4(vm, args[0], args[1], dict, result, cont, env)
		},
	},
	"put": {
		1: func(vm *VM, args []Term, dict Term, result Term, cont Cont, env *Env) *Promise {
			return PutDict3(vm, args[0], dict, result, cont, env)
		},
	},
	// TODO: to continue (https://www.swi-prolog.org/pldoc/man?section=ext-dicts-predefined)
}

// Dict is a term that represents a dictionary.
//
// Dicts are currently represented as a compound term using the functor `dict`.
// The first argument is the tag. The remaining arguments create an array of sorted key-value pairs.
type Dict interface {
	Compound

	// Tag returns the tag of the dictionary.
	Tag() Term
	// All returns an iterator over all key-value pairs in the dictionary.
	All() iter.Seq2[Atom, Term]

	// Value returns the value associated with the given key and a boolean indicating if the key exists.
	Value(key Atom) (Term, bool)
	// At returns the key and value at the specified index and a boolean indicating if the index is valid.
	At(i int) (Atom, Term, bool)
	// Len returns the number of key-value pairs in the dictionary.
	Len() int
}

type dict struct {
	compound
}

// NewDict creates a new dictionary (Dict) from the provided arguments (args).
// It processes the arguments and returns a Dict instance or an error if the
// arguments are invalid.
//
// The first argument is the tag. The remaining arguments are the key and value pairs.
func NewDict(args []Term) (Dict, error) {
	return newDictWithEnv(args, nil)
}

func newDictWithEnv(args []Term, env *Env) (d Dict, err error) {
	defer recoverMeterError(&err)

	processed, err := processArgs(args, env)
	if err != nil {
		return nil, err
	}
	return newDict(processed), nil
}

func newDict(args []Term) Dict {
	return &dict{
		compound: compound{
			functor: atomDict,
			args:    args,
		},
	}
}

func processArgs(args []Term, env *Env) ([]Term, error) {
	if len(args) == 0 || len(args)%2 == 0 {
		return nil, errInvalidDict
	}

	tag := args[0]
	rest := args[1:]

	kv := make(map[Atom]Term, len(rest)/2)
	for i := 0; i < len(rest); i += 2 {
		key, ok := rest[i].(Atom)
		value := rest[i+1]
		if !ok {
			return nil, errKeyExpected
		}

		if _, exists := kv[key]; exists {
			return nil, duplicateKeyError{key: key}
		}

		kv[key] = value
	}
	keys := make([]Atom, 0, len(kv))
	for k := range kv {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	pairs := int64(len(kv))
	processedArgs := makeTerms(addTermCells(addTermCells(pairs, pairs, env), 1, env), env)
	processedArgs[0] = tag
	for i, key := range keys {
		offset := 1 + 2*i
		processedArgs[offset], processedArgs[offset+1] = key, kv[key]
	}

	return processedArgs, nil
}

// WriteTerm outputs the Stream to an io.Writer.
func (d *dict) WriteTerm(w io.Writer, opts *WriteOptions, env *Env) error {
	err := d.Tag().WriteTerm(w, opts, env)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte("{"))
	if err != nil {
		return err
	}

	for i := 1; i < d.Arity(); i = i + 2 {
		if i > 1 {
			if _, err = w.Write([]byte(",")); err != nil {
				return err
			}
		}
		if err := d.Arg(i).WriteTerm(w, opts, env); err != nil {
			return err
		}
		if _, err = w.Write([]byte(":")); err != nil {
			return err
		}
		if err := d.Arg(i+1).WriteTerm(w, opts, env); err != nil {
			return err
		}
	}

	_, err = w.Write([]byte("}"))
	if err != nil {
		return err
	}
	return nil
}

// Compare compares the Stream with a Term.
func (d *dict) Compare(t Term, env *Env) int {
	return d.compound.Compare(t, env)
}

func (d *dict) Arg(n int) Term {
	return d.compound.Arg(n)
}

func (d *dict) Arity() int {
	return d.compound.Arity()
}

func (d *dict) Functor() Atom {
	return d.compound.Functor()
}

func (d *dict) Tag() Term {
	return d.compound.Arg(0)
}

func (d *dict) Len() int {
	return (d.Arity() - 1) / 2
}

func (d *dict) Value(key Atom) (Term, bool) {
	n := (d.Arity() - 1) / 2
	lo, hi := 0, n-1

	for lo <= hi {
		mid := (lo + hi) / 2
		i := 1 + 2*mid
		k := d.Arg(i).(Atom)
		if k == key {
			return d.Arg(i + 1), true
		}
		if k < key {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return nil, false
}

func (d *dict) At(i int) (Atom, Term, bool) {
	if i < 0 || i >= d.Len() {
		return "", nil, false
	}
	pos := 1 + 2*i
	return d.Arg(pos).(Atom), d.Arg(pos + 1), true
}

func (d *dict) All() iter.Seq2[Atom, Term] {
	return func(yield func(k Atom, v Term) bool) {
		for i := 0; i < d.Len(); i++ {
			k, v, _ := d.At(i)
			cont := yield(k, v)
			if !cont {
				return
			}
		}
	}
}

// Op3 primarily evaluates "./2" terms within Dict expressions.
// If the provided Function is an atom, the function checks for the corresponding key in the Dict,
// raising an exception if the key is missing.
// For compound terms, it interprets Function as a call to a predefined set of functions, processing it accordingly.
func Op3(vm *VM, dict, function, result Term, cont Cont, env *Env) (promise *Promise) {
	defer ensurePromise(&promise)

	env = vm.ownedEnv(env)
	switch dict := env.Resolve(dict).(type) {
	case Variable:
		return Error(InstantiationError(env))
	case Dict:
		switch function := env.Resolve(function).(type) {
		case Variable:
			return GetDict3(vm, function, dict, result, cont, env)
		case Atom:
			extracted, ok := dict.Value(function)
			if !ok {
				return Error(domainError(validDomainDictKey, function, env))
			}
			return Unify(vm, result, extracted, cont, env)
		case Compound:
			if funcs, ok := predefinedFuncs[function.Functor()]; ok {
				arity := function.Arity()
				if f, ok := funcs[arity]; ok {
					args := makeTerms(int64(arity), env)
					for i := range args {
						args[i] = function.Arg(i)
					}
					return f(vm, args, dict, result, cont, env)
				}
				return Error(existenceError(objectTypeProcedure, function, env))
			}
			return Error(existenceError(objectTypeProcedure, function, env))
		default:
			return Error(typeError(validTypeCallable, function, env))
		}
	default:
		return Error(typeError(validTypeDict, dict, env))
	}
}

// GetDict3 return the value associated with keyPath.
// keyPath is either a single key or a term Key1/Key2/.... Each key is either an atom, small integer or a variable.
// While Dict.Key (see Op3) throws an existence error, this function fails silently if a key does not exist in the
// target dict.
func GetDict3(vm *VM, keyPath Term, dict Term, result Term, cont Cont, env *Env) *Promise {
	switch dict := env.Resolve(dict).(type) {
	case Variable:
		return Error(InstantiationError(env))
	case Dict:
		switch keyPath := env.Resolve(keyPath).(type) {
		case Variable:
			promises := make([]PromiseFunc, 0, dict.Len())
			for key := range dict.All() {
				key := key
				promises = append(promises, func(context.Context) *Promise {
					value, _ := dict.Value(key)
					return Unify(vm, tuple(keyPath, result), tuple(key, value), cont, env)
				})
			}

			return Delay(promises...)
		case Atom:
			if value, ok := dict.Value(keyPath); ok {
				return Unify(vm, result, value, cont, env)
			}
			return Bool(false)
		case Compound:
			switch keyPath.Functor() {
			case atomSlash:
				if keyPath.Arity() == 2 {
					tempA := vm.NewVariable()
					return GetDict3(vm, keyPath.Arg(0), dict, tempA, func(env *Env) *Promise {
						tempB := vm.NewVariable()
						return GetDict3(vm, keyPath.Arg(1), tempA, tempB, func(env *Env) *Promise {
							return Unify(vm, tempB, result, cont, env)
						}, env)
					}, env)
				}
			}
			return Error(domainError(validDomainDictKey, keyPath, env))
		default:
			return Error(domainError(validDomainDictKey, keyPath, env))
		}
	default:
		return Error(typeError(validTypeDict, dict, env))
	}
}

// GetDict4 resolves a keyPath within dict like GetDict3, but unifies result with defaultValue
// when no matches are found. If keyPath includes variables, all possible bindings are produced before
// falling back to defaultValue.
func GetDict4(vm *VM, keyPath Term, defaultValue Term, dict Term, result Term, cont Cont, env *Env) *Promise {
	defaultValue = env.Resolve(defaultValue)
	if _, ok := defaultValue.(Variable); ok {
		return Error(InstantiationError(env))
	}

	found := false
	return Delay(
		func(ctx context.Context) *Promise {
			return GetDict3(vm, keyPath, dict, result, func(env *Env) *Promise {
				found = true
				return cont(env)
			}, env)
		},
		func(context.Context) *Promise {
			if found {
				return Bool(false)
			}
			return Unify(vm, result, defaultValue, cont, env)
		},
	)
}

// PutDict3 evaluates to a new dict where the key-values in dictIn replace or extend the key-values in the original dict.
//
// new is either a dict or list of attribute-value pairs using the syntax Key:Value, Key=Value, Key-Value or Key(Value)
func PutDict3(vm *VM, new Term, dictIn Term, dictOut Term, cont Cont, env *Env) (promise *Promise) {
	defer ensurePromise(&promise)

	env = vm.ownedEnv(env)
	switch dictIn := env.Resolve(dictIn).(type) {
	case Variable:
		return Error(InstantiationError(env))
	case Dict:
		switch new := env.Resolve(new).(type) {
		case Variable:
			return Error(InstantiationError(env))
		case Dict:
			dictIn = mergeDict(new, dictIn, env)
			return Unify(vm, dictOut, dictIn, cont, env)
		case Compound:
			dict, err := newDictFromListOfPairs(vm, new, env)
			if err != nil {
				return Error(err)
			}
			dictIn = mergeDict(dict, dictIn, env)
			return Unify(vm, dictOut, dictIn, cont, env)
		default:
			return Error(typeError(validTypePair, new, env))
		}
	default:
		return Error(typeError(validTypeDict, dictIn, env))
	}
}

// DelDict4 evaluates to a new dict where the key-value associated with key is removed from dictIn.
// It unifies value with the removed value and dictOut with the resulting dict. The predicate fails
// when key is not present in dictIn.
func DelDict4(vm *VM, key Term, dictIn Term, value Term, dictOut Term, cont Cont, env *Env) (promise *Promise) {
	defer ensurePromise(&promise)

	env = vm.ownedEnv(env)
	dictIn = env.Resolve(dictIn)
	switch dt := dictIn.(type) {
	case Variable:
		return Error(InstantiationError(env))
	case Dict:
		rk := env.Resolve(key)
		switch k := rk.(type) {
		case Variable:
			return Error(InstantiationError(env))
		case Atom:
			removed, ok := dt.Value(k)
			if !ok {
				return Bool(false)
			}

			return Unify(vm, value, removed, func(env *Env) *Promise {
				pairs := int64(dt.Len())
				if pairs < 1 {
					return Error(resourceError(resourceMemory, env))
				}
				pairs--
				args := makeTerms(addTermCells(addTermCells(pairs, pairs, env), 1, env), env)[:0]
				args = append(args, dt.Tag())

				dt.All()(func(kk Atom, vv Term) bool {
					if kk != k {
						args = append(args, kk, vv)
					}
					return true
				})

				newDict := newDict(args)
				return Unify(vm, dictOut, newDict, cont, env)
			}, env)
		default:
			return Error(domainError(validDomainDictKey, rk, env))
		}
	default:
		return Error(typeError(validTypeDict, dt, env))
	}
}

// mergeDict merges n into d returning a new Dict.
func mergeDict(n Dict, d Dict, env *Env) Dict {
	dLen, nLen := d.Len(), n.Len()
	pairs := addTermCells(int64(dLen), int64(nLen), env)
	args := makeTerms(addTermCells(addTermCells(pairs, pairs, env), 1, env), env)[:0]
	args = append(args, d.Tag())

	i, j := 0, 0
	for i < dLen && j < nLen {
		dk, dv, _ := d.At(i)
		nk, nv, _ := n.At(j)

		switch {
		case dk == nk:
			args = append(args, nk, nv)
			i++
			j++
		case dk < nk:
			args = append(args, dk, dv)
			i++
		default:
			args = append(args, nk, nv)
			j++
		}
	}

	for i < dLen {
		k, v, _ := d.At(i)
		args = append(args, k, v)
		i++
	}
	for j < nLen {
		k, v, _ := n.At(j)
		args = append(args, k, v)
		j++
	}

	return newDict(args)
}

func newDictFromListOfPairs(vm *VM, l Compound, env *Env) (d Dict, err error) {
	defer recoverMeterError(&err)

	args := makeTerms(1, env)[:0]
	args = append(args, vm.NewVariable())

	iter := ListIterator{List: l, Env: env}
	for iter.Next() {
		k, v, err := assertPair(iter.Current(), env)
		if err != nil {
			return nil, err
		}
		args = appendTerms(args, env, k, v)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return newDictWithEnv(args, env)
}

func assertPair(pair Term, env *Env) (Atom, Term, error) {
	switch pair := pair.(type) {
	case Compound:
		switch pair.Arity() {
		case 1: // Key(Value)
			return pair.Functor(), pair.Arg(0), nil
		case 2: // Key:Value, Key=Value, Key-Value
			switch pair.Functor() {
			case atomColon, atomEqual, atomMinus:
				if key, ok := pair.Arg(0).(Atom); ok {
					return key, pair.Arg(1), nil
				}
			}
		}
	}
	return "", nil, typeError(validTypePair, pair, env)
}

type duplicateKeyError struct {
	key Atom
}

func (e duplicateKeyError) Error() string {
	return fmt.Sprintf("duplicate key: %s", e.key)
}
