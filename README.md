# tinyclaw

A test-first, debug-bundle-driven system with 100% coverage enforcement.

> **Status: WIP** — Coverage gate is enforced.

## Quick Start

```bash
make test       # unit + contract tests
make system     # offline end-to-end system tests
make coverage   # fails unless coverage = 100.0%
```

## Architecture

See [AGENTS.md](AGENTS.md) for architecture, invariants, and development rules.
