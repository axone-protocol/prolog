package engine

import "unsafe"

// These bounds protect slice lengths and byte sizes before conversion to int.
// They are representability checks, not physical memory limits.
const (
	maxTermCells     = int64(^uint(0)>>1) / int64(unsafe.Sizeof(Term(nil)))
	maxVariableCells = int64(^uint(0)>>1) / int64(unsafe.Sizeof(Variable{}))
)

func termCells(n int64, env *Env) int {
	if n < 0 || n > maxTermCells || n > maxVariableCells {
		panic(meterPanic{exception: resourceError(resourceMemory, env)})
	}
	return int(n)
}

func addTermCells(a, b int64, env *Env) int64 {
	termCells(a, env)
	termCells(b, env)
	if a > maxTermCells-b || a > maxVariableCells-b {
		panic(meterPanic{exception: resourceError(resourceMemory, env)})
	}
	return a + b
}

func chargeTermCells(n int64, env *Env) {
	termCells(n, env)
	env.charge(MeterTermCell, uint64(n))
}

// makeTerms reserves logical term slots before requesting their backing array.
// The host meter supplies the quota; no installed meter means no finite quota.
func makeTerms(n int64, env *Env) []Term {
	chargeTermCells(n, env)
	return make([]Term, int(n))
}

// MakeTerms reserves n logical term slots using the VM's installed meter before
// allocating a buffer for host-built terms. A negative or unrepresentable size
// returns a resource_error(memory) exception, as does a memory quota refusal.
func (vm *VM) MakeTerms(n int) (terms []Term, err error) {
	defer recoverMeterError(&err)
	return makeTerms(int64(n), vm.NewEnv()), nil
}

// appendTerms charges new logical slots, not Go's implicit slice overcapacity.
// Fill already charged reservations with ordinary append instead.
func appendTerms(ts []Term, env *Env, terms ...Term) []Term {
	addTermCells(int64(len(ts)), int64(len(terms)), env)
	chargeTermCells(int64(len(terms)), env)
	return append(ts, terms...)
}
