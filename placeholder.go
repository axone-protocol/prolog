package prolog

import (
	"fmt"
	"reflect"

	"github.com/axone-protocol/prolog/v3/engine"
)

func (i *Interpreter) placeholderTerms(args []interface{}) ([]engine.Term, error) {
	if len(args) == 0 {
		return nil, nil
	}

	terms, err := i.MakeTerms(len(args))
	if err != nil {
		return nil, err
	}
	for n, arg := range args {
		terms[n], err = i.placeholderTerm(reflect.ValueOf(arg))
		if err != nil {
			return nil, err
		}
	}
	return terms, nil
}

func (i *Interpreter) placeholderTerm(value reflect.Value) (engine.Term, error) {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return engine.Integer(value.Int()), nil
	case reflect.String:
		return i.DoubleQuoted(value.String()), nil
	case reflect.Array, reflect.Slice:
		terms, err := i.MakeTerms(value.Len())
		if err != nil {
			return nil, err
		}
		for n := range terms {
			terms[n], err = i.placeholderTerm(value.Index(n))
			if err != nil {
				return nil, err
			}
		}
		return engine.List(terms...), nil
	default:
		return nil, fmt.Errorf("unsupported placeholder type %s", value.Kind())
	}
}
