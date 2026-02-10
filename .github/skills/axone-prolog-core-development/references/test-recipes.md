# Test Recipes

Run commands from repository root.

## Fast Path

Run the most relevant tests first, then widen:

1. Targeted pattern:
   - `go test ./engine -run '<pattern>'`
2. Package-level:
   - `go test ./engine`
   - `go test ./cmd/1pl`
   - `go test .`
3. Full parity:
   - `go build -v ./...`
   - `go test -v -race -covermode=atomic -coverpkg=./... -coverprofile=coverage.txt ./...`

## Determinism Validation

Determinism is not established by repetition counts. Validate by construction:

1. Reject hidden entropy in implementation paths:
   - `rg -n "time\\.Now|time\\.Since|time\\.Unix|rand\\.|range .*map\\[" engine interpreter.go cmd/1pl`
2. Ensure tests assert full deterministic semantics (exact output/order/identity), not partial fields.
3. Ensure any unavoidable external input is explicit and injected so tests control it completely.

## Common Patterns

- Stream and open mode changes:
  - `go test ./engine -run '^Test(Open|SetInput|SetOutput|SetStreamPosition|Stream_)'`
- VM reset or execution changes:
  - `go test ./engine -run '^TestVM_'`
- Parser changes:
  - `go test ./engine -run '^TestParser_'`
- Lexer changes:
  - `go test ./engine -run '^TestLexer_'`
- Public interpreter behavior:
  - `go test . -run '^TestInterpreter_|^TestNew'`

## Useful Flags

- Add `-count=1` to bypass cached results when debugging.
- Add `-v` to inspect failing test names and subtests.
- Keep `-race` for full runs; skip it only for fast local feedback.
