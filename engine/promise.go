package engine

import (
	"context"
	"fmt"
)

var (
	truePromise  = &Promise{ok: true}
	falsePromise = &Promise{ok: false}
)

// PromiseFunc defines the type of a function that returns a promise.
type PromiseFunc = func(context.Context) *Promise

// NextFunc defines the type of a function that returns the next PromiseFunc in a sequence,
// along with a boolean indicating whether the returned value is valid.
type NextFunc = func() (PromiseFunc, bool)

// Promise is a delayed execution that results in (bool, error). The zero value for Promise is equivalent to Bool(false).
type Promise struct {
	// delayed execution with multiple choices
	delayed *NextFunc

	// final result
	ok  bool
	err error

	// execution control
	cutParent *Promise
	repeat    bool
	recover   func(error) *Promise
}

// Delay delays an execution of k.
// Should be used with reasonable quantity of k, otherwise prefer DelaySeq.
func Delay(k ...PromiseFunc) *Promise {
	return DelaySeq(makeNextFunc(k...))
}

// DelaySeq delays an execution of a sequence of promises.
func DelaySeq(next NextFunc) *Promise {
	return &Promise{
		delayed: &next,
	}
}

// Bool returns a promise that simply returns (ok, nil).
func Bool(ok bool) *Promise {
	if ok {
		return truePromise
	}
	return falsePromise
}

// Error returns a promise that simply returns (false, err).
func Error(err error) *Promise {
	return &Promise{err: err}
}

var dummyCutParent Promise

// cut returns a promise that once the execution reaches it, it eliminates other possible choices.
func cut(parent *Promise, k PromiseFunc) *Promise {
	if parent == nil {
		parent = &dummyCutParent
	}
	next := makeNextFunc(k)
	return &Promise{
		delayed:   &next,
		cutParent: parent,
	}
}

// repeat returns a promise that repeats k.
func repeat(k PromiseFunc) *Promise {
	next := makeNextFunc(k)
	return &Promise{
		delayed: &next,
		repeat:  true,
	}
}

// catch returns a promise with a recovering function.
// Once a promise results in error, the error goes through ancestor promises looking for a recovering function that
// returns a non-nil promise to continue on.
func catch(recover func(error) *Promise, k PromiseFunc) *Promise {
	next := makeNextFunc(k)

	return &Promise{
		delayed: &next,
		recover: recover,
	}
}

// Force enforces the delayed execution and returns the result. (i.e. trampoline)
func (p *Promise) Force(ctx context.Context) (ok bool, err error) {
	stack := promiseStack{p}
	for len(stack) > 0 {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		default:
			p := stack.pop()

			if p.delayed == nil {
				switch {
				case p.err != nil:
					if err := stack.recover(p.err); err != nil {
						return false, err
					}
					continue
				case p.ok:
					return true, nil
				default:
					continue
				}
			}

			// If cut, we eliminate other possibilities.
			if p.cutParent != nil {
				stack.popUntil(p.cutParent)
				p.cutParent = nil // we don't have to do this again when we revisit.
			}

			// Try the child promises from left to right.
			q := p.child(ctx)
			if q == nil {
				stack = append(stack, p)
			} else {
				stack = append(stack, p, q)
			}
		}
	}
	return false, nil
}

func (p *Promise) child(ctx context.Context) (promise *Promise) {
	defer ensurePromise(&promise)

	promiseFn, ok := (*p.delayed)()
	if !ok {
		p.delayed = nil
		return nil
	}

	promise = promiseFn(ctx)

	if p.repeat {
		nextFunc := makeNextFunc(promiseFn)
		p.delayed = &nextFunc
	}

	return
}

func ensurePromise(p **Promise) {
	if r := recover(); r != nil {
		*p = Error(panicError(r))
	}
}

func panicError(r interface{}) error {
	switch r := r.(type) {
	case Exception:
		return r
	case error:
		return Exception{term: atomError.Apply(NewAtom("panic_error").Apply(NewAtom(r.Error())))}
	default:
		return Exception{term: atomError.Apply(NewAtom("panic_error").Apply(NewAtom(fmt.Sprintf("%v", r))))}
	}
}

// makeNextFunc creates a NextFunc that iterates over a list of PromiseFunc.
// It returns the next PromiseFunc in the list and a boolean indicating if a valid function was returned.
// Once all PromiseFuncs are consumed, the boolean will be false.
func makeNextFunc(k ...PromiseFunc) NextFunc {
	return func() (PromiseFunc, bool) {
		if len(k) == 0 {
			return nil, false
		}
		f := k[0]
		k = k[1:]
		return f, true
	}
}

type promiseStack []*Promise

func (s *promiseStack) pop() *Promise {
	var p *Promise
	p, *s, (*s)[len(*s)-1] = (*s)[len(*s)-1], (*s)[:len(*s)-1], nil
	return p
}

func (s *promiseStack) popUntil(p *Promise) {
	for len(*s) > 0 {
		if pop := s.pop(); pop == p {
			break
		}
	}
}

func (s *promiseStack) recover(err error) error {
	// look for an ancestor promise with a recovering function that is applicable to the error.
	for len(*s) > 0 {
		pop := s.pop()
		if pop.recover == nil {
			continue
		}
		if q := pop.recover(err); q != nil {
			*s = append(*s, q)
			return nil
		}
	}

	// went through all the ancestor promises and still got the unhandled error.
	return err
}

// PanicError is an error thrown once panic occurs during the execution of a promise.
type PanicError struct {
	OriginErr error
}

func (p PanicError) Error() string {
	return fmt.Sprintf("panic: %v", p.OriginErr)
}
