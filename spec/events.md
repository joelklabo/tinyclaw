# RunEvent Types

The harness emits a stream of `RunEvent` values during an agent run.
Each event has a `kind` string and a `data` payload.

## Event Kinds

### status

Indicates a phase change in the agent run.

```json
{"kind": "status", "data": {"status": "thinking"}}
```

| Field  | Type   | Description                                      |
|--------|--------|--------------------------------------------------|
| status | string | One of: thinking, running, tool_use, done, error |

### delta

A streaming text chunk from the agent.

```json
{"kind": "delta", "data": {"text": "Hello, "}}
```

| Field | Type   | Description              |
|-------|--------|--------------------------|
| text  | string | Incremental text output  |

### tool

A tool invocation performed by the agent.

```json
{"kind": "tool", "data": {"name": "bash", "input": {"cmd": "ls"}, "output": "file.txt"}}
```

| Field  | Type   | Description            |
|--------|--------|------------------------|
| name   | string | Tool name              |
| input  | any    | Tool input parameters  |
| output | any    | Tool output result     |

### fault

An error during the agent run.

```json
{"kind": "fault", "data": {"kind": "auth", "message": "invalid token"}}
```

| Field   | Type   | Description                                   |
|---------|--------|-----------------------------------------------|
| kind    | string | Error kind: auth, quota, transient, fatal     |
| message | string | Human-readable error description              |

### final

The final complete response from the agent. Always the last event emitted.

```json
{"kind": "final", "data": {"text": "Here is the complete response..."}}
```

| Field | Type   | Description                   |
|-------|--------|-------------------------------|
| text  | string | Complete final response text  |

## Ordering Guarantees

1. The first event is always `status` with `{"status": "thinking"}`.
2. Zero or more `delta`, `tool`, and `status` events follow in any order.
3. The stream ends with exactly one of:
   - `final` (success)
   - `fault` with kind `fatal` (unrecoverable failure)
4. After the terminal event, no more events are emitted and the channel
   is closed.

## Mapping to Transport Operations

The orchestrator maps harness events to transport operations:

| RunEvent kind | Transport operation              |
|---------------|----------------------------------|
| status        | `ui.typing` (while not done)     |
| delta         | `ui.edit` (streaming update)     |
| tool          | (logged to bundle, not sent)     |
| fault         | `ui.post` (error message)        |
| final         | `ui.post` (final response)       |
