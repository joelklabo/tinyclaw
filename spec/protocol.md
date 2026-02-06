# Protocol Specification

## Transport

tinyclaw uses **stdio** as its transport layer. The supervisor process
launches each plugin as a child process and communicates over the child's
stdin/stdout. Stderr is reserved for human-readable diagnostics.

## Framing

Every message is a single line of JSON terminated by `\n` (0x0A).

- Messages MUST NOT contain embedded newlines within the JSON payload.
- Receivers MUST treat bare `\n` as the frame delimiter.
- Receivers MUST reject any frame that is not valid JSON.
- The empty line (just `\n`) is silently ignored for robustness.

## Message Semantics

Messages follow JSON-RPC 2.0 structure with three forms:

### Request

```json
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{...}}
```

- `id` is a non-null string or integer. The responder echoes it back.
- `method` is a dotted string (e.g., `tools/call`, `resources/read`).

### Response

```json
{"jsonrpc":"2.0","id":1,"result":{...}}
```

Or on error:

```json
{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"..."}}
```

### Notification

```json
{"jsonrpc":"2.0","method":"logging/message","params":{...}}
```

Notifications omit the `id` field and expect no response.

## Initialize Handshake

The first exchange on every connection is the `initialize` request:

1. Supervisor sends `initialize` with `protocolVersion`, `capabilities`,
   and `clientInfo`.
2. Plugin responds with its own `capabilities` and `serverInfo`.
3. Supervisor sends the `initialized` notification.
4. Normal message flow begins.

No other messages may be sent before the handshake completes.

## Tool and Resource Concepts

- **Tools** are callable operations exposed by plugins (e.g., `ui.post`,
  `agent.run`). The supervisor invokes them via `tools/call`.
- **Resources** are readable data sources (e.g., `chat://events`). The
  supervisor reads them via `resources/read`.

Tool and resource schemas are defined in `contracts/`.

## Logging and Progress

Plugins emit structured log messages via the `logging/message`
notification:

```json
{"jsonrpc":"2.0","method":"logging/message","params":{"level":"info","logger":"harness","data":{...}}}
```

Progress on long-running operations is reported via `notifications/progress`:

```json
{"jsonrpc":"2.0","method":"notifications/progress","params":{"progressToken":1,"progress":50,"total":100}}
```

## Error Taxonomy

Errors are classified into four kinds:

| Kind        | Description                              | Retry? |
|-------------|------------------------------------------|--------|
| `auth`      | Authentication or authorization failure  | No     |
| `quota`     | Rate limit or quota exceeded             | After backoff |
| `transient` | Retryable network or service error       | Yes    |
| `fatal`     | Unrecoverable error                      | No     |

JSON-RPC error codes follow the standard ranges:

- `-32700` Parse error
- `-32600` Invalid request
- `-32601` Method not found
- `-32602` Invalid params
- `-32603` Internal error
- `-32000` to `-32099` Server-defined errors (mapped to error taxonomy)
