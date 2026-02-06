# tinyclaw

[![CI](https://github.com/klabo/tinyclaw/actions/workflows/ci.yml/badge.svg)](https://github.com/klabo/tinyclaw/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/coverage-100%25-brightgreen)](https://github.com/klabo/tinyclaw)
[![Go Report Card](https://goreportcard.com/badge/github.com/klabo/tinyclaw)](https://goreportcard.com/report/github.com/klabo/tinyclaw)
[![Docs](https://img.shields.io/badge/docs-pages-blue)](https://klabo.github.io/tinyclaw)

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
