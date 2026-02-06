# Contracts

This directory contains **normative JSON Schemas** (draft 2020-12) that
define the data structures exchanged between tinyclaw components.

## Layout

```
contracts/
  tools/
    ui.tools.json        — ui.post, ui.edit, ui.upload tool outputs
    agent.tools.json     — agent.run task output
  resources/
    chat.events.json     — inbound chat event payloads
  logging/
    agent.logging.json   — harness RunEvent log payloads
```

## Rules

1. Every schema MUST validate as JSON Schema draft 2020-12.
2. Changes to any schema require a corresponding update to the spec
   documents in `spec/` and to all affected tests.
3. Schemas are tested automatically by `internal/contracts/validate_test.go`
   which loads every `contracts/**/*.json` and compiles it.
4. Breaking changes require a protocol version bump in the initialize
   handshake.
