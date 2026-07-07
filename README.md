# cssh

Terminal UI for managing and connecting to SSH hosts. Stores connections in `~/.cssh/config.json`.

## Install

```
go install github.com/taavtamm/cssh@latest
```

Or clone and build:

```
git clone https://github.com/taavtamm/cssh.git
cd cssh
make install
```

## Usage

```
cssh              # launch the TUI
cssh --version    # print version
cssh --help       # show usage
```

### Keys

| Key | Action |
|-----|--------|
| `enter` | Connect to selected host |
| `i` | Show connection details |
| `a` | Add connection |
| `e` | Edit connection |
| `c` | Duplicate connection |
| `d` | Delete connection |
| `y` | Copy SSH command to clipboard |
| `/` | Search / filter |
| `T` | Cycle theme |
| `j/k` | Navigate |
| `q` | Quit |

While typing a search, use `↑`/`↓` (or `ctrl+p`/`ctrl+n`) to navigate — letters, including `j` and `k`, go into the query.

Inside the edit form: `tab`/`shift+tab` to move between fields, `ctrl+s` to save, `ctrl+k` to browse SSH keys, `ctrl+f` to add a port forward.

## Features

- Connections organized by groups with tags
- Local, remote, and dynamic port forwarding
- SSH key picker (scans `~/.ssh/`)
- Search across names, hosts, tags, and groups
- Multiple color themes
- Cross-platform clipboard (macOS, Linux via xclip/wl-copy)

Extra args support quoting, e.g. `-o ProxyCommand="ssh -W %h:%p jumphost"` is passed to ssh as a single option.

## Config

Stored at `~/.cssh/config.json` and edited through the TUI.

Notes:

- The app rewrites the file on every change (atomically, via temp file + rename). If you hand-edit it, unknown/extra JSON fields will be dropped on the next save.
- The file lists your hosts and usernames — treat it like `~/.ssh/config` and keep it out of public dotfile repos.

## Development

```
make build    # build ./cssh with version info
make test     # go test ./...
make lint     # gofmt + go vet
make run      # go run .
```

CI runs gofmt, vet, build, cross-compile, and tests on every push and PR. Tagged pushes (`v*`) build release binaries via goreleaser.

## Requirements

- Go 1.22+
- `ssh` on PATH
- For clipboard on Linux: `xclip` or `wl-copy`
