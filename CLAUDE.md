# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Go OwO Discord **selfbot** (user token, not a bot token) that automates hunting, battling,
checklist, inventory, gems, gambling, quests, and HuntBot. One process runs one goroutine per
token, so several accounts share the binary. See `README.md` for the user-facing feature and
config reference, and `COMPARISON.md` for how the feature set compares to owo-dusk.

## Commands

```bash
go run ./cmd/sempatowo                  # run (needs TOKEN in .env)
go run ./cmd/sempatowo -check-config    # validate env + every config file, no Discord connection
go run ./cmd/sempatowo -simulate-captcha # connect and inject a fake captcha to test pause/browser/notify
go build -o sempatowo ./cmd/sempatowo

go test ./...
go test -race ./...                     # what CI runs; concurrency is the main risk area here
go test ./internal/farm -run TestSuperviseLoop -race   # single test
go vet ./...
gofmt -l ./internal ./cmd               # CI fails on any output; deps/ is deliberately excluded
go generate ./internal/config           # regenerate config.schema.json after changing config structs
```

CI (`.github/workflows/ci.yml`) also checks `go mod tidy` leaves `go.mod`/`go.sum` unchanged and
cross-builds for linux/windows/darwin.

## Architecture

### Ownership: `farm.Bot` is the hub, subsystems talk back through interfaces

`internal/farm.Bot` owns the Discord client, config loader, message routing, and the outgoing
queue. Each larger feature lives in its own package (`huntbot`, `gamble`, `quest`, `daily`) and
depends on the bot only through a small `BotContext` interface it declares itself
(`internal/*/types.go`). The implementation of each interface is a thin adapter file in
`internal/farm/` named after the package it serves — `huntbot.go`, `gamble.go`, `daily.go`,
`autoquest.go`. **Adding a capability a subsystem needs means adding a method to its `BotContext`
and implementing it in the matching adapter**, not importing `farm` from the subsystem.

`internal/config` is a leaf package: it declares its own `Logger` interface rather than importing
`internal/util`.

### Everything is driven by OwO's replies, not by wall-clock timers alone

`onMessage`/`onMessageUpdate` in `farm.go` are the single entry point for OwO text. Order there is
load-bearing: verification success and captcha detection run *before* the "is this addressed to
me" check, because a captcha must stop the farm immediately.

The farm scheduler (`scheduler.go`, `sched_state.go`) is a heap of due commands, not one timer per
command. After a command is sent it is marked *awaiting*; the next run is scheduled only when the
matching reply is recognised (`farm_response.go`) or a 30s timeout fires. `ClaimAwaiting` is the
race resolver — exactly one of reply-vs-timeout wins and reschedules. An empty heap means
everything is in flight, not that work is done, so the loop blocks on `wake` rather than exiting.

Command definitions live in one table, `farmCommands` in `scheduler.go`. Adding a scheduled OwO
command means adding a row there plus a recogniser in `farmCmdsFromMessage`.

### Concurrency shape

`Bot` deliberately does **not** funnel everything through one mutex. Independent state components
own their own locks: `sender` (outgoing queue), `sched` (`farmSchedState`), `stats` (`farmStats`),
`captchaTimers`, `sleeper`. `b.mu` guards only the bot's own flags (`active`, `ready`,
`captchaSolving`, `checklistAwaiting`, `questOwo`, `timerCancel`).

Lock discipline, and the reason several methods have `…Locked` twins: **`b.mu` may be held while
taking a component lock, so a component must never reach back into the bot while holding its own.**
`sender.mu` is a leaf; its callbacks (`canSend`, `deliver`) are always invoked unlocked. Teardown
(`teardownSession`, `handleCaptcha`) follows this order — mutate under `b.mu`, release, then stop
the components.

The config pointer is `atomic.Pointer[config.Loader]` because `onReady` installs it on the gateway
goroutine while handlers and timers read it. Read settings via `b.settings()`, which returns
`config.Defaults()` when nothing is loaded yet.

