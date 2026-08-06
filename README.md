# glabtodos

A command-line tool that periodically checks a GitLab instance for pending
TODOs and displays a desktop notification when any are found.

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

Tokens are intentionally not read from the TOML file. Use `op_path`,
`GLAB_TOKEN`, or `--token` instead. Configuration precedence is:

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

If `GLAB_OP_PATH` is set, it takes precedence over `GLAB_TOKEN`. The application
retries every five seconds until the 1Password CLI can read the secret, which
allows it to start before 1Password is ready.

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
