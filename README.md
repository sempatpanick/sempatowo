# sempatowo

Go rewrite of [owofarm](https://github.com/) — an OwO Discord selfbot that automates hunting, battling, checklist, inventory, gems, and HuntBot.

> **Warning:** Selfbots violate Discord's Terms of Service. Use at your own risk.

## Requirements

- Go 1.25+
- A Discord **user** token (not a bot token)
- Chrome (optional, for captcha solving)

## Quick start

```bash
# 1. Copy env and add your token
cp .env.example .env

# 2. Check the environment before connecting anything
go run ./cmd/sempatowo -check-config

# 3. Run (creates var/config/{your_user_id}.json on first login)
go run ./cmd/sempatowo
```

Or build a binary:

```bash
go build -o sempatowo ./cmd/sempatowo
```

## Project layout

```
cmd/sempatowo/     Entry point — loads .env, starts one bot per token
  main.go            wiring
  checkconfig.go     -check-config: validate everything without connecting
cmd/gen-config-schema/  go:generate target for internal/config/config.schema.json
internal/
  config/          Schema, defaults, validation, hot-reload, migrations, env
    format.go        the readable JSON writer used for config files
    schemagen/       builds config.schema.json from the structs' doc comments
  farm/            Main bot: connection supervisor, message routing, subsystem adapters
    sender.go        outgoing message queue
    sched_state.go   farm command heap
    stats.go         inventory, checklist, counters
    huntbot.go       adapter to internal/huntbot
    gamble.go        adapter to internal/gamble
    daily.go         adapter to internal/daily
    autoquest.go     adapter to internal/quest
  huntbot/         HuntBot autohunt + essence upgrades + password captcha
  gamble/          Coinflip, slots, blackjack
  quest/           Quest parsing (OCR) and execution
  daily/           Standalone daily claim at PST midnight
  captcha/         OwO captcha browser + optional auto-solver
  notify/          Desktop notifications (per-OS)
  items/           Inventory item IDs (gems, crates, lootboxes)
  util/            Logger, panic recovery, helpers
var/               All writable state (gitignored) — override with DATA_DIR
  config/            per-account JSON, named after the Discord user ID,
                     plus the generated config.schema.json they point at
  data/              daily-claim timestamps
  browser-profiles/  isolated Chrome profiles for captcha solving
deps/              Vendored discordgo-self fork — see deps/README.md
```

## Configuration

One JSON file per account at `var/config/{user_id}.json`, created from defaults
on first login. Edit it while the bot runs — changes reload automatically, and a
file that fails validation is rejected with the previous settings left in force.

### Shape

The top level is grouped by what a setting is about, and the rest is one block
per feature, so everything needed to run a feature is in one place. This is the
file as it is actually written — anything that fits on a line stays on a line:

```json
{
  "$schema": "config.schema.json",
  "schemaVersion": 2,
  "label": "sempatpanick",
  "trackBalance": true,
  "discord": {
    "prefix": "w",
    "defaultChannel": "1513744333579489310",
    "owoBotId": "408785106942164992"
  },
  "humanize": { "typing": true, "sendMessageInterval": "5s" },
  "features": {
    "hunt": { "enabled": true, "delay": { "min": "50s", "max": "3m20s" } },
    "battle": { "enabled": true, "delay": { "min": "50s", "max": "3m20s" } },
    "pray": { "enabled": true, "delay": "5m5s", "target": "" },
    "checklist": { "enabled": false, "delay": "16m40s", "stopFarmingWhenDone": false },
    "cookie": { "enabled": false, "target": "469369739131617291" },
    "lootbox": { "enabled": true, "fabled": true },
    "crate": { "enabled": true },
    "quest": {
      "enabled": false,
      "delay": "1m",
      "owoDelay": "32s",
      "auto": { "enabled": false, "acknowledgeExperimental": false }
    },
    "huntbot": { "enabled": false, "cashToSpend": 10000 },
    "gamble": { "allottedAmount": 10000 }
  }
}
```

| Top-level block | What is in it |
| --------------- | ------------- |
| `discord` | `prefix`, `defaultChannel`, `owoBotId` — how commands are addressed |
| `humanize` | `typing`, `sendMessageInterval` — knobs whose only job is not looking like a program |
| `trackBalance` | Not grouped: the running cash total the gamble limits and the daily handler both depend on |
| `features` | One block per automated behaviour |

`label` is for humans reading the directory; the filename is the user ID, which
does not change when the username does.

### Three rules worth knowing

**Durations are strings.** `"15s"`, `"5m"`, `"1m30s"` — anything Go's
`time.ParseDuration` accepts. A bare number is rejected rather than guessed at.

**Every wait is a range, written one of two ways.** `"delay": "5m"` is a fixed
wait; `"delay": { "min": "50s", "max": "3m20s" }` jitters between the two. Both
forms are accepted everywhere, and a fixed one is rewritten to the short form.
Jitter is available on every scheduled command, not just hunt and battle — a
perfectly periodic command is the easiest kind of automation to spot.

**A feature's channel falls back to `defaultChannel`.** Set `"channel"` inside a
scheduled block only when it should go somewhere else.

### Editor support

The `"$schema"` key points at `config.schema.json`, written next to your configs
on every start. VS Code and JetBrains pick it up with no setup: hovering a key
shows its documentation, unknown keys are underlined, and autocomplete offers the
settings that exist. It is generated from the Go source — after changing a
setting, run:

```bash
go generate ./internal/config
```

### Features

| Block | Description |
| ----- | ----------- |
| `hunt`, `battle` | The core loops |
| `pray`, `curse` | Scheduled, with an optional `target` (empty = yourself) |
| `zoo`, `inventory`, `checklist` | Periodic status commands; `checklist.stopFarmingWhenDone` halts farming once every item is ticked |
| `cookie` | Sent from the checklist reply, to `target` |
| `lootbox` (`fabled`), `crate`, `gems` | Opened/used as inventory reports them |
| `daily` | Standalone daily claim at PST midnight reset |
| `quest` | Quest log checks; `quest.auto` is the experimental full auto-quest |
| `huntbot` | HuntBot autohunt with essence upgrades; independent of `hunt`, both can run |
| `gamble` | Coinflip, slots, blackjack, with `allottedAmount` and `goalSystem` limits |

Every scheduled block takes `enabled`, `delay`, and an optional `channel` that
overrides `discord.defaultChannel`.

### Validation

`-check-config` validates the environment and every config file without
connecting to Discord:

```bash
go run ./cmd/sempatowo -check-config
```

It reports errors — an enabled feature with no channel, an inverted delay range,
a coinflip with neither side selected, gambling with `trackBalance` off — and
warnings, like auto-quest enabled but not acknowledged, or gems switched on with
inventory checks off.

### Upgrading from an older format

Any config older than the current `schemaVersion` is migrated automatically on
first start, and the original is kept alongside it as `*.json.v{N}.bak`, named
after the version it came from. A file named after your username is also renamed
to your user ID.

**Pre-1.0 → v1** flattened `status`/`interval`/`channels`/`target` into one block
per feature and made every duration a string with its unit. The one thing that
does not migrate is `ocrApi` — it is a credential, so it moved to the environment
as `OCR_API_KEY`; the migration prints the value it found so you can paste it
into `.env`.

**v1 → v2** grouped the top level. `prefix`, `defaultChannel` and `owoBotId`
moved under `discord`; `typing` and `sendMessageInterval` under `humanize`; and
`stopWhenChecklistDone` became `features.checklist.stopFarmingWhenDone`. Nothing
else changed, and no value is lost.

## Environment variables

Everything is read and validated once at startup, so a bad value fails before
the bot connects rather than the first time it is needed.

| Variable | Description |
| -------- | ----------- |
| `TOKEN` | User token(s), comma-separated for multiple accounts |
| `DATA_DIR` | Where writable state lives (default `./var`) |
| `OCR_API_KEY` | OCR.space key for quest image parsing |
| `CAPTCHA_API_KEY` | Optional capsolver/capmonster/2captcha key |
| `CAPTCHA_SERVICE` | `capsolver` (default), `capmonster`, or `2captcha` |
| `CAPTCHA_SOLVE_TIMEOUT` | Seconds to wait for a solve (default 90) |
| `BROWSER_ISOLATED` | `true` (default) — separate Chrome profile per account |
| `BROWSER_PROFILES_DIR` | Override the profile location |
| `BROWSER_EXECUTABLE` | Path to Chrome, if it is not where we look |
| `NOTIFICATIONS` | `true` (default) — desktop notification on captcha |
| `NO_COLOR` | Any value disables coloured log output |

## Testing

```bash
go test ./...
go test -race ./...
```

## What was simplified vs TypeScript

- No animal sell/sacrifice features
- Cleaner package split instead of one 800-line file
- Standard Go patterns: `sync.Mutex`, goroutines + tickers instead of nested `setTimeout`

## Learn Go from this project

Good files to read as a beginner:

1. `cmd/sempatowo/main.go` — small entry point
2. `internal/config/config.go` — structs + JSON tags
3. `internal/config/duration.go` — custom JSON marshalling on a named type
4. `internal/farm/sender.go` — a goroutine, a channel, and a mutex doing one job
5. `internal/farm/farm.go` — main logic (read in sections)
6. `internal/huntbot/handler.go` — interface pattern (`BotContext`)
