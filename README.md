# tinyclaw

[![CI](https://github.com/klabo/tinyclaw/actions/workflows/ci.yml/badge.svg)](https://github.com/klabo/tinyclaw/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/klabo/tinyclaw)](https://goreportcard.com/report/github.com/klabo/tinyclaw)

A minimal Discord bot that pipes messages through Claude Code. Point it at a channel, a working directory, and a `DISCORD_TOKEN` — it handles the rest.

Built for [OpenClaw](https://github.com/klabo/openclaw) users who want a lightweight alternative to the full gateway for single-channel Claude Code automation.

## Prerequisites

- Go 1.25+
- [Claude Code](https://claude.ai/claude-code) CLI installed and authenticated (`claude` in your PATH)
- A Discord bot token with `GuildMessages` and `MessageContent` intents enabled
- (Optional) An `.openclaw/` directory in your working directory for bootstrap context

## Install

```bash
go install github.com/klabo/tinyclaw/cmd/tinyclaw@latest
```

Or build from source:

```bash
git clone https://github.com/klabo/tinyclaw.git
cd tinyclaw
make build
```

## Quick Start

### 1. Get your Discord bot token

If you're an existing OpenClaw user, your token is in `~/.openclaw/openclaw.json` under `channels.discord.token`. Otherwise, create a bot at [discord.com/developers](https://discord.com/developers/applications) and enable the `MESSAGE CONTENT` privileged intent.

### 2. Get your channel ID

Right-click the Discord channel > Copy Channel ID (requires Developer Mode in Discord settings). OpenClaw users can find channel IDs in their `openclaw.json` under `channels.discord.guilds`.

### 3. Run

```bash
export DISCORD_TOKEN=your-bot-token
tinyclaw run --channel 1234567890 --workdir ~/myproject
```

That's it. The bot will:
- Connect to Discord and listen for messages in the specified channel
- Spawn a fresh `claude` subprocess for each incoming message
- Read `.openclaw/` context files from the working directory (if present)
- Post Claude's response back to Discord
- Save a bundle (JSONL frames + metadata) for every run in `./bundles/`

### Options

```
tinyclaw run [flags]

  --channel <id>    Discord channel ID to listen on (required, repeatable)
  --workdir <dir>   Working directory for Claude Code (default: .)
  --config <file>   Config file path (optional)
```

Environment variables:
- `DISCORD_TOKEN` — Bot token (required, never passed as a flag)
- `TINYCLAW_BUNDLE_DIR` — Override bundle output directory (default: `bundles`)
- `TINYCLAW_LOG_LEVEL` — Log level: debug, info, warn, error (default: `info`)

### Config file

Optional YAML config (passed via `--config`):

```yaml
log_level: debug
bundle_dir: /var/log/tinyclaw/bundles
```

## How It Works

```
Discord message
  -> discordgo WebSocket
  -> Transport.Subscribe (channel filter)
  -> Orchestrator.route (channel -> profile)
  -> openclaw.Gather (reads .openclaw/ context)
  -> claude --output-format stream-json --verbose --print -p <message>
  -> stream events parsed (status, delta, tool, result)
  -> Transport.Post (final answer -> Discord)
  -> Bundle saved (run.json + frames.jsonl + phases.jsonl)
```

Each message gets its own goroutine, harness process, and bundle. Up to 5 messages are processed concurrently. Graceful shutdown on SIGINT/SIGTERM drains in-flight runs before exiting.

## OpenClaw Context

If your working directory has an `.openclaw/` folder, tinyclaw reads it and injects the files as context items into each Claude Code run. This is the same context format used by the OpenClaw gateway — no migration needed.

## Other Commands

```bash
tinyclaw test scenarios/basic.yaml   # Run a scenario test
tinyclaw replay --bundle ./bundles/bundle-live-1  # Replay/validate a bundle
tinyclaw version                     # Print version
```

## Development

```bash
make test       # Unit tests with race detector
make coverage   # Fails unless coverage >= 95%
make fmt        # Format code
make lint       # go vet
make build      # Build binary
```

## Architecture

9 packages, strict boundaries:

| Package | Role |
|---------|------|
| `internal/plugin` | Interfaces: Transport, Harness, data types |
| `internal/orchestrator` | Routes events, runs harness pipeline |
| `internal/bundle` | JSONL writer + loader/validator |
| `internal/scenario` | YAML test scenarios + assertions |
| `internal/cli` | Parsing, config, serve loop, wiring |
| `plugins/discord` | Discord transport + chunker + discordgo adapter |
| `plugins/claudecode` | Claude Code harness + exec runner |
| `plugins/openclaw` | Context reader for `.openclaw/` directories |
| `cmd/tinyclaw` | Entry point |

## License

MIT
