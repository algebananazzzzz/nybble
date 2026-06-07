# nybble

**Nybble books your canteen lunch for you, automatically, every week.**

`nybble` is a little Go terminal app (built with [Charm](https://charm.sh)) that automates your weekly canteen-lunch booking. You rank the vendors and dishes you like in the TUI; when next week's slots open it logs in, picks the best available match to your preferences for each day, and books the whole week in one shot.

![demo](docs/demo.gif)

> Personal automation for **your own** account and meals. See [configuration](#configuration)) on how to configure endpoints and environment variables.

## How the ranking algorithm works

For each day, nybble walks your preferences from the top and books the first dish that's actually in stock:

1. **Favorite dishes first.** Your ranked dishes (from "Favorites & menu") are tried in order — the highest one still in stock gets booked, no matter which stall it comes from. A favorite always beats a non-favorite.
2. **Then your vendor ranking.** If no favorite is in stock, it falls back to your ranked vendors and moves to your highest-ranked stall that still has something left.
3. **Least available wins.** To choose _within_ that stall, it takes the dish with the least stock remaining — the scarcer an item and the faster it books out, the more popular it is, so the near-sold-out dish is the canteen's own vote for the best one. If your vendor ranking comes up empty too, that same least-available rule runs across every remaining dish on the menu. **So something always gets booked**.

## Install

**curl (latest release):**

```bash
curl -fsSL https://raw.githubusercontent.com/algebananazzzzz/nybble/main/install.sh | sh
```

Downloads the latest macOS release, verifies its checksum, and drops `nybble` into `/usr/local/bin` (or `~/.local/bin`).

**go install:**

```bash
go install github.com/algebananazzzzz/nybble/cmd/nybble@latest
```

**From source:**

```bash
make build      # produces ./nybble
```

Or grab a `nybble_<version>_darwin_<arch>.tar.gz` from [Releases](https://github.com/algebananazzzzz/nybble/releases), check it against `checksums.txt`, and put `nybble` on your `PATH`.

## Requirements

- **macOS** — scheduling and wake use `launchd` and `pmset`.
- **Go 1.26+** — only if you build from source.
- **[`playwright-cli`](https://www.npmjs.com/package/@playwright/cli)** — required for `nybble auth`, the one-time browser QR login. Install with `npm i -g @playwright/cli && playwright-cli install chromium`. `nybble` checks for it and tells you exactly this if it's missing.
- **[`lark-cli`](https://www.npmjs.com/package/@larksuite/cli)** — optional, for notifications. Install with `npm i -g @larksuite/cli`. The Settings screen auto-detects it (the binary plus a working bot identity) and only offers the Lark channel once it's actually usable; otherwise notifications stay **Off**.

## Configuration

The portal endpoints are **required** and read from the environment — `nybble` exits with a clear error if either is missing.

| Variable           | Required | Purpose                                                         |
| ------------------ | -------- | -------------------------------------------------------------- |
| `NYBBLE_API_BASE`  | yes      | Base URL of the ordering API, e.g. `https://<host>/<app-path>`. |
| `NYBBLE_LOGIN_URL` | yes      | Page opened in the browser for the one-time SSO login.          |

These two endpoints are all you configure by hand. Everything else — your building, pickup point, meal slot, and notification target — is detected from your session during `nybble auth` and adjusted in the TUI, then saved to `config.json`. By default booking notifications DM yourself (the bot uses the `union_id` from your session); pick a different Lark target in Settings.

Put them in a `.env` at the config root, `~/.config/nybble/.env`, so both the TUI and the scheduled daemon read them on every run:

```dotenv
# ~/.config/nybble/.env
NYBBLE_API_BASE=https://<host>/<app-path>
NYBBLE_LOGIN_URL=https://<host>/<login-path>
```

A filled-in `.env` is gitignored — don't commit it.

## First run

Run `nybble` to open the TUI — you set everything up from the dashboard:

- **Re-authenticate** — opens the browser SSO login. Scan the QR and wait for your canteen menu to load (that's also how nybble detects your building). One-time, until the session expires.
- **Favorites & menu** — rank your dishes and vendors. Press `r` to pull in the live menu, reorder the rows, and `s` to save. Rank both: dishes drive the normal pick, vendors decide the fallback.
- **Schedule** — pick your run day, which weekdays to book, the open hour, and how many minutes before the run you want a heads-up. Flip **Enable** on and nybble installs the weekly job that wakes your Mac and books hands-free. It asks for your password once (to schedule the wake) — that's the only prompt.
- **Settings** — where booking alerts go (Lark, or off).

Everything is stored under `~/.config/nybble/` (`config.json`, your rankings, the dish catalog, and your private `cookies.json` session).

## Before you trust it

1. Run `nybble book --dry` — it selects picks and prints the would-be batch but submits nothing. (`--dry` is the only path that doesn't book for real.)
2. Optionally let the scheduled job fire once and check that `~/Library/Logs/nybble.log` shows it woke, fetched live menus, and logged a sensible batch.
3. Booked something you don't want? Cancel it in the portal's own app.

## Releases

Every merge to `main` automatically bumps a semver tag and publishes a GitHub Release (built by [GoReleaser](https://goreleaser.com)). The bump comes from commit messages: **patch** by default, `#minor` or `#major` to go bigger, `#none` to skip a release for that merge.

## Security

`~/.config/nybble/cookies.json` and `.auth/` hold a live SSO session — never commit or share them (both are gitignored). `nybble` books only your own meals, on your schedule, with a single prepared request.

## License

[MIT](LICENSE)
