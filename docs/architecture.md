# Architecture

## Plugin Types

### Transport
Receives inbound events from messaging platforms and sends outbound operations.

### Harness
Runs an AI agent and streams normalized events (status, delta, tool, fault, final).

### Context
Builds context manifests from workspace files for injection into agent prompts.

## State Machine

```
Ingress -> Routed -> ContextBuilt -> Running -> Completed
                                             -> Failed
```

## Debug Bundles

Every system test run produces a bundle containing:
- `run.json` -- run metadata
- `frames.jsonl` -- protocol frames
- `events.jsonl` -- run events
- `transitions.jsonl` -- state transitions
- `ctx.json` -- context manifest
- `transport.jsonl` -- transport operations
- `logs.jsonl` -- structured logs
- `FAIL.md` -- failure report (on failure only)
