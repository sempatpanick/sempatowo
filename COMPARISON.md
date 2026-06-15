# sempatowo vs owo-dusk — Feature Summary

Comparison of [sempatowo](.) (Go) and [owo-dusk](../../python/Bot/owo-dusk) (Python).

> **Warning:** Both are Discord selfbots for OwO Bot. Selfbots violate Discord's Terms of Service. Use at your own risk.

---

## sempatowo

Go rewrite of **owofarm** (TypeScript). Focused OwO farming bot — one binary, minimal surface area.

### Core farming

| Feature            | Notes                                                                                           |
| ------------------ | ----------------------------------------------------------------------------------------------- |
| **Hunt / Battle**  | Scheduled loops with random min/max delays                                                      |
| **HuntBot**        | Full autohunt flow: start hunt, password captcha, essence trait upgrades with weighted priority |
| **Pray / Curse**   | Optional; can target a user ID                                                                  |
| **Zoo**            | Interval-based                                                                                  |
| **Inventory**      | Periodic `inv` checks                                                                           |
| **Checklist**      | Auto `cl`; triggers **daily**, **cookie**, tracks quest/lootbox/crate completion                |
| **Stop when done** | `checklist_completed` stops the farm when checklist is fully done                               |

### Smart automation

| Feature                | Notes                                                                |
| ---------------------- | -------------------------------------------------------------------- |
| **Gems**               | Auto-use hunt / empowered / lucky gems when missing after a hunt     |
| **Crates & lootboxes** | Auto `crate all`, `lootbox all`, `lootbox fabled all` from inventory |
| **Quest**              | Basic only: detects “Say 'owo'” quests and spams `owo` until done    |
| **XP tracking**        | Logs total hunt XP                                                   |

### Captcha & safety

| Feature            | Notes                                                        |
| ------------------ | ------------------------------------------------------------ |
| **Captcha detect** | Pauses all automation                                        |
| **Browser**        | Opens OwO captcha page (isolated Chrome profile per account) |
| **Auto-solver**    | capsolver / capmonster / 2captcha via `CAPTCHA_API_KEY`      |
| **Desktop alerts** | Windows/macOS notifications with deadline warnings (~10 min) |

### Operations

| Feature                  | Notes                                                     |
| ------------------------ | --------------------------------------------------------- |
| **Multi-account**        | Comma-separated `TOKEN` in `.env`                         |
| **Per-account config**   | `config/{username}.json`, **hot-reload** while running    |
| **Message queue**        | Single send queue with configurable interval              |
| **Scheduler**            | Heap-based command scheduling                             |
| **Typing indicator**     | Optional                                                  |
| **Prefix randomization** | `owo` or custom prefix + command aliases (`h`, `b`, etc.) |

### Explicitly out of scope

- Gamble / animal sell-sacrifice (per README; same as unused TS features)
- Broader quest automation beyond “say owo”

---

## owo-dusk

Full-featured Python OwO selfbot (v2.5.0) with a **cog-based** architecture, web dashboard, and many extras.

### Core farming (overlaps with sempatowo)

Hunt, battle, HuntBot + upgrader, pray/curse, zoo, gems, lootbox/crate, cookie, daily, checklist-style flows, captcha handling.

### Extra farming & economy

| Feature               | Cog / area                                             |
| --------------------- | ------------------------------------------------------ |
| **Auto daily**        | Dedicated scheduler (PST midnight logic + persistence) |
| **Lottery**           | Auto bet                                               |
| **Shop**              | Auto-buy rings by tier                                 |
| **Sell / Sacrifice**  | Animals by rarity                                      |
| **Army / Pup / Piku** | Periodic commands                                      |
| **Cash check**        | Balance monitoring                                     |
| **Claim mail**        | Auto-claim OwO mail via Discord components             |
| **Level grind**       | Random strings or quotes on a timer                    |

### Gambling

| Feature                          | Notes                                 |
| -------------------------------- | ------------------------------------- |
| **Coinflip / Slots / Blackjack** | Martingale-style (`multiplierOnLose`) |
| **Goal system**                  | Stop gambling at a target balance     |

### Combat & events

| Feature             | Notes                                                   |
| ------------------- | ------------------------------------------------------- |
| **Boss battles**    | Auto-join across guilds, ticket tracking, guild filters |
| **Giveaway joiner** | Scans channels and joins OwO giveaways                  |
| **Event detection** | Seasonal/event message handling (`others` cog)          |

### Quest system (much deeper)

| Feature               | Notes                                                           |
| --------------------- | --------------------------------------------------------------- |
| **Full auto-quest**   | Battle XP, gamble, boss, hunt, catch-by-rank, action-send, etc. |
| **Help others**       | Complete quests for other users                                 |
| **Help channel**      | Post quest help in a channel                                    |
| **OCR / image tools** | Quest detail extraction (`image_to_text`, ONNX captcha model)   |

### Humanization & anti-detection

