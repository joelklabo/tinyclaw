# Plugin Interfaces

tinyclaw defines three plugin types. Each runs as a separate process
communicating over stdio JSON-RPC.

## Transport

A Transport plugin bridges an external messaging platform (Discord, Slack,
IRC, etc.) to tinyclaw.

**Inbound**: The transport receives events from the platform and exposes
them as resources. The supervisor reads `chat://events` to get new messages.

**Outbound**: The supervisor calls transport tools to perform operations:

- `ui.post` — send a message to a channel
- `ui.edit` — edit an existing message
- `ui.upload` — upload a file attachment
- `ui.typing` — indicate typing status

## Harness

A Harness plugin runs an agent (Claude, GPT, local model, replay, etc.)
and emits a stream of RunEvents.

**Input**: The supervisor calls `agent.run` with a context manifest and
conversation history.

**Output**: The harness emits RunEvents as logging notifications:

| Event         | Description                                |
|---------------|--------------------------------------------|
| `agent.phase` | Status change (thinking, tool_use, done)   |
| `agent.delta` | Incremental text output                    |
| `agent.tool`  | Tool call request or result                |
| `agent.fault` | Error during agent execution               |
| `agent.delegate` | Delegation to another agent             |

The final response is returned as the `agent.run` tool result.

## Context

A Context plugin builds a context manifest from workspace files. The
supervisor calls it before handing context to the harness.

**Input**: Workspace root path and configuration.

**Output**: A context manifest containing:

- Relevant file contents
- System prompt fragments
- Tool definitions available to the agent
- Resource URIs for reference data
