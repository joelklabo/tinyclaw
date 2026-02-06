# Scenario Specification

A **scenario** is a YAML file describing a deterministic test case for an agent
run. Scenarios drive the test harness and validate transport behavior.

## YAML Format

```yaml
name: hello-world-streaming        # unique scenario identifier
description: Basic streaming test  # human-readable description

inbound_events:                    # events fed to the transport
  - type: message
    data:
      content: "hi"
      channel_id: "test-channel"
      author_id: "test-user"
    delay: 0                       # optional delay in milliseconds before emit

harness_events:                    # events the harness will emit
  - kind: status
    data:
      status: "thinking"
  - kind: delta
    data:
      text: "Hello"
  - kind: final
    data:
      text: "Hello world!"

expected_transport_ops:            # outbound operations to validate
  - kind: post
  - kind: edit
  - kind: edit

expected_context:                  # optional context expectations
  must_include:
    - name: "readme"               # context item name that must be present

expected_failures:                 # optional expected error conditions
  - kind: auth                     # error kind (auth, quota, transient, fatal)
    message_contains: "token"      # substring match on error message
```

## Field Reference

### Top-Level Fields

| Field                    | Type     | Required | Description                          |
|--------------------------|----------|----------|--------------------------------------|
| name                     | string   | yes      | Unique scenario identifier           |
| description              | string   | no       | Human-readable description           |
| inbound_events           | list     | yes      | Events to feed the transport         |
| harness_events           | list     | yes      | Events the harness produces          |
| expected_transport_ops   | list     | no       | Outbound ops to validate             |
| expected_context         | object   | no       | Context expectations                 |
| expected_failures        | list     | no       | Expected error conditions            |

### InboundEvent

| Field  | Type   | Required | Description                                 |
|--------|--------|----------|---------------------------------------------|
| type   | string | yes      | Event type (e.g., `message`)                |
| data   | object | yes      | Event payload                               |
| delay  | int    | no       | Milliseconds to wait before emitting (0)    |

### HarnessEvent

| Field | Type   | Required | Description                                   |
|-------|--------|----------|-----------------------------------------------|
| kind  | string | yes      | `status`, `delta`, `tool`, `fault`, `final`   |
| data  | object | yes      | Kind-specific payload                         |

### ExpectedTransportOp

| Field | Type   | Required | Description                                   |
|-------|--------|----------|-----------------------------------------------|
| kind  | string | yes      | `post`, `edit`, `upload`, `typing`            |

### ExpectedContext

| Field        | Type | Required | Description                             |
|--------------|------|----------|-----------------------------------------|
| must_include | list | no       | Context items that must be present      |

### ExpectedFailure

| Field            | Type   | Required | Description                          |
|------------------|--------|----------|--------------------------------------|
| kind             | string | yes      | Error kind to expect                 |
| message_contains | string | no       | Substring match on error message     |

## Execution Model

1. The test runner loads the scenario YAML.
2. Inbound events are fed to the transport (respecting `delay`).
3. The harness emits `harness_events` in order.
4. Outbound transport operations are captured and compared against
   `expected_transport_ops` (ordered, kind-matched).
5. If `expected_context` is present, the context manifest is validated.
6. If `expected_failures` is present, the run must produce matching errors.
7. A run bundle is written capturing the full execution for replay.
