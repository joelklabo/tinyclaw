# AGENTS.md — tinyclaw project brain

## Architecture

tinyclaw is a modular agent orchestration system built around:
- **Core state machine** (`internal/core/`): pure, no IO — Ingress → Routed → ContextBuilt → Running → Completed
- **Protocol codec** (`internal/protocol/`): stdio JSON-RPC line-delimited framing
- **Debug bundles** (`internal/bundles/`): every system test run writes a replayable bundle
- **Plugin system** (`internal/plugin/`): transport, harness, and context plugins
- **Router** (`internal/router/`): deterministic routing with delegation loop prevention
- **Context system** (`internal/context/`): strategy-based context building with OpenClaw compat

## Invariants

1. **Tests first** — No implementation work begins until failing tests exist.
2. **One failing test at a time** — Fix one, move on.
3. **Debug bundles require zero human intervention** — Every system test produces a bundle. On failure, `FAIL.md` explains what failed and how to replay.
4. **100% coverage enforced** — CI fails if coverage drops below 100% for all non-`cmd/` packages.
5. **No logic in `cmd/`** — All logic in `internal/*` and `plugins/*`.

## Stack

- Go (latest stable)
- JSON-RPC over stdio
- JSON Schema (draft 2020-12) for contracts
- YAML for scenario definitions
- zap for structured logging
- OpenTelemetry for instrumentation

## Key Commands

- `make test` — unit + contract tests with `-race`
- `make system` — offline system scenarios
- `make coverage` — 100% coverage gate
- `make fmt` — format code
- `make lint` — lint code
- `tinyclaw test` — run scenarios, write bundles
- `tinyclaw replay --bundle <dir>` — replay deterministically

## Bundle Layout

```
bundle-<id>/
  run.json           # run metadata
  frames.jsonl       # protocol frames
  events.jsonl       # run events
  transitions.jsonl  # state transitions
  ctx.json           # context manifest
  transport.jsonl    # transport operations
  logs.jsonl         # structured logs
  FAIL.md            # failure report (on failure only)
```

## Error Taxonomy

- `AuthError` — authentication/authorization failures
- `QuotaError` — rate limit / quota exceeded
- `TransientError` — retryable network/service errors
- `FatalError` — unrecoverable errors

## Backlog Discipline

When you notice something, add a bullet to `BACKLOG.md` and keep going. Never scope-creep.
