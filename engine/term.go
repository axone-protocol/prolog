package engine

import (
	"fmt"
	"io"
	"strings"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

// Term is a prolog term.
type Term interface {
	WriteTerm(w io.Writer, opts *WriteOptions, env *Env) error
	Compare(t Term, env *Env) int
}

// WriteOptions specify how the Term writes itself.
type WriteOptions struct {
	ignoreOps     bool
	quoted        bool
	variableNames map[Variable]Atom
	numberVars    bool

	_ops        *operators
	priority    Integer
	visited     map[termID]struct{}
	left, right operator
	maxDepth    Integer
}

func (o WriteOptions) withQuoted(quoted bool) *WriteOptions {
	o.quoted = quoted
	return &o
}

func (o WriteOptions) withVisited(t Term) *WriteOptions {
	visited := make(map[termID]struct{}, len(o.visited))
	for k, v := range o.visited {
		visited[k] = v
	}
	visited[id(t)] = struct{}{}
	o.visited = visited
	return &o
}

func (o WriteOptions) withPriority(priority Integer) *WriteOptions {
	o.priority = priority
	return &o
}

func (o WriteOptions) withLeft(op operator) *WriteOptions {
	o.left = op
	return &o
}

func (o WriteOptions) withRight(op operator) *WriteOptions {
	o.right = op
	return &o
}

func (o WriteOptions) getOps() *operators {
	if o._ops == nil {
		o._ops = newOperators()
	}
	return o._ops
}

var defaultWriteOptions = WriteOptions{
	_ops: &operators{
		OrderedMap: orderedmap.New[Atom, [_operatorClassLen]operator](
			orderedmap.WithInitialData(
				orderedmap.Pair[Atom, [_operatorClassLen]operator]{
					Key: atomPlus,
					Value: [_operatorClassLen]operator{
						operatorClassInfix: {priority: 500, specifier: operatorSpecifierYFX, name: atomPlus}, // for flag+value
					},
				},
				orderedmap.Pair[Atom, [_operatorClassLen]operator]{
					Key: atomSlash,
					Value: [_operatorClassLen]operator{
						operatorClassInfix: {priority: 400, specifier: operatorSpecifierYFX, name: atomSlash}, // for principal functors
					},
				},
			),
		),
	},
	variableNames: map[Variable]Atom{},
	priority:      1200,
}

// CompareAtomic compares a custom atomic term of type T with a Term and returns -1, 0, or 1.
// The order is Variable < Float < Integer < Atom < custom atomic terms < Compound
// where different types of custom atomic terms are ordered by the Go-syntax representation of the types.
// It compares values of the same custom atomic term type T by the provided comparison function.
func CompareAtomic[T Term](a T, t Term, cmp func(T, T) int, env *Env) int {
	switch t := env.Resolve(t).(type) {
	case Variable, Float, Integer, Atom:
		return 1
	case T:
		return cmp(a, t)
	case Compound:
		return -1
	default: // Custom atomic term.
		return strings.Compare(fmt.Sprintf("%T", a), fmt.Sprintf("%T", t))
	}
}

// termIDer lets a Term which is not comparable per se return its termID for comparison.
type termIDer interface {
	termID() termID
}

// termID is an identifier for a Term.
type termID interface{}

// id returns a termID for the Term.
func id(t Term) termID {
	switch t := t.(type) {
	case termIDer:
		return t.termID()
	default:
		return t // Assuming it's comparable.
	}
}
