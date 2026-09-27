# glabtodos

A command-line tool that periodically checks one or more GitLab instances for pending
TODOs and displays a desktop notification with their combined count.

Notifications are provided by [beeep](https://github.com/gen2brain/beeep), a
cross-platform Go library that supports Linux, macOS, and Windows.

## Setup

The application automatically loads this file when it exists:

- Linux: `~/.config/glabtodos/config.toml`
- macOS: `~/Library/Application Support/glabtodos/config.toml`
- Windows: `%APPDATA%\\glabtodos\\config.toml`

A different file can be selected with `--config PATH`; `--no-config` disables
configuration-file loading. An explicitly supplied file must exist.

Example `config.toml`:

```toml
host = "https://gitlab.example.com"
api_path = "/api/v3/"
op_path = "op://Personal/GitLab/API Token"
op_command = "op.exe"
delay = "90s"
```

For multiple GitLab servers, configure an instance list in the same file:

```toml
delay = "90s"
op_command = "op.exe" # optional default for all instances

[[instances]]
name = "work"
host = "https://gitlab.work.example"
api_path = "/api/v4/"
op_path = "op://Work/GitLab/API Token"

[[instances]]
name = "personal"
host = "https://gitlab.personal.example"
api_path = "/api/v4/"
op_path = "op://Personal/GitLab/API Token"
# op_command = "op" # optional per-instance override
```

Names must be unique. Each instance needs its own `op_path`; tokens are never
read from TOML. Shared `delay`, `notify`, and `icon` settings apply to all
instances. Single-instance host, API path, token, and 1Password flags/environment
variables are ignored when `[[instances]]` is present; `GLAB_OP_COMMAND` or
`--op-command` sets the default CLI for instances without an `op_command`.

Instances are polled independently. The notification combines available counts;
if one instance fails, its count is excluded and its name is shown as unavailable.
Each failed instance retries with exponential backoff (up to 30 minutes) without
stopping the others. If all fail, no TODO notification is sent. Counts include
all GitLab API pages.

In single-instance mode, use `op_path`, `GLAB_TOKEN`, or `--token` instead of
storing a token in TOML. Configuration precedence is:

```text
defaults < TOML file < environment variables < command-line flags
```

Set the following environment variables, or provide the equivalent command-line
flags:

- `GLAB_HOST` - The scheme and host (for example, `https://gitlab.example.com`)
- `GLAB_APIPATH` - The GitLab API path (for example, `/api/v3/`)
- `GLAB_TOKEN` - Your GitLab access token
- `GLAB_OP_PATH` - 1Password secret reference for the GitLab token (for example, `op://Personal/GitLab/API Token`)
- `GLAB_OP_COMMAND` - 1Password CLI command; defaults to `op.exe`
- `GLAB_DELAY` - The interval between polling requests; defaults to `90s`

If `GLAB_OP_PATH` is set, it takes precedence over `GLAB_TOKEN` in
single-instance mode. An unavailable 1Password secret is retried with per-instance
backoff; it does not prevent other instances from being checked.

Optional settings:

- `GLAB_ICON` - Path to an icon used for desktop notifications
- `GLAB_NOTIFY` - External command to run when pending TODOs are found

## Install

```sh
go install github.com/outten45/glabtodos@latest
```

## Build

Build binaries for Linux, macOS, and Windows with:

```sh
make
```

The binaries are placed in `dist/`:

- `dist/glabtodos-linux-amd64`
- `dist/glabtodos-darwin-amd64`
- `dist/glabtodos-windows-amd64.exe`

To run directly on the current system:

```sh
make run
```

## Development

For local development, [Devbox](https://www.jetify.com/devbox) is recommended.
