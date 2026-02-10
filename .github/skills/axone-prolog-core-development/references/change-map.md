# Change Map

Use this map to move from implementation edits to the right tests quickly.
For VM model and design rationale, read `references/architecture.md`.

## Core Mapping

- Builtins, predicate semantics, exception shape:
  - Primary files: `engine/builtin.go`
  - First tests: `go test ./engine -run '^Test(Open|SetInput|SetOutput|SetStreamPosition|ReadTerm|WriteTerm)$'`
  - Note: adapt pattern to touched predicate names.
- VM execution, registration, reset, hooks:
  - Primary files: `engine/vm.go`
  - First tests: `go test ./engine -run '^TestVM_'`
- Stream internals, mode checks, seek/read/write behavior:
  - Primary files: `engine/stream.go`
  - First tests: `go test ./engine -run '^TestStream_'`
- Parser behavior, placeholders, term parsing:
  - Primary files: `engine/parser.go`
  - First tests: `go test ./engine -run '^TestParser_'`
- Lexer/token behavior:
  - Primary files: `engine/lexer.go`
  - First tests: `go test ./engine -run '^TestLexer_'`
- Text loading and consult/dynamic directives:
  - Primary files: `engine/text.go`
  - First tests: `go test ./engine -run '^TestVM_(Compile|Consult)$'`
- Public interpreter API and default builtins wiring:
  - Primary files: `interpreter.go`
  - First tests: `go test . -run '^TestInterpreter_|^TestNew'`
- CLI helper wrapper behavior:
  - Primary files: `cmd/1pl/interpreter.go`
  - First tests: `go test ./cmd/1pl -run '^TestNew$'`

## Determinism Hot Spots

- Ordered term and dict behavior:
  - Files: `engine/dict.go`, call sites in `engine/builtin.go`
  - Risk: unstable key iteration or output ordering
- Stream identity and reset lifecycle:
  - Files: `engine/stream.go`, `engine/vm.go`
  - Risk: non-repeatable stream IDs across runs or after env reset
- Open/read_write integration with host filesystem:
  - Files: `engine/builtin.go` (`OpenFileFS`, `openSourceSink`)
  - Risk: mode-dependent behavior diverging between environments
- Numeric semantics under blockchain constraints:
  - Files: `engine/number.go`, `engine/integer.go`, `engine/float.go`
  - Risk: behavior drift from secure/arbitrary precision handling

## Discovery Commands

Use these commands to find existing behavior patterns before patching:

- `rg -n "typeError\\(|domainError\\(|permissionError\\(|existenceError\\(|resourceError\\(" engine`
- `rg -n "OpenFileFS|openSourceSink|ioModeReadWrite|SetInput|SetOutput|SetStreamPosition" engine`
- `rg -n "^func Test(Open|SetInput|SetOutput|SetStreamPosition|VM_|Stream_|Parser_|Lexer_)" engine/*.go`
- `rg -n "^func Test(New|Interpreter_)" interpreter_test.go cmd/1pl/interpreter_test.go`
- `rg -n "time\\.Now|rand\\.|range .*map\\[" engine interpreter.go cmd/1pl`
