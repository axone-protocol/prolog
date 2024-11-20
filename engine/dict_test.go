package engine

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestNewDict(t *testing.T) {
	tests := []struct {
		name    string
		args    []Term
		want    Dict
		wantErr string
	}{
		{
			name: "valid dict",
			args: []Term{NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)},
			want: makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
		},
		{
			name: "valid empty dict",
			args: []Term{NewAtom("empty")},
			want: makeDict(NewAtom("empty")),
		},
		{
			name: "valid dict with not ordered keys",
			args: []Term{NewAtom("point"), NewAtom("y"), Integer(2), NewAtom("x"), Integer(1)},
			want: makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
		},
		{
			name:    "invalid dict with even number of args",
			args:    []Term{NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y")},
			wantErr: "invalid dict",
		},
		{
			name:    "invalid dict with no args",
			args:    []Term{},
			wantErr: "invalid dict",
		},
		{
			name:    "invalid dict with non-atom key",
			args:    []Term{NewAtom("point"), Integer(1), Integer(1), NewAtom("y"), Integer(2)},
			wantErr: "key expected",
		},
		{
			name:    "invalid dict with duplicate keys",
			args:    []Term{NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("x"), Integer(2)},
			wantErr: "duplicate key: x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDict(tt.args)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDictCompare(t *testing.T) {
	tests := []struct {
		name         string
		thisDictArgs []Term
		thatDictArgs []Term
		want         int
	}{
		{
			name:         "equal",
			thisDictArgs: []Term{NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)},
			thatDictArgs: []Term{NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)},
			want:         0,
		},
		{
			name:         "lower than",
			thisDictArgs: []Term{NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)},
			thatDictArgs: []Term{NewAtom("point"), NewAtom("x"), Integer(2), NewAtom("y"), Integer(3)},
			want:         -1,
		},
		{
			name:         "greater than",
			thisDictArgs: []Term{NewAtom("point"), NewAtom("x"), Integer(2), NewAtom("y"), Integer(3)},
			thatDictArgs: []Term{NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)},
			want:         1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := NewEnv()
			thisDict, err := NewDict(tt.thisDictArgs)
			assert.NoError(t, err)
			thatDict, err := NewDict(tt.thatDictArgs)
			assert.NoError(t, err)

			got := thisDict.Compare(thatDict, env)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDictTag(t *testing.T) {
	tests := []struct {
		name string
		dict Dict
		want Atom
	}{
		{
			name: "simple dict",
			dict: makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
			want: NewAtom("point"),
		},
		{
			name: "empty dict",
			dict: &dict{
				compound: compound{
					functor: atomDict,
					args:    []Term{NewAtom("empty")}}},
			want: NewAtom("empty"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := tt.dict.Tag()
			assert.Equal(t, tt.want, got)

		})
	}
}

func TestDictAll(t *testing.T) {
	tests := []struct {
		name      string
		dict      Dict
		wantPairs []orderedmap.Pair[Atom, Term]
	}{
		{
			name: "simple dict",
			dict: makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
			wantPairs: []orderedmap.Pair[Atom, Term]{
				{Key: NewAtom("x"), Value: Integer(1)},
				{Key: NewAtom("y"), Value: Integer(2)},
			},
		},
		{
			name:      "empty dict",
			dict:      makeDict(NewAtom("empty")),
			wantPairs: []orderedmap.Pair[Atom, Term]{},
		},
		{
			name: "dict with nested dict",
			dict: makeDict(
				NewAtom("point"),
				NewAtom("x"), Integer(1),
				NewAtom("y"), Integer(2),
				NewAtom("z"), makeDict(NewAtom("nested"), NewAtom("foo"), NewAtom("bar"))),
			wantPairs: []orderedmap.Pair[Atom, Term]{
				{Key: NewAtom("x"), Value: Integer(1)},
				{Key: NewAtom("y"), Value: Integer(2)},
				{Key: NewAtom("z"), Value: makeDict(NewAtom("nested"), NewAtom("foo"), NewAtom("bar")),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			want := orderedmap.New[Atom, Term]()
			want.AddPairs(tt.wantPairs...)

			got := orderedmap.New[Atom, Term]()
			tt.dict.All()(func(k Atom, v Term) bool {
				got.Set(k, v)
				return true
			})

			assert.Equal(t, want, got)
		})
	}
}

func TestDictLen(t *testing.T) {
	tests := []struct {
		name string
		dict Dict
		want int
	}{
		{
			name: "simple dict",
			dict: makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
			want: 2,
		},
		{
			name: "empty dict",
			dict: makeDict(NewAtom("empty")),
			want: 0,
		},
		{
			name: "dict with nested dict",
			dict: makeDict(
				NewAtom("point"),
				NewAtom("x"), Integer(1),
				NewAtom("y"), Integer(2),
				NewAtom("z"), makeDict(NewAtom("nested"), NewAtom("foo"), NewAtom("bar"))),
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := tt.dict.Len()
			assert.Equal(t, tt.want, got)

		})
	}
}

func TestDictWrite(t *testing.T) {
	tests := []struct {
		name string
		dict Dict
		want string
	}{
		{
			name: "simple dict",
			dict: makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
			want: "point{x:1,y:2}",
		},
		{
			name: "empty dict",
			dict: makeDict(NewAtom("empty")),
			want: "empty{}",
		},
		{
			name: "dict with nested dict",
			dict: makeDict(
				NewAtom("point"),
				NewAtom("x"), Integer(1),
				NewAtom("y"), Integer(2),
				NewAtom("z"), makeDict(NewAtom("nested"), NewAtom("foo"), NewAtom("bar"))),
			want: "point{x:1,y:2,z:nested{foo:bar}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var env *Env
			var buf bytes.Buffer
			err := tt.dict.WriteTerm(&buf, &WriteOptions{quoted: false}, env)
			assert.NoError(t, err)
			got := buf.String()
			assert.Equal(t, tt.want, got)

		})
	}
}
func TestDictValue(t *testing.T) {
	tests := []struct {
		name      string
		dict      Dict
		key       Atom
		wantValue Term
		wantFound bool
	}{
		{
			name:      "key exists",
			dict:      makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
			key:       NewAtom("x"),
			wantValue: Integer(1),
			wantFound: true,
		},
		{
			name:      "key does not exist",
			dict:      makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
			key:       NewAtom("z"),
			wantValue: nil,
			wantFound: false,
		},
		{
			name:      "empty dict",
			dict:      makeDict(NewAtom("empty")),
			key:       NewAtom("x"),
			wantValue: nil,
			wantFound: false,
		},
		{
			name: "nested dict",
			dict: makeDict(
				NewAtom("point"),
				NewAtom("x"), Integer(1),
				NewAtom("y"), Integer(2),
				NewAtom("z"), makeDict(NewAtom("nested"), NewAtom("foo"), NewAtom("bar"))),
			key:       NewAtom("z"),
			wantValue: makeDict(NewAtom("nested"), NewAtom("foo"), NewAtom("bar")),
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotValue, gotFound := tt.dict.Value(tt.key)
			assert.Equal(t, tt.wantValue, gotValue)
			assert.Equal(t, tt.wantFound, gotFound)
		})
	}
}
func TestDictAt(t *testing.T) {
	tests := []struct {
		name      string
		dict      Dict
		index     int
		wantKey   Atom
		wantValue Term
		wantFound bool
	}{
		{
			name:      "valid index",
			dict:      makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
			index:     0,
			wantKey:   NewAtom("x"),
			wantValue: Integer(1),
			wantFound: true,
		},
		{
			name:      "valid index second pair",
			dict:      makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
			index:     1,
			wantKey:   NewAtom("y"),
			wantValue: Integer(2),
			wantFound: true,
		},
		{
			name:      "index out of bounds negative",
			dict:      makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
			index:     -1,
			wantKey:   "",
			wantValue: nil,
			wantFound: false,
		},
		{
			name:      "index out of bounds positive",
			dict:      makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
			index:     2,
			wantKey:   "",
			wantValue: nil,
			wantFound: false,
		},
		{
			name:      "empty dict",
			dict:      makeDict(NewAtom("empty")),
			index:     0,
			wantKey:   "",
			wantValue: nil,
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotKey, gotValue, gotFound := tt.dict.At(tt.index)
			assert.Equal(t, tt.wantKey, gotKey)
			assert.Equal(t, tt.wantValue, gotValue)
			assert.Equal(t, tt.wantFound, gotFound)
		})
	}
}
func TestOp3(t *testing.T) {
	tests := []struct {
		name       string
		dict       Term
		function   Term
		wantResult Term
		wantError  string
	}{
		// Access
		{
			name:       "access existing key",
			dict:       makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
			function:   NewAtom("x"),
			wantResult: Integer(1),
		},
		{
			name:      "access non-existing key",
			dict:      makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
			function:  NewAtom("z"),
			wantError: "error(domain_error(dict_key,z),root)",
		},
		// Pathological
		{
			name:      "invalid dict type",
			dict:      Integer(42),
			function:  NewAtom("x"),
			wantError: "error(type_error(dict,42),root)",
		},
		{
			name:      "not enough instantiated",
			dict:      NewVariable(),
			function:  NewAtom("x"),
			wantError: "error(instantiation_error,root)",
		},
		{
			name:      "invalid function type",
			dict:      makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
			function:  Integer(1),
			wantError: "error(type_error(callable,1),root)",
		},
		{
			name:      "invalid function name",
			dict:      makeDict(NewAtom("point"), NewAtom("x"), Integer(1), NewAtom("y"), Integer(2)),
			function:  NewAtom("foo").Apply(NewAtom("bar")),
			wantError: "error(existence_error(procedure,foo(bar)),root)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var vm VM
			var env *Env

			result := NewVariable()
			ok, err := Op3(&vm, tt.dict, tt.function, result, func(env *Env) *Promise {
				assert.Equal(t, tt.wantResult, env.Resolve(result))
				return Bool(true)
			}, env).Force(context.Background())

			if tt.wantError != "" {
				assert.False(t, ok)
				assert.EqualError(t, err, tt.wantError)
			} else {
				if tt.wantResult != nil {
					assert.True(t, ok)
					assert.NoError(t, err)
				} else {
					assert.False(t, ok)
				}
			}
		})
	}
}

func makeDict(args ...Term) Dict {
	return newDict(args)
}
