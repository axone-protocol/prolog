package engine

// MeterKind identifies a metered execution resource exposed by the VM.
type MeterKind uint8

const (
	// MeterInstruction charges one unit per VM opcode dispatched by the execution loop.
	// This is the baseline cost of opcode-visible Prolog control flow.
	MeterInstruction MeterKind = iota

	// MeterUnifyStep charges one unit per semantic unification step, including recursive descent,
	// variable binding, and occurs-check traversal when applicable.
	MeterUnifyStep

	// MeterListCell charges one unit per list cell consumed by a VM iterator.
	// This covers list-processing builtins implemented in Go that would otherwise do free linear work.
	MeterListCell

	// MeterCopyNode charges one unit per variable or structural term node materially copied by the VM.
	// It covers operations such as copy_term/2 and the internal copying performed by findall/3.
	MeterCopyNode

	// MeterArithNode charges one unit per arithmetic expression node evaluated by the VM.
	// It includes operators, constants, and numeric literals traversed by arithmetic evaluation.
	MeterArithNode

	// MeterCompareStep charges one unit per structural comparison step in the standard order of terms.
	// It captures work performed by compare/3, sort/2, keysort/2, setof/3, and related operations.
	MeterCompareStep
)

// MeterFunc is called by the VM whenever it consumes a metered resource.
// Returning nil continues execution.
// Returning a non-nil term aborts execution by throwing error(Formal, Context),
// where the returned term is used as Formal and the VM supplies the current Context.
type MeterFunc func(kind MeterKind, units uint64) Term

type meterPanic struct {
	exception Exception
}

func chargeMeter(m MeterFunc, kind MeterKind, units uint64, env *Env) {
	if m == nil || units == 0 {
		return
	}
	if formal := m(kind, units); formal != nil {
		env = env.withoutMeter()
		exception := NewException(atomError.Apply(formal, env.Resolve(varContext)), env)
		panic(meterPanic{exception})
	}
}
