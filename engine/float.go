package engine

import (
	"fmt"
	"github.com/cockroachdb/apd"
	"io"
	"strings"
)

// Float is a prolog floating-point number.
// The underlying implementation is not based on floating-point, it's a [GDA](https://speleotrove.com/decimal/)
// compatible implementation to avoid approximation and determinism issues.
// It uses under the hood a decimal128 with 34 precision digits.
type Float struct {
	dec *apd.Decimal
}

// The context that must be used for operations on Float.
var decimal128Ctx = apd.Context{
	Precision:   34,
	MaxExponent: 6144,
	MinExponent: -6143,
	Traps:       apd.DefaultTraps,
}

func NewFloatFromString(s string) (Float, error) {
	dec, c, err := decimal128Ctx.NewFromString(s)
	if err != nil {
		return Float{}, decimalConditionAsErr(c)
	}

	return Float{dec: dec}, nil
}

func NewFloatFromInt64(i int64) Float {
	var dec apd.Decimal
	dec.SetInt64(i)

	return Float{dec: &dec}
}

func decimalConditionAsErr(flags apd.Condition) error {
	e := flags & decimal128Ctx.Traps
	if e == 0 {
		return exceptionalValueUndefined
	}

	for m := apd.Condition(1); m > 0; m <<= 1 {
		err := e & m
		if err == 0 {
			continue
		}

		switch err {
		case apd.Overflow:
			return exceptionalValueFloatOverflow
		case apd.Underflow:
			return exceptionalValueUnderflow
		case apd.Subnormal:
			return exceptionalValueUnderflow
		case apd.DivisionByZero:
			return exceptionalValueZeroDivisor
		default:
			return exceptionalValueUndefined
		}
	}

	return exceptionalValueUndefined
}

func (f Float) number() {}

// WriteTerm outputs the Float to an io.Writer.
func (f Float) WriteTerm(_ *VM, w io.Writer, opts *WriteOptions, _ *Env) error {
	ew := errWriter{w: w}
	openClose := opts.left.name == atomMinus && opts.left.specifier.class() == operatorClassPrefix && !f.Negative()

	if openClose || (f.Negative() && opts.left != operator{}) {
		_, _ = ew.Write([]byte(" "))
	}

	if openClose {
		_, _ = ew.Write([]byte("("))
	}

	s := fmt.Sprintf("%g", f.dec)
	if !strings.ContainsRune(s, '.') {
		if strings.ContainsRune(s, 'e') {
			s = strings.Replace(s, "e", ".0e", 1)
		} else {
			s += ".0"
		}
	}
	_, _ = ew.Write([]byte(s))

	if openClose {
		_, _ = ew.Write([]byte(")"))
	}

	if !openClose && opts.right != (operator{}) && (opts.right.name == atomSmallE || opts.right.name == atomE) {
		_, _ = ew.Write([]byte(" "))
	}

	return ew.err
}

// Compare compares the Float with a Term.
func (f Float) Compare(vm *VM, t Term, env *Env) int {
	switch t := env.Resolve(vm, t).(type) {
	case Variable:
		return 1
	case Float:
		return f.dec.Cmp(t.dec)
	default: // Integer, Atom, custom atomic terms, Compound.
		return -1
	}
}

func (f Float) String() string {
	return fmt.Sprintf("%g", f.dec)
}

func (f Float) Negative() bool {
	return f.dec.Sign() < 0
}

func (f Float) Positive() bool {
	return f.dec.Sign() > 0
}

func (f Float) Zero() bool {
	return f.dec.Sign() == 0
}

func (f Float) Eq(other Float) bool {
	return f.dec.Cmp(other.dec) == 0
}

func (f Float) Gt(other Float) bool {
	return f.dec.Cmp(other.dec) == 1
}

func (f Float) Gte(other Float) bool {
	return f.dec.Cmp(other.dec) >= 0
}

func (f Float) Lt(other Float) bool {
	return f.dec.Cmp(other.dec) == -1
}

func (f Float) Lte(other Float) bool {
	return f.dec.Cmp(other.dec) <= 0
}
