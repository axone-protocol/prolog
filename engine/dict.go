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

var (
	atomColon = NewAtom(":")
)

var (
	// predefinedFuncs are the predefined (reserved) functions that can be called on a Dict.
	predefinedFuncs = map[Atom]func(*VM, Term, Term, Term, Cont, *Env) *Promise{
		"get": GetDict3,
		"put": PutDict3,
		// TODO: to continue (https://www.swi-prolog.org/pldoc/man?section=ext-dicts-predefined)
	}
)

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
	args, err := processArgs(args)
	if err != nil {
		return nil, err
	}
	return newDict(args), nil
}

func newDict(args []Term) Dict {
	return &dict{
		compound: compound{
			functor: atomDict,
			args:    args,
		},
	}
}

func processArgs(args []Term) ([]Term, error) {
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

	processedArgs := make([]Term, 0, len(rest))
	processedArgs = append(processedArgs, tag)
	for _, key := range keys {
		processedArgs = append(processedArgs, key, kv[key])
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
func Op3(vm *VM, dict, function, result Term, cont Cont, env *Env) *Promise {
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
			if f, ok := predefinedFuncs[function.Functor()]; ok && function.Arity() == 1 {
				return f(vm, function.Arg(0), dict, result, cont, env)
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
					tempA := NewVariable()
					return GetDict3(vm, keyPath.Arg(0), dict, tempA, func(env *Env) *Promise {
						tempB := NewVariable()
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

// PutDict3 evaluates to a new dict where the key-values in dictIn replace or extend the key-values in the original dict.
//
// new is either a dict or list of attribute-value pairs using the syntax Key:Value, Key=Value, Key-Value or Key(Value)
func PutDict3(vm *VM, new Term, dictIn Term, dictOut Term, cont Cont, env *Env) *Promise {
	switch dictIn := env.Resolve(dictIn).(type) {
	case Variable:
		return Error(InstantiationError(env))
	case Dict:
		switch new := env.Resolve(new).(type) {
		case Variable:
			return Error(InstantiationError(env))
		case Dict:
			dictIn = mergeDict(new, dictIn)
			return Unify(vm, dictOut, dictIn, cont, env)
		case Compound:
			dict, err := newDictFromListOfPairs(new, env)
			if err != nil {
				return Error(err)
			}
			dictIn = mergeDict(dict, dictIn)
			return Unify(vm, dictOut, dictIn, cont, env)
		default:
			return Error(typeError(validTypePair, new, env))
		}
	default:
		return Error(typeError(validTypeDict, dictIn, env))
	}
}

// mergeDict merge n into d returning a new Dict.
func mergeDict(n Dict, d Dict) Dict {
	totalLen := d.Len() + n.Len()
	args := make([]Term, 0, totalLen*2+1)
	args = append(args, d.Tag())

	dPairs := make([]Term, 0, d.Len()*2)
	for k, v := range d.All() {
		dPairs = append(dPairs, k, v)
	}

	nPairs := make([]Term, 0, n.Len()*2)
	for k, v := range n.All() {
		nPairs = append(nPairs, k, v)
	}

	i, j := 0, 0
	for i < len(dPairs) && j < len(nPairs) {
		dk, nk := dPairs[i].(Atom), nPairs[j].(Atom)

		switch {
		case dk == nk:
			args = append(args, nk, nPairs[j+1])
			i += 2
			j += 2
		case dk < nk:
			args = append(args, dk, dPairs[i+1])
			i += 2
		case nk < dk:
			args = append(args, nk, nPairs[j+1])
			j += 2
		}
	}

	for i < len(dPairs) {
		args = append(args, dPairs[i], dPairs[i+1])
		i += 2
	}

	for j < len(nPairs) {
		args = append(args, nPairs[j], nPairs[j+1])
		j += 2
	}

	return newDict(args)
}

func newDictFromListOfPairs(l Compound, env *Env) (Dict, error) {
	var args []Term
	args = append(args, NewVariable())

	iter := ListIterator{List: l, Env: env}
	for iter.Next() {
		k, v, err := assertPair(iter.Current(), env)
		if err != nil {
			return nil, err
		}
		args = append(args, k, v)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return NewDict(args)
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
