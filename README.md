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

# 2. Edit config/sempatpanick.json (created automatically on first run as config/{username}.json)

# 3. Run
go run ./cmd/sempatowo
```

Or build a binary:

```bash
go build -o sempatowo.exe ./cmd/sempatowo
./sempatowo.exe
```

## Project layout

```
cmd/sempatowo/     Entry point — loads .env, starts one bot per token
internal/
  config/          JSON config types, defaults, hot-reload
  farm/            Main bot: timers, queue, OwO message handlers
  huntbot/         HuntBot autohunt + essence upgrades + password captcha
  captcha/         OwO captcha browser + optional auto-solver
  items/           Inventory item IDs (gems, crates, lootboxes)
  util/            Logger and helpers
config/            Per-account JSON (named after Discord username)
```

## Configuration

Same JSON schema as the TypeScript version. On first login, `config/{your_username}.json` is created from defaults if missing. Edit that file while the bot runs — changes reload automatically.

Key settings:

| Setting                                   | Description                                                |
| ----------------------------------------- | ---------------------------------------------------------- |
| `channels.hunt`                           | Discord channel ID for OwO commands                        |
| `status.hunt` / `status.battle`           | Enable manual hunt/battle loops                            |
| `interval.hunt.minDelay` / `maxDelay`     | Random delay between hunts (ms)                            |
| `huntbot.enabled`                         | Use HuntBot instead of manual hunt                         |
| `checklist_completed`                     | Stop farm when checklist is fully done                     |
| `cashCheck`                               | Track balance for gamble safety limits                     |
| `autoDaily`                               | Standalone daily at PST midnight reset                     |
| `allowAutoQuest`                          | Safety gate for experimental auto-quest (like danger.toml) |
| `ocrApi`                                  | OCR.space API key for quest image parsing                  |
| `autoQuest`                               | Full auto-quest (owo-dusk style)                           |
| `gamble.coinflip` / `slots` / `blackjack` | Auto gambling (owo-dusk style)                             |
| `gamble.goalSystem`                       | Stop gambling when net profit goal hit                     |

## Environment variables

| Variable           | Description                                            |
| ------------------ | ------------------------------------------------------ |
| `TOKEN`            | User token(s), comma-separated for multiple accounts   |
| `CAPTCHA_API_KEY`  | Optional capsolver/capmonster/2captcha key             |
| `CAPTCHA_SERVICE`  | `capsolver` (default), `capmonster`, or `2captcha`     |
| `BROWSER_ISOLATED` | `true` (default) — separate Chrome profile per account |

## What was simplified vs TypeScript

- No animal sell/sacrifice features
- Cleaner package split instead of one 800-line file
- Standard Go patterns: `sync.Mutex`, goroutines + tickers instead of nested `setTimeout`

## Learn Go from this project

Good files to read as a beginner:

1. `cmd/sempatowo/main.go` — small entry point
2. `internal/config/config.go` — structs + JSON tags
3. `internal/farm/farm.go` — main logic (read in sections)
4. `internal/huntbot/handler.go` — interface pattern (`BotContext`)
