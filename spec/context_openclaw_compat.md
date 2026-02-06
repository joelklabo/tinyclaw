# OpenClaw Compatibility — Context Injection

## Overview

The OpenClaw compatibility layer reads a bootstrap file set from the
`.openclaw` directory in the workspace root and injects their contents into
the context manifest before handing it to the harness.

## Bootstrap File Discovery

1. Walk `.openclaw/` recursively. Only regular files are included.
2. Sort discovered paths lexicographically for stable ordering.
3. For each path, read the file content.

## Truncation

Each file is truncated to `max_chars` characters (default 8192). When
truncation occurs the content is cut at the limit and the marker
`\n[truncated at <max_chars> chars]` is appended.

## Missing-File Markers

If a file listed in `.openclaw/manifest.json` (the optional reference list)
does not exist on disk, the context item is still emitted with content:

```
[missing file: <relative-path>]
```

This ensures the harness sees every expected file, enabling it to request
the file explicitly if needed.

## Context Items

Each bootstrap file produces a `ContextItem`:

| Field    | Value                                     |
|----------|-------------------------------------------|
| name     | Relative path from workspace root         |
| content  | File content (possibly truncated)         |
| source   | `"openclaw"`                              |
| priority | 0 (highest — always included)             |

## `/context list` Output

Returns a JSON array of objects, one per context item:

| Field    | Type   | Description                       |
|----------|--------|-----------------------------------|
| name     | string | Item name (relative path)         |
| source   | string | Provider that produced the item   |
| priority | int    | Priority value                    |
| chars    | int    | Character count of content        |

Items are ordered by (priority ASC, name ASC) for stable output.

## `/context detail <name>` Output

Returns a single JSON object:

| Field    | Type   | Description                       |
|----------|--------|-----------------------------------|
| name     | string | Item name                         |
| source   | string | Provider name                     |
| priority | int    | Priority value                    |
| content  | string | Full content (not truncated)      |

## Stable Ordering Guarantees

1. Bootstrap files are sorted lexicographically by relative path.
2. Context items from multiple providers are merged by
   (priority ASC, name ASC).
3. `Manifest.ToJSON()` produces deterministic output — same inputs always
   yield byte-identical JSON.
