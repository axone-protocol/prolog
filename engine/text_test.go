package engine

import (
	"context"
	"embed"
	"errors"
	"io"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

//go:embed testdata
var testdata embed.FS

func mustOpen(fs fs.FS, name string) fs.File {
	f, err := fs.Open(name)
	if err != nil {
		panic(err)
	}
	return f
}

func TestVM_Compile(t *testing.T) {
	varCounter.count = 1

	tests := []struct {
		title  string
		text   string
		args   []interface{}
		err    error
		result *orderedmap.OrderedMap[procedureIndicator, procedure]
	}{
		{title: "shebang", text: `#!/foo/bar
foo(a).
`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("a")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("a")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},
		{title: "shebang: no following lines", text: `#!/foo/bar`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					multifile: true,
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("c")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},
		{title: "facts", text: `
foo(a).
foo(b).
`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("a")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("a")},
								{opcode: OpExit},
							},
						},
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("b")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("b")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},
		{title: "rules", text: `
bar :- true.
bar(X, "abc", [a, b], [a, b|Y], f(a)) :- X, !, foo(X, "abc", [a, b], [a, b|Y], f(a)).
`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					multifile: true,
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("c")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
			procedurePair{
				Key: procedureIndicator{name: NewAtom("bar"), arity: 0},
				Value: &userDefined{
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("bar"), arity: 0},
							raw: atomIf.Apply(NewAtom("bar"), atomTrue),
							bytecode: bytecode{
								{opcode: OpEnter},
								{opcode: OpCall, operand: procedureIndicator{name: atomTrue, arity: 0}},
								{opcode: OpExit},
							},
						},
					},
				},
			},
			procedurePair{
				Key: procedureIndicator{name: NewAtom("bar"), arity: 5},
				Value: &userDefined{
					clauses: clauses{
						{
							pi: procedureIndicator{name: NewAtom("bar"), arity: 5},
							raw: atomIf.Apply(
								NewAtom("bar").Apply(lastVariable()+1, charList("abc"), List(NewAtom("a"), NewAtom("b")), PartialList(lastVariable()+2, NewAtom("a"), NewAtom("b")), NewAtom("f").Apply(NewAtom("a"))),
								seq(atomComma,
									lastVariable()+1,
									atomCut,
									NewAtom("foo").Apply(lastVariable()+1, charList("abc"), List(NewAtom("a"), NewAtom("b")), PartialList(lastVariable()+2, NewAtom("a"), NewAtom("b")), NewAtom("f").Apply(NewAtom("a"))),
								),
							),
							vars: []Variable{lastVariable() + 1, lastVariable() + 2},
							bytecode: bytecode{
								{opcode: OpGetVar, operand: Integer(0)},
								{opcode: OpGetConst, operand: charList("abc")},
								{opcode: OpGetList, operand: Integer(2)},
								{opcode: OpGetConst, operand: NewAtom("a")},
								{opcode: OpGetConst, operand: NewAtom("b")},
								{opcode: OpPop},
								{opcode: OpGetPartial, operand: Integer(2)},
								{opcode: OpGetVar, operand: Integer(1)},
								{opcode: OpGetConst, operand: NewAtom("a")},
								{opcode: OpGetConst, operand: NewAtom("b")},
								{opcode: OpPop},
								{opcode: OpGetFunctor, operand: procedureIndicator{name: NewAtom("f"), arity: 1}},
								{opcode: OpGetConst, operand: NewAtom("a")},
								{opcode: OpPop},
								{opcode: OpEnter},
								{opcode: OpPutVar, operand: Integer(0)},
								{opcode: OpCall, operand: procedureIndicator{name: atomCall, arity: 1}},
								{opcode: OpCut},
								{opcode: OpPutVar, operand: Integer(0)},
								{opcode: OpPutConst, operand: charList("abc")},
								{opcode: OpPutList, operand: Integer(2)},
								{opcode: OpPutConst, operand: NewAtom("a")},
								{opcode: OpPutConst, operand: NewAtom("b")},
								{opcode: OpPop},
								{opcode: OpPutPartial, operand: Integer(2)},
								{opcode: OpPutVar, operand: Integer(1)},
								{opcode: OpPutConst, operand: NewAtom("a")},
								{opcode: OpPutConst, operand: NewAtom("b")},
								{opcode: OpPop},
								{opcode: OpPutFunctor, operand: procedureIndicator{name: NewAtom("f"), arity: 1}},
								{opcode: OpPutConst, operand: NewAtom("a")},
								{opcode: OpPop},
								{opcode: OpCall, operand: procedureIndicator{name: NewAtom("foo"), arity: 5}},
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},
		{title: "dict head", text: `
point(point{x: 5}).
`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					multifile: true,
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("c")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
			procedurePair{
				Key: procedureIndicator{name: NewAtom("point"), arity: 1},
				Value: &userDefined{
					clauses: clauses{
						{
							pi: procedureIndicator{name: NewAtom("point"), arity: 1},
							raw: &compound{functor: NewAtom("point"), args: []Term{
								&dict{compound: compound{functor: NewAtom("dict"), args: []Term{
									NewAtom("point"), NewAtom("x"), Integer(5)}}},
							}},
							bytecode: bytecode{
								{opcode: OpGetDict, operand: Integer(3)},
								{opcode: OpGetConst, operand: NewAtom("point")},
								{opcode: OpGetConst, operand: NewAtom("x")},
								{opcode: OpGetConst, operand: Integer(5)},
								{opcode: OpPop},
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},
		{title: "dict head (2)", text: `
point(point{x: 5}.x).
`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					multifile: true,
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("c")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
			procedurePair{
				Key: procedureIndicator{name: NewAtom("point"), arity: 1},
				Value: &userDefined{
					clauses: clauses{
						{
							pi: procedureIndicator{name: NewAtom("point"), arity: 1},
							raw: &compound{functor: "point", args: []Term{
								&compound{functor: "$dot", args: []Term{
									&dict{compound: compound{functor: "dict", args: []Term{NewAtom("point"), NewAtom("x"), Integer(5)}}},
									NewAtom("x"),
								}},
							}},
							vars: []Variable{lastVariable() + 1},
							bytecode: bytecode{
								{opcode: OpGetVar, operand: Integer(0)},
								{opcode: OpEnter},
								{opcode: OpPutDict, operand: Integer(3)},
								{opcode: OpPutConst, operand: NewAtom("point")},
								{opcode: OpPutConst, operand: NewAtom("x")},
								{opcode: OpPutConst, operand: Integer(5)},
								{opcode: OpPop},
								{opcode: OpPutConst, operand: NewAtom("x")},
								{opcode: OpPutVar, operand: Integer(0)},
								{opcode: OpCall, operand: procedureIndicator{name: atomDot, arity: Integer(3)}},
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},
		{title: "dict head (3)", text: `
point(point{x: 5}.x) :- true.
`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					multifile: true,
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("c")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
			procedurePair{
				Key: procedureIndicator{name: NewAtom("point"), arity: 1},
				Value: &userDefined{
					clauses: clauses{
						{
							pi: procedureIndicator{name: NewAtom("point"), arity: 1},
							raw: atomIf.Apply(
								&compound{functor: "point", args: []Term{
									&compound{functor: "$dot", args: []Term{
										&dict{compound: compound{functor: "dict", args: []Term{NewAtom("point"), NewAtom("x"), Integer(5)}}},
										NewAtom("x"),
									}}}},
								NewAtom("true")),
							vars: []Variable{lastVariable() + 1},
							bytecode: bytecode{
								{opcode: OpGetVar, operand: Integer(0)},
								{opcode: OpEnter},
								{opcode: OpCall, operand: procedureIndicator{name: atomTrue, arity: 0}},
								{opcode: OpPutDict, operand: Integer(3)},
								{opcode: OpPutConst, operand: NewAtom("point")},
								{opcode: OpPutConst, operand: NewAtom("x")},
								{opcode: OpPutConst, operand: Integer(5)},
								{opcode: OpPop},
								{opcode: OpPutConst, operand: NewAtom("x")},
								{opcode: OpPutVar, operand: Integer(0)},
								{opcode: OpCall, operand: procedureIndicator{name: atomDot, arity: Integer(3)}},
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},
		{title: "dict body", text: `
p :- foo(point{x: 5}).
`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					multifile: true,
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("c")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
			procedurePair{
				Key: procedureIndicator{name: NewAtom("p"), arity: 0},
				Value: &userDefined{
					clauses: clauses{
						{
							pi: procedureIndicator{name: NewAtom("p"), arity: 0},
							raw: atomIf.Apply(
								NewAtom("p"),
								&compound{functor: NewAtom("foo"), args: []Term{
									&dict{compound: compound{functor: NewAtom("dict"), args: []Term{
										NewAtom("point"), NewAtom("x"), Integer(5)},
									}}}}),
							bytecode: bytecode{
								{opcode: OpEnter},
								{opcode: OpPutDict, operand: Integer(3)},
								{opcode: OpPutConst, operand: NewAtom("point")},
								{opcode: OpPutConst, operand: NewAtom("x")},
								{opcode: OpPutConst, operand: Integer(5)},
								{opcode: OpPop},
								{opcode: OpCall, operand: procedureIndicator{name: NewAtom("foo"), arity: 1}},
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},
		{title: "dict body (2)", text: `
x(X) :- p(P), =(X, P.x).
`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					multifile: true,
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("c")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
			procedurePair{
				Key: procedureIndicator{name: NewAtom("x"), arity: 1},
				Value: &userDefined{
					clauses: clauses{
						{
							pi: procedureIndicator{name: NewAtom("x"), arity: 1},
							raw: atomIf.Apply(
								NewAtom("x").Apply(lastVariable()+1),
								seq(
									atomComma,
									NewAtom("p").Apply(lastVariable()+2),
									atomEqual.Apply(lastVariable()+1, NewAtom("$dot").Apply(lastVariable()+2, NewAtom("x"))),
								)),
							vars: []Variable{lastVariable() + 1, lastVariable() + 2, lastVariable() + 3},
							bytecode: bytecode{
								{opcode: OpGetVar, operand: Integer(0)},
								{opcode: OpEnter},
								{opcode: OpPutVar, operand: Integer(1)},
								{opcode: OpCall, operand: procedureIndicator{name: NewAtom("p"), arity: 1}},
								{opcode: OpPutVar, operand: Integer(1)},
								{opcode: OpPutConst, operand: NewAtom("x")},
								{opcode: OpPutVar, operand: Integer(2)},
								{opcode: OpCall, operand: procedureIndicator{name: NewAtom("."), arity: 3}},
								{opcode: OpPutVar, operand: Integer(0)},
								{opcode: OpPutVar, operand: Integer(2)},
								{opcode: OpCall, operand: procedureIndicator{name: NewAtom("="), arity: 2}},
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},
		{title: "dynamic", text: `
:- dynamic(foo/1).
foo(a).
foo(b).
`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					public:  true,
					dynamic: true,
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("a")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("a")},
								{opcode: OpExit},
							},
						},
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("b")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("b")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},
		{title: "multifile", text: `
:- multifile(foo/1).
foo(a).
foo(b).
`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					multifile: true,
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("c")},
								{opcode: OpExit},
							},
						},
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("a")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("a")},
								{opcode: OpExit},
							},
						},
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("b")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("b")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},
		{title: "discontiguous", text: `
:- discontiguous(foo/1).
foo(a).
bar(a).
foo(b).
`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					discontiguous: true,
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("a")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("a")},
								{opcode: OpExit},
							},
						},
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("b")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("b")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
			procedurePair{
				Key: procedureIndicator{name: NewAtom("bar"), arity: 1},
				Value: &userDefined{
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("bar"), arity: 1},
							raw: &compound{functor: NewAtom("bar"), args: []Term{NewAtom("a")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("a")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},
		{title: "include", text: `
:- include('testdata/foo').
`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					multifile: true,
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("c")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 0},
				Value: &userDefined{
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 0},
							raw: NewAtom("foo"),
							bytecode: bytecode{
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},
		{title: "ensure_loaded", text: `
:- ensure_loaded('testdata/foo').
`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					multifile: true,
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("c")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 0},
				Value: &userDefined{
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 0},
							raw: NewAtom("foo"),
							bytecode: bytecode{
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},
		{title: "initialization", text: `
:- initialization(foo(c)).
`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					multifile: true,
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("c")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},
		{title: "predicate-backed directive", text: `
:- foo(c).
`, result: buildOrderedMap(
			procedurePair{
				Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
				Value: &userDefined{
					multifile: true,
					clauses: clauses{
						{
							pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
							raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}},
							bytecode: bytecode{
								{opcode: OpGetConst, operand: NewAtom("c")},
								{opcode: OpExit},
							},
						},
					},
				},
			},
		)},

		{title: "error: invalid argument", text: `
foo(?).
`, args: []interface{}{nil}, err: errors.New("can't convert to term: <invalid reflect.Value>")},
		{title: "error: syntax error", text: `
foo().
`, err: unexpectedTokenError{actual: Token{kind: tokenClose, val: ")"}}},
		{title: "error: expansion error", text: `
:- ensure_loaded('testdata/break_term_expansion').
foo(a).
`, err: Exception{term: NewAtom("ball")}},
		{title: "error: variable fact", text: `
X.
`, err: InstantiationError(nil)},
		{title: "error: variable rule", text: `
X :- X.
`, err: InstantiationError(nil)},
		{title: "error: non-callable rule body", text: `
foo :- 1.
`, err: typeError(validTypeCallable, Integer(1), nil)},
		{title: "error: non-PI argument, variable", text: `:- dynamic(PI).`, err: InstantiationError(nil)},
		{title: "error: non-PI argument, not compound", text: `:- dynamic(foo).`, err: typeError(validTypePredicateIndicator, NewAtom("foo"), nil)},
		{title: "error: non-PI argument, compound", text: `:- dynamic(foo(a, b)).`, err: typeError(validTypePredicateIndicator, NewAtom("foo").Apply(NewAtom("a"), NewAtom("b")), nil)},
		{title: "error: non-PI argument, name is variable", text: `:- dynamic(Name/2).`, err: InstantiationError(nil)},
		{title: "error: non-PI argument, arity is variable", text: `:- dynamic(foo/Arity).`, err: InstantiationError(nil)},
		{title: "error: non-PI argument, arity is not integer", text: `:- dynamic(foo/bar).`, err: typeError(validTypePredicateIndicator, atomSlash.Apply(NewAtom("foo"), NewAtom("bar")), nil)},
		{title: "error: non-PI argument, name is not atom", text: `:- dynamic(0/2).`, err: typeError(validTypePredicateIndicator, atomSlash.Apply(Integer(0), Integer(2)), nil)},
		{title: "error: included variable", text: `
:- include(X).
`, err: InstantiationError(nil)},
		{title: "error: included file not found", text: `
:- include('testdata/not_found').
`, err: existenceError(objectTypeSourceSink, NewAtom("testdata/not_found"), nil)},
		{title: "error: included non-atom", text: `
:- include(1).
`, err: typeError(validTypeAtom, Integer(1), nil)},
		{title: "error: initialization exception", text: `
:- initialization(bar).
`, err: existenceError(objectTypeProcedure, atomSlash.Apply(NewAtom("bar"), Integer(0)), nil)},
		{title: "error: initialization failure", text: `
:- initialization(foo(d)).
`, err: errors.New("failed initialization goal: foo(d)")},
		{title: "error: predicate-backed directive exception", text: `
:- bar.
`, err: existenceError(objectTypeProcedure, atomSlash.Apply(NewAtom("bar"), Integer(0)), nil)},
		{title: "error: predicate-backed directive failure", text: `
:- foo(d).
`, err: errors.New("failed directive: foo(d)")},
		{title: "error: discontiguous, end of text", text: `
foo(a).
bar(a).
foo(b).
`, err: &discontiguousError{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}}},
		{title: "error: discontiguous, before directive", text: `
foo(a).
bar(a).
foo(b).
:- foo(c).
`, err: &discontiguousError{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}}},
		{title: "error: discontiguous, before other facts", text: `
foo(a).
bar(a).
foo(b).
bar(b).
`, err: &discontiguousError{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}}},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			var vm VM
			varCounter.count = 1 // Global var cause issues in testing environment that call in randomly order for checking equality between procedure clause args

			vm.getOperators().define(1200, operatorSpecifierXFX, atomIf)
			vm.getOperators().define(1200, operatorSpecifierXFX, atomArrow)
			vm.getOperators().define(1200, operatorSpecifierFX, atomIf)
			vm.getOperators().define(1000, operatorSpecifierXFY, atomComma)
			vm.getOperators().define(400, operatorSpecifierYFX, atomSlash)
			vm.procedures = buildOrderedMap(
				procedurePair{
					Key: procedureIndicator{name: NewAtom("foo"), arity: 1},
					Value: &userDefined{
						multifile: true,
						clauses: clauses{
							{
								pi:  procedureIndicator{name: NewAtom("foo"), arity: 1},
								raw: &compound{functor: NewAtom("foo"), args: []Term{NewAtom("c")}},
								bytecode: bytecode{
									{opcode: OpGetConst, operand: NewAtom("c")},
									{opcode: OpExit},
								},
							},
						},
					},
				},
			)
			vm.FS = testdata
			vm.Register1(NewAtom("throw"), Throw)
			assert.Equal(t, tt.err, vm.Compile(context.Background(), tt.text, tt.args...))
			if tt.err == nil {
				vm.procedures.Delete(procedureIndicator{name: NewAtom("throw"), arity: 1})
				assert.EqualValues(t, tt.result, vm.procedures)
			}
		})
	}
}

func TestVM_Consult(t *testing.T) {
	x := NewVariable()

	tests := []struct {
		title string
		files Term
		ok    bool
		err   error
	}{
		{title: `:- consult('testdata/empty.txt').`, files: NewAtom("testdata/empty.txt"), ok: true},
		{title: `:- consult([]).`, files: List(), ok: true},
		{title: `:- consult(['testdata/empty.txt']).`, files: List(NewAtom("testdata/empty.txt")), ok: true},
		{title: `:- consult(['testdata/empty.txt', 'testdata/empty.txt']).`, files: List(NewAtom("testdata/empty.txt"), NewAtom("testdata/empty.txt")), ok: true},

		{title: `:- consult('testdata/abc.txt').`, files: NewAtom("testdata/abc.txt"), err: io.EOF},
		{title: `:- consult(['testdata/abc.txt']).`, files: List(NewAtom("testdata/abc.txt")), err: io.EOF},

		{title: `:- consult(X).`, files: x, err: InstantiationError(nil)},
		{title: `:- consult(foo(bar)).`, files: NewAtom("foo").Apply(NewAtom("bar")), err: typeError(validTypeAtom, NewAtom("foo").Apply(NewAtom("bar")), nil)},
		{title: `:- consult(1).`, files: Integer(1), err: typeError(validTypeAtom, Integer(1), nil)},
		{title: `:- consult(['testdata/empty.txt'|_]).`, files: PartialList(NewVariable(), NewAtom("testdata/empty.txt")), err: typeError(validTypeAtom, PartialList(NewVariable(), NewAtom("testdata/empty.txt")), nil)},
		{title: `:- consult([X]).`, files: List(x), err: InstantiationError(nil)},
		{title: `:- consult([1]).`, files: List(Integer(1)), err: typeError(validTypeAtom, Integer(1), nil)},

		{title: `:- consult('testdata/not_found.txt').`, files: NewAtom("testdata/not_found.txt"), err: existenceError(objectTypeSourceSink, NewAtom("testdata/not_found.txt"), nil)},
		{title: `:- consult(['testdata/not_found.txt']).`, files: List(NewAtom("testdata/not_found.txt")), err: existenceError(objectTypeSourceSink, NewAtom("testdata/not_found.txt"), nil)},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			vm := VM{
				FS: testdata,
			}
			ok, err := Consult(&vm, tt.files, Success, nil).Force(context.Background())
			assert.Equal(t, tt.ok, ok)
			if e, ok := tt.err.(Exception); ok {
				_, ok := NewEnv().Unify(e.Term(), err.(Exception).Term())
				assert.True(t, ok)
			} else {
				assert.Equal(t, tt.err, err)
			}
		})
	}
}

func TestDiscontiguousError_Error(t *testing.T) {
	e := discontiguousError{pi: procedureIndicator{name: NewAtom("foo"), arity: 1}}
	assert.Equal(t, "foo/1 is discontiguous", e.Error())
}
