# Run Bundle Specification

A **run bundle** is a directory capturing every detail of a single agent run,
enabling deterministic replay and post-mortem debugging.

## Directory Layout

```
bundle-<uuid>/
  run.json            # run metadata
  frames.jsonl        # ordered stream frames (deltas, tool calls, status)
  events.jsonl        # inbound events that triggered the run
  transitions.jsonl   # state-machine transitions
  ctx.json            # context manifest snapshot
  transport.jsonl     # outbound transport operations (posts, edits, uploads)
  logs.jsonl          # structured log lines
  FAIL.md             # present only on failure — human-readable explanation
```

## File Descriptions

### run.json
Top-level metadata for the run.

| Field       | Type   | Description                                |
|-------------|--------|--------------------------------------------|
| id          | string | UUID of the run                            |
| start_time  | string | RFC 3339 timestamp                         |
| scenario    | string | scenario name that initiated the run       |
| status      | string | `pass`, `fail`, or `error`                 |
| end_time    | string | RFC 3339 timestamp (set on finalize)       |

### frames.jsonl
One JSON object per line. Each frame has a `kind` field (`status`, `delta`,
`tool`, `fault`, `final`) and a `data` object containing kind-specific payload.

### events.jsonl
Inbound events consumed during the run. Each line is a JSON object with `type`
and `data` fields.

### transitions.jsonl
State machine transitions. Each line records `from`, `to`, and `timestamp`.

### ctx.json
Snapshot of the context manifest used for this run. A single JSON object with
a `items` array of context items.

### transport.jsonl
Outbound operations sent to the transport. Each line has `kind`
(`post`, `edit`, `upload`, `typing`) and `data`.

### logs.jsonl
Structured log lines. Each line has `level`, `msg`, `ts`, and optional fields.

### FAIL.md
Present only when `run.json` status is `fail` or `error`. Contains a
human-readable markdown explanation of the failure, including error kind,
message, and any relevant context.

## Deterministic Replay Guarantees

1. **Event ordering**: `events.jsonl` captures inbound events in the exact order
   they were received. Replaying these events in order must produce identical
   `frames.jsonl` output.

2. **Timestamp isolation**: Replay harnesses must use recorded timestamps, not
   wall-clock time, so frame ordering is deterministic.

3. **Context pinning**: `ctx.json` captures the exact context used. Replay must
   inject this context rather than re-gathering from the workspace.

4. **Transport recording**: `transport.jsonl` captures what was sent. Replay
   validates outbound operations match without actually sending them.

5. **Idempotent runs**: Given identical `events.jsonl` and `ctx.json`, a replay
   must produce byte-identical `frames.jsonl` and `transport.jsonl`.
