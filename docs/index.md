# tinyclaw

A test-first, debug-bundle-driven agent orchestration system.

## Overview

tinyclaw orchestrates AI agent interactions through a plugin-based architecture with deterministic replay and 100% test coverage.

## Architecture

- **Core State Machine**: Pure, no-IO state transitions (Ingress -> Routed -> ContextBuilt -> Running -> Completed)
- **Plugin System**: Transport, Harness, and Context plugins
- **Debug Bundles**: Every run produces a replayable bundle
- **Protocol**: stdio JSON-RPC line-delimited framing

## Quick Start

See the [README](https://github.com/klabo/tinyclaw) for installation and usage.