Every goroutine spawned from bot code goes through `util.Go` / `util.Recover`: handlers parse
untrusted OwO text on library goroutines, and a panic there would otherwise kill every account
sharing the process.

### Connection supervision

`Bot.Run` is a supervisor loop. The gateway library resumes on its own but gives up permanently
after a bounded number of attempts, leaving a process that looks healthy and does nothing.
`superviseLoop` polls `IsConnected` and, after `connectionGrace` (90s — must stay above the
library's own 1+2+4+8+16s ladder) of continuous downtime, returns so `Run` tears the session down
and rebuilds the client with backoff. A failure on the *first* attempt returns instead of retrying,
since that is almost always a bad token.

`onReady` fires again on every reconnect. It reuses the existing `config.Loader` rather than
building a new one, because `NewLoader` starts an unstoppable file-watcher goroutine.

### Config: schema, hot-reload, migrations

One JSON file per account at `var/config/{discord_user_id}.json` (keyed by ID, not username;
`label` inside carries the human name). `internal/config`:

- `config.go` — the `Settings` structs. Field order here *is* the order the file is written in.
  `Discord` and `Humanize` are embedded with a JSON tag, so they nest in the file while Go still
  promotes `s.Prefix`, `s.OwoBotID`, etc.
- `loader.go` — load, validate, and hot-reload via fsnotify. Defaults merge is just
  `json.Unmarshal` into an already-populated `Defaults()`. A reload that fails to parse or validate
  is **rejected with the previous settings left in force**. Unknown keys are reported as probable
  typos rather than silently ignored.
- `duration.go` / `Range` — durations are strings (`"5m"`, `"1m30s"`); a bare number is rejected.
  Every wait is a range: `"5m"` or `{"min": …, "max": …}`, both accepted everywhere.
- `format.go` — the readable JSON writer; anything that fits on a line stays on a line.
- `legacy.go` (pre-1.0 → v1) and `legacy_v1.go` (v1 → v2) — migrations run on load, keeping the
  original as `*.json.v{N}.bak`. Bumping `SchemaVersion` means adding a migration and a case in
  `loadFromFile`.
- `validate.go` — errors block loading; `Warnings()` are advisory (e.g. gems on with inventory
  off). A warning must name a real interaction: `hunt` and `huntbot` are independent OwO features
  that only share the send queue, so running both is not one.

`config.schema.json` is generated from the struct doc comments by `internal/config/schemagen`,
checked in, embedded, and written next to the user's configs on every start.
`TestSchemaFileIsUpToDate` fails if you edit config structs without running `go generate`.

`onConfigChange` restarts only the subsystems whose settings actually moved — a blanket restart
used to reset in-flight delays and lose gamble martingale state on any one-character edit. New
config sections need a corresponding comparison there.

### Environment and writable state

All env vars are read and validated once in `config.LoadEnv()` at startup, before anything
connects, so a bad `CAPTCHA_SERVICE` fails immediately rather than the first time a captcha
appears. Credentials (`TOKEN`, `OCR_API_KEY`, `CAPTCHA_API_KEY`) live in the environment, never in
the config file — the pre-1.0 migration exists partly to move `ocrApi` out.

All writable state hangs off one root (`DATA_DIR`, default `./var`): `config/`, `data/`,
`browser-profiles/`. `var/` is gitignored.

### Captcha handling

Detection (`captcha_detect.go`) pauses everything, opens a browser (`internal/captcha`), fires a
desktop notification (`internal/notify`, per-OS build-tagged files), and optionally runs an
auto-solver against capsolver/capmonster/2captcha. Countdown warnings fire at 8/5/2/1 minutes
against a 10-minute deadline. When browsers are not isolated per account, `browser_queue.go`
serialises accounts through the one shared system browser — `AcquireBrowserSlot` /
`ReleaseBrowserSlot` must stay paired, including on the deadline path.

## Conventions

- `deps/discordgo-self` is vendored third-party source wired in with a `replace` directive. It is
  excluded from gofmt checks so its diff against upstream stays readable, and it is invisible to
  `go get -u` and `govulncheck`. Record any local patch in the table in `deps/README.md`.
- Comments here explain *why*, usually citing the bug the code prevents. Match that: a comment
  restating the code is worse than none, but a non-obvious ordering, lock rule, or regex quirk
  should say what breaks without it.
- Regexes that parse OwO text carry a note about the case that motivated them (e.g. `[\d,]+`
  because OwO comma-groups from 1,000 up). Keep that when touching them.
- Prefer nil-safe accessors (`internal/farm/safe.go`) over reaching into `b.client` directly.

## Commits

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Existing history predates this convention** — do not use `git log` as a style reference. Apply
the format to new commits only; do not rewrite old ones.

### Type

| Type | Use for |
| ---- | ------- |
| `feat` | A new automated behaviour, config setting, or CLI flag |
| `fix` | Corrected behaviour — a race, a missed OwO reply, a bad regex match |
| `refactor` | Restructuring with no behaviour change (e.g. splitting state into components) |
| `perf` | Measurably faster or lighter |
| `test` | Tests only |
| `docs` | `README.md`, `COMPARISON.md`, `deps/README.md`, `CLAUDE.md`, doc comments |
| `build` | `go.mod`/`go.sum`, the vendored `deps/discordgo-self`, build tags |
| `ci` | `.github/workflows/` |
| `chore` | Anything left over — nothing in it changes for a user |

### Scope

Optional, but use it when the change is confined to one area. Prefer the package name:
`farm`, `config`, `huntbot`, `gamble`, `quest`, `daily`, `captcha`, `notify`, `util`, `items`,
`schemagen`, `deps`. Use `cmd` for the entry points. Omit it for a change that genuinely spans the
codebase (e.g. a gofmt sweep). A scope for the config *file format* rather than the package is
fine: `config-schema`.

### Subject

Imperative mood, lowercase, no trailing period, aim for ≤ 72 characters:
`fix(farm): reschedule hunt when the reply arrives during teardown`, not
`fix(farm): Fixed the scheduler.`

### Body

Optional, and worth writing whenever the change is not self-evident. Match the codebase's comment
style: say *why*, and name the failure the change prevents — "a blanket restart reset every
in-flight delay" is the useful sentence, not "changed onConfigChange". Wrap at 72 columns.

### Footer

- Breaking changes: `BREAKING CHANGE: <what breaks and what to do>` in the footer, or a `!` after
  the type/scope (`feat(config)!: …`). For this project that mostly means a `SchemaVersion` bump —
  say which migration handles it.
- Reference issues with `Refs #12` / `Closes #12`.
- End every commit message you author with:
  `Co-Authored-By: Claude Opus 4.8 <noreply@anthropic.com>`

### Examples

```
fix(farm): stop the gateway supervisor from fighting the library's resume

connectionGrace has to exceed the library's own 1+2+4+8+16s backoff ladder;
below that we tore down a session that was about to resume on its own and
restarted every timer for nothing.
```

```
feat(config)!: group the top level and write the file readably

prefix, defaultChannel and owoBotId move under "discord"; typing and
sendMessageInterval under "humanize". Embedded-with-tag structs keep
s.Prefix working, so no call site changed.

BREAKING CHANGE: schemaVersion 2. migrateV1 converts v1 files on load and
keeps the original as *.json.v1.bak.
```

### Practice

- Commit or push only when asked. Branches use `<kind>/<short-description>`
  (e.g. `hardening/ci-fmt-and-concurrency`); if on `master`, branch first.
- Before committing, run at minimum `gofmt -l ./internal ./cmd`, `go vet ./...`, and
  `go test -race ./...` — the same gates CI applies. If config structs changed, run
  `go generate ./internal/config` and commit the regenerated `config.schema.json` in the same
  commit as the struct change.
- One logical change per commit. A behaviour fix and the refactor that made room for it are two
  commits.
- Never commit `.env`, `var/`, or anything under it.
