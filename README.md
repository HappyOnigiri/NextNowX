# Next Now X

English | [日本語](README.ja.md) | [简体中文](README.zh-CN.md)

**Next Now X** keeps an initiative that is spread across many GitHub pull requests visible from your own machine.
Register how the tasks depend on each other, and Next Now X separates the work you can start now from the work that is waiting on something.

## Features

- **Shows what you can start** — Pull out only the tasks whose dependencies are all satisfied, instead of working out the order again every time.
- **Shows where things are stuck** — Look over the whole initiative as a graph and follow which task is waiting on what, including pull requests awaiting review and tasks that have gone stale.
- **Reflects pull-request state** — Review status, conflicts, and merges are read from GitHub and folded into how the tasks progress.
- **Builds prompts for agents** — Assemble a prompt from a template with the task and its dependencies filled in, ready to copy.
- **Stays on your machine** — Data is stored locally, and the server accepts only local connections by default.

## Installation

### macOS

A prebuilt binary is published for Apple Silicon.

```sh
curl -fsSL https://github.com/HappyOnigiri/NextNowX/releases/latest/download/install.sh | bash
```

Run the same command again to update, or run `nnx update` to check for a newer release and install it. The WebUI offers the same update when one is available.

### Linux / WSL2

Build from source; the same steps work on macOS. Windows is not supported natively.

```sh
git clone https://github.com/HappyOnigiri/NextNowX.git
cd NextNowX
make install
```

`make install` builds the WebUI along with the binary and installs it to `~/.local/bin/nnx`. Set `INSTALL_DIR` to install it elsewhere.
`nnx daemon` and `nnx open` manage the background service on macOS only, so start the server with `nnx serve` instead.

## Usage

Open http://localhost:7331/ in a browser. When that port is taken and the server moved to another one, `nnx open` opens whichever address it is actually listening on.

The `nnx` command reads and writes the same data, so an AI agent can look at the current state and register tasks and dependencies.

```sh
nnx ready      # pull out the tasks you can start
nnx graph F-1  # see a whole initiative with its tasks and dependencies
nnx prompt T-1 # assemble the prompt to hand to a task
```

See `nnx -h` and `nnx <command> -h` for commands and options.

## More options

- **Synchronize with GitHub:** supply a credential through `nnx config`, `GITHUB_TOKEN`, `GH_TOKEN`, or an authenticated `gh` CLI. Tasks and dependencies work the same way without it.
- **Pin a port:** `nnx config server update PORT`. Run `nnx daemon restart` afterwards to move a server that is already running.
- **Run in the foreground:** `nnx serve` runs the server in the foreground on any operating system.
- **Language:** `nnx setup` asks whether to use English or Japanese and saves the answer; `nnx config language update auto|en|ja` changes it later.
- **Sample data:** the first `nnx setup` adds a small sample project in the language you chose when it creates the database; `nnx setup --no-sample-data` skips it.
- **Demo:** `nnx serve --demo` starts a demo loaded with sample data. It leaves your own data untouched, so use it to try Next Now X first.

## Uninstallation

```sh
curl -fsSL https://github.com/HappyOnigiri/NextNowX/releases/latest/download/uninstall.sh | bash
```

## Development

```sh
make dev  # start the development server: http://127.0.0.1:7331
make demo # same, backed by isolated demo data
make ci   # run every check before handing off a change
```

`make demo` restarts the API on Go changes, which recreates the demo data from scratch.

## Documentation

- [docs/cli/nnx.md](docs/cli/nnx.md): the CLI reference in Markdown.
- [docs/design/](docs/design/README.md): design decisions and the reasoning behind them (Japanese).
- [docs/development.md](docs/development.md): verification and release rules (Japanese).

## Contributing

Contributions are welcome!
Share bug reports and ideas through [Issues](https://github.com/HappyOnigiri/NextNowX/issues), or send a [pull request](https://github.com/HappyOnigiri/NextNowX/pulls).
Documentation improvements and translations are welcome too.
