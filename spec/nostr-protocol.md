# Nostr AI Conversation Protocol

Nostr event kinds for agent conversations between tinyclaw and clients.

## Event Kinds

| Kind  | Name          | Stored? | Purpose                          |
|-------|---------------|---------|----------------------------------|
| 5800  | ai.prompt     | Yes     | User sends message to agent      |
| 5801  | ai.response   | Yes     | Agent's final complete response  |
| 5802  | ai.tool_call  | Yes     | Tool invocation record           |
| 5803  | ai.error      | Yes     | Agent error/fault                |
| 25800 | ai.status     | No      | State change (ephemeral)         |
| 25801 | ai.delta      | No      | Streaming text fragment          |

## Tag Schema

All events carry:

```
["p", "<recipient-pubkey>"]
["r", "<run-id>"]
["s", "<session-key>"]
```

Response events (5801, 5802, 5803, 25800, 25801) also carry:

```
["e", "<prompt-event-id>", "", "root"]
```

Kind-specific tags:

- **ai.status (25800)**: `["state", "thinking"|"tool_use"|"writing"|"done"]`
- **ai.delta (25801)**: `["seq", "<number>"]`
- **ai.tool_call (5802)**: `["tool", "<name>"]`, `["phase", "start"|"result"]`
- **ai.error (5803)**: `["error_kind", "auth"|"quota"|"transient"|"fatal"]`

## Content (JSON)

- **ai.prompt**: `{"message": "...", "thinking": "low"}`
- **ai.response**: `{"text": "...", "usage": {...}}`
- **ai.status**: `{"state": "thinking"}`
- **ai.delta**: `{"text": "partial", "seq": 1}`
- **ai.tool_call**: `{"name": "bash", "phase": "start", "args": {...}}` or `{"name": "bash", "phase": "result", "output": "..."}`
- **ai.error**: `{"kind": "fatal", "message": "..."}`

## State Machine

```
Tinyclaw RunEvent    ->  Nostr Kind   ->  Butter State
status(thinking)        25800            "thinking"
status(tool_use)        25800            "tool_call"
delta(text)             25801            "writing"
tool(name, args)        5802             tool_call detail
fault(kind, msg)        5803             error
final(text)             5801             "response"
```
