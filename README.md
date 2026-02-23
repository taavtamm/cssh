# cssh

Terminal UI for managing and connecting to SSH hosts. Stores connections in `~/.cssh/config.json`.

## Install

```
go install github.com/taavitammiste/cssh@latest
```

Or clone and build:

```
git clone https://github.com/taavitammiste/cssh.git
cd cssh
make install
```

## Usage

```
cssh
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

Inside the edit form: `tab`/`shift+tab` to move between fields, `ctrl+s` to save, `ctrl+k` to browse SSH keys, `ctrl+f` to add a port forward.

## Features

- Connections organized by groups with tags
- Local, remote, and dynamic port forwarding
- SSH key picker (scans `~/.ssh/`)
- Search across names, hosts, tags, and groups
- Multiple color themes
- Cross-platform clipboard (macOS, Linux via xclip/wl-copy)

## Config

Stored at `~/.cssh/config.json`. Edited through the TUI — no need to touch it manually.

## Requirements

- Go 1.22+
- `ssh` on PATH
- For clipboard on Linux: `xclip` or `wl-copy`