| Feature              | Notes                                                  |
| -------------------- | ------------------------------------------------------ |
| **Misspell**         | Typos + correction delays                              |
| **Sleep mode**       | Random idle pauses                                     |
| **Reaction bot**     | Trigger hunt/battle/owo/pray via OwO reaction messages |
| **Channel switcher** | Rotate farming channels on a timer                     |
| **Slash commands**   | Optional `useSlashCommands`                            |

### Captcha (richer)

| Feature                     | Notes                             |
| --------------------------- | --------------------------------- |
| **Audio alert**             | Plays sound on captcha            |
| **Termux/mobile**           | Toast, vibrate, TTS on Android    |
| **Image captcha solver**    | Local ONNX model                  |
| **YesCaptcha**              | Additional solver integration     |
| **Recurring notifications** | Repeat alerts until solved        |
| **Console hooks**           | Run shell commands on captcha/ban |
| **Restart after captcha**   | Text command to resume            |

### Control & UI

| Feature              | Notes                                                              |
| -------------------- | ------------------------------------------------------------------ |
| **Text commands**    | `.stop`, `.start`, `.sleep`, `.restart_captcha`, `.switch_channel` |
| **Web dashboard**    | Flask UI — logs, stats, config (password-protected)                |
| **Rich CLI**         | Colored panels, compact mode                                       |
| **Discord webhooks** | Log hunts, rare animals, lootbox, channel switches, captcha pings  |
| **SQLite DB**        | Boss tickets, giveaway history, stats                              |
| **Auto-updater**     | Git pull + config merge                                            |
| **Battery check**    | Pause on low battery (desktop/Termux)                              |
| **Offline status**   | Appear offline while running                                       |
| **Custom commands**  | User-defined command loops                                         |
| **Battle streak**    | Console display + loss notifications                               |

### Infrastructure

- Install scripts (Windows / Linux / Termux)
- `tokens.txt` multi-account
- Priority command queue with rate limiting (3 cmds / 5 sec)
- Discord Components v2 support for buttons/menus
- Weekly runtime tracking

---

## Features in owo-dusk not in sempatowo

### Economy & items

- Lottery, shop auto-buy, animal sell/sacrifice
- Army / pup / piku commands
- Cash balance monitoring
- Mail auto-claim
- Level grinding (random text / quotes)

### Gambling

- Coinflip, slots, blackjack
- Goal-based gambling stop

### Combat & social

- Boss battle auto-join (multi-guild, ticket system)
- Giveaway joiner
- Seasonal/event detection

### Quest automation

- Full quest solver (not just “say owo”)
- Quest types: battle XP, gamble, boss, hunt, catch-by-rank, etc.
- Help others / help-channel posting
- OCR / image-based quest helpers

### Anti-detection & behavior

- Misspell simulation
- Random sleep/idle mode
- Reaction-based command triggering
- Channel switcher (rotate channels)
- Slash command mode
- Offline Discord status
- Silent messages

### Captcha & alerts (beyond sempatowo)

- Audio alerts
- Termux: toast, vibrate, text-to-speech
- Local image captcha solver (ONNX)
- YesCaptcha
- Recurring notification loop
- Shell commands on captcha/ban
- Webhook pings (with user mention on captcha)

### Control plane & ops

- Web dashboard (Flask)
- In-Discord text commands (stop/start/sleep/restart/switch channel)
- Discord webhook logging (rare animals, lootbox, etc.)
- SQLite persistence (boss, giveaways, stats)
- Auto-updater with config merge
- Battery-low pause
- Custom user-defined commands
- Battle streak tracking & loss alerts
- Rich terminal UI
- Termux/Android-first tooling
- Dedicated daily scheduler (PST-based with file persistence)
- Auto vote (sempatowo tracks vote on checklist but does not act on it)
- Configurable gem tiers, special gems, dynamic special-gem logic, “disable hunt if no gems”
- Command priority queue with Discord slowmode awareness
- Multiple pray/curse targets, ping options, custom channels, counters

---

## Features sempatowo has that owo-dusk does not emphasize

| Feature                       | Notes                                         |
| ----------------------------- | --------------------------------------------- |
| **Go single-binary**          | No Python/venv required                       |
| **Config hot-reload**         | Edit JSON while running; no restart           |
| **Simpler architecture**      | Easier to read/maintain for core farming only |
| **Isolated browser profiles** | `BROWSER_ISOLATED` per account                |

---

## Quick comparison

|              | **sempatowo**                                | **owo-dusk**                                        |
| ------------ | -------------------------------------------- | --------------------------------------------------- |
| **Goal**     | Lean Go port of owofarm                      | Full-featured community selfbot                     |
| **Best for** | Core hunt/battle/HuntBot + checklist farming | Everything: gamble, boss, quests, dashboard, mobile |
| **Quest**    | “Say owo” only                               | Full automation                                     |
| **UI**       | CLI logs                                     | Rich CLI + web dashboard                            |
| **Scope**    | ~28 files, focused packages                  | 27 cogs + website + DB + utils                      |
