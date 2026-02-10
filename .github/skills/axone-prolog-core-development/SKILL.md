---
name: axone-prolog-core-development
description: Implement and review core changes in Axone's Go-based Prolog VM and interpreter with strict determinism for blockchain embedding. Use when editing predicate semantics, VM execution state, parser or lexer behavior, stream or filesystem I/O (open/*, read_write, set_stream_position), or interpreter wiring in engine/*.go, interpreter.go, and cmd/1pl/*, while preserving ISO compliance where feasible and maintaining explicit project deviations where required by blockchain constraints.
---

# Axone Prolog Core Development

Use this workflow to implement behavior changes without breaking ISO-style error semantics, stream behavior, or deterministic VM state.

## Architecture Context

- Use `references/architecture.md` as canonical VM architecture context for instruction model, register semantics, and design intent.
- Read it before structural VM changes (opcode behavior, registers, execution flow) and before introducing new execution state.

## Blockchain Execution Priority

- Treat this engine as an embedded blockchain VM: deterministic behavior is mandatory.
- Preserve ISO compliance where feasible, but keep project-specific deviations when needed for deterministic or secure execution.
- If a change introduces, removes, or alters a deviation from ISO behavior, update tests and update `README.md` deviations accordingly.
- Prefer existing deterministic patterns used in this codebase (ordered maps, explicit permission checks, controlled stream IDs and reset state) over generic Go shortcuts.

## Workflow

1. Classify the change scope.
2. Edit implementation in the primary file set.
3. Add or update tests before broad refactors.
4. Run focused tests for the changed area.
5. Run full build and test parity commands.
6. Update `README.md` if user-visible behavior changed.

## Classify Change Scope

- Change predicate semantics or exception behavior: edit `engine/builtin.go`; mirror in `engine/builtin_test.go`.
- Change VM execution state, registration, hooks, or reset logic: edit `engine/vm.go`; mirror in `engine/vm_test.go`.
- Change stream internals or file mode handling: edit `engine/stream.go` and related `engine/builtin.go` sections; mirror in `engine/stream_test.go` and `engine/builtin_test.go`.
- Change parsing or tokenization behavior: edit `engine/parser.go` or `engine/lexer.go`; mirror in `engine/parser_test.go` or `engine/lexer_test.go`.
- Change interpreter registration or public behavior: edit `interpreter.go`; mirror in `interpreter_test.go`.
- Change CLI wrapper behavior: edit `cmd/1pl/interpreter.go`; mirror in `cmd/1pl/interpreter_test.go`.

Read `references/change-map.md` for a quick change-to-test map and discovery commands.

## Preserve Core Invariants

- Preserve deterministic behavior; avoid map-order-dependent logic and hidden runtime variability.
- Avoid introducing nondeterministic dependencies (wall clock, random data, unspecified iteration order) into core engine paths.
- Enforce determinism by construction: if behavior depends on external state (time, randomness, host I/O ordering), redesign the API so inputs are explicit and deterministic.
- Reuse existing exception constructors (`typeError`, `domainError`, `permissionError`, `existenceError`, `resourceError`) and match nearby patterns.
- Preserve stream mode permissions:
  - Keep `SetInput` compatible with `read` and `read_write`.
  - Keep `SetOutput` compatible with `write`, `append`, and `read_write`.
- Preserve filesystem capability checks for write-like modes in `open/3` and `open/4` (`OpenFileFS` path).
- Preserve reset behavior in `ResetEnv()`, including stream ID and variable counter reset.

## Test Workflow

1. Run focused tests first by area (see `references/change-map.md`).
2. Run package-level tests for touched packages.
3. Run full CI-parity build and test commands.

Read `references/test-recipes.md` for concrete command sets.

## Completion Checklist

- Add at least one regression test for each behavior fix.
- For determinism-sensitive changes, prove determinism at design level:
  - no hidden entropy in implementation paths;
  - tests assert complete value/order/identity semantics, not partial fields;
  - any non-deterministic source is injected as explicit input and controlled in tests.
- Confirm all touched tests pass locally.
- Confirm full build and race+coverage test command passes.
- Update docs when behavior changes are externally visible, including `README.md` deviations if ISO/constraint boundaries changed.
