# canteen — a TUI canteen-lunch autobooker

Auto-books your canteen lunch by winning the weekly booking race the moment next
week's slots open. A single Go binary with a [Charm](https://charm.sh) TUI for
setup and a headless `book` command the scheduler fires.

![demo](docs/demo.gif)

> Personal automation for **your own** account and meals. The canteen ordering
> portal it talks to is deployment-specific and is configured entirely through
> environment variables (see [Configuration](#configuration)) — no endpoint is
> baked into the source.

## How it works

1. When next week's lunch slots open, each dish has a limited stock.
2. The job wakes the Mac (`pmset`) a few minutes early, then at the open time
   fetches each open weekday's menu, picks your highest-ranked **in-stock**
   favorite per day, and submits them all in **one batch**.
3. Sold-out picks fall through to your next favorite. You get a notification with
   the result.

## Install

### curl (latest release)

```bash
curl -fsSL https://raw.githubusercontent.com/algebananazzzzz/bytecanteen/main/install.sh | sh
```

Downloads the latest macOS release, verifies its checksum, and installs `canteen`
to `/usr/local/bin` (or `~/.local/bin`).

### go install

```bash
go install github.com/algebananazzzzz/bytecanteen/cmd/canteen@latest
```

### Manual

Grab a `canteen_<version>_darwin_<arch>.tar.gz` from
[Releases](https://github.com/algebananazzzzz/bytecanteen/releases), verify it
against `checksums.txt`, extract, and move `canteen` onto your `PATH`.

### From source

```bash
make build      # produces ./canteen
```

## Configuration

The portal endpoints are **required** and read from the environment — the app
exits with a clear error if either is unset. Put them in your shell profile:

| Variable | Required | Purpose |
|---|---|---|
| `CANTEEN_API_BASE` | yes | Base URL of the ordering API, e.g. `https://<host>/<app-path>`. |
| `CANTEEN_LOGIN_URL` | yes | Page opened in the browser for the one-time SSO login. |
| `CANTEEN_LARK_TARGET` | no | Lark receive-id (`ou_…`/`oc_…`) for booking notifications. |

Either export them in your shell profile, or — easier — drop them in a `.env`
file. `canteen` loads `.env` from `./.env` then `~/.config/canteen/.env` on every
run (real environment variables still win; set `CANTEEN_ENV_FILE` to point
elsewhere). Copy [`.env.example`](.env.example) to get started:

```bash
mkdir -p ~/.config/canteen
cp .env.example ~/.config/canteen/.env
# edit ~/.config/canteen/.env with your real URLs
```

```dotenv
# ~/.config/canteen/.env
CANTEEN_API_BASE=https://<host>/<app-path>
CANTEEN_LOGIN_URL=https://<host>/<login-path>
```

A `.env` is gitignored — never commit a filled-in one. `canteen schedule on`
also snapshots the resolved values into the launchd job, so scheduled runs
resolve the same endpoints even with an empty shell.

## Requirements

- macOS (uses `launchd` and `pmset` for scheduling/wake).
- Go 1.26+ to build from source.
- [`playwright-cli`](https://www.npmjs.com/package/@playwright/cli) installed
  globally — used **once** for the QR login
  (`npm i -g @playwright/cli && playwright-cli install chromium`).
- [`lark-cli`](https://www.npmjs.com/package/@larksuite/cli) (`@larksuite/cli`),
  **optional**, for notifications. Settings auto-detects it (binary plus a working
  bot identity, via a no-side-effect `bot/v3/info` check) and only then offers the
  Lark channel; messages send `--as bot`. Otherwise notifications are **Off**.

## Setup (first run)

```bash
canteen auth        # opens a browser; scan the SSO QR, wait for the menu, press Enter
canteen             # launches the TUI:
                    #   Settings         -> run day, booking days, open hour, notifications
                    #   Favorites & menu -> rank dishes (Shift+J/K to reorder, s to save)
canteen menu        # prints upcoming menus + grows the dish catalog (run a few times to seed it)
canteen book --dry  # dress rehearsal: shows what it WOULD book, places nothing
canteen schedule on # installs the weekly launchd job + pmset wake (asks sudo for pmset)
```

Config lives in `~/.config/canteen/` (`config.json`, `favorites.json`,
`catalog.json`, `cookies.json`). `cookies.json` is your session — keep it private.

## The weekly flow

- The Mac wakes a few minutes before your **run day/time** and `canteen book`
  runs at the open time.
- It books your top in-stock favorite for each weekday ticked in Settings
  (**Book on days**). Booking is always live — there is no mode toggle.
- Notifications report: run started, booked dishes (or "no favorites"), and
  auth-expiry.

### Before you trust it

1. Run `canteen book --dry` — a preview that selects picks and prints the
   would-be batch but submits nothing. (`--dry` is the only non-live path.)
2. Optionally let the scheduled job fire once and check
   `~/Library/Logs/canteen.log` shows it woke, fetched live menus, and logged a
   sensible batch.
3. Wrong pick? Cancel it in the portal's own app.

## Commands

| Command | Does |
|---|---|
| `canteen` | Launch the TUI (state-aware dashboard). |
| `canteen auth` | One-time browser QR login -> saves cookies. |
| `canteen menu` | Print upcoming menus, update the dish catalog. |
| `canteen book [--dry]` | Run a booking (the scheduler calls this). `--dry` forces a no-submit preview. |
| `canteen schedule on/off` | Install / remove the weekly launchd job + pmset wake. |
| `canteen --version` | Print version, commit, and build date. |

## Versioning & releases

Every merge to `main` automatically bumps a semver tag and publishes a GitHub
Release (built by [GoReleaser](https://goreleaser.com)). The bump is computed
from commit messages:

- default: **patch** (`v1.2.3` -> `v1.2.4`)
- include `#minor` in a commit subject for a minor bump, `#major` for major
- include `#none` to skip a release for that merge

## Security

`~/.config/canteen/cookies.json` and `.auth/` hold a live SSO session — never
commit or share them (both are gitignored). This tool books only your own meals,
on your schedule, with a single prepared request.

## Layout

```
cmd/canteen      entry (cobra; no args -> TUI)
internal/
  config menu catalog selector booker   core booking logic (unit-tested vs fixtures)
  api session                           HTTP client + cookie session
  run                                   orchestrator + book/menu commands
  notify                                Lark notifications (lark-cli, --as bot)
  clock schedule                        race timing + launchd/pmset
  tui                                   Bubble Tea dashboard + screens
```

## License

[MIT](LICENSE)
