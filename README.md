# tuborkey

[中文文档](README.zh-CN.md)

A Linux keyboard autofire tool. **Works on both Wayland and X11** via XDG RemoteDesktop Portal — no XTest, no root, no display server dependency.

Useful for DFO/DNF and similar games.

## How it works

Most autofire tools rely on X11's XTest extension, which **does not work on Wayland**. tuborkey uses the [XDG RemoteDesktop Portal](https://flatpak.github.io/xdg-desktop-portal/) instead — the official, display-server-agnostic way to inject input on Linux. This means it works on GNOME, KDE, Sway, Hyprland, and any other Wayland compositor.

A one-time permission dialog will appear when the session starts.

## Usage

```bash
go build -o tuborkey ./src/...
./tuborkey
```

Hold the configured key to trigger turbo-fire, release to stop. Press Ctrl+C to exit.

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `-lang` | `en` | Language: `en` (English) or `zh` (Chinese) |
| `-key` | `j` | Key to turbo-fire, supports key names (j/k/space/enter etc.) or evdev codes (e.g. 36) |
| `-interval` | `50` | Turbo-fire interval in milliseconds |
| `-toggle` | `\` | Toggle hotkey to enable/disable turbo-fire, supports key names (f1/scrolllock/\\ etc.) or evdev codes |

### Examples

```bash
# Default: toggle with \, turbo-fire J when held, 50ms interval
./tuborkey

# Toggle with F1, turbo-fire K when held
./tuborkey -toggle f1 -key k

# Toggle with ScrollLock, turbo-fire Space when held, 30ms interval
./tuborkey -toggle scrolllock -key space -interval 30

# Use Chinese interface
./tuborkey -lang zh
```

## Permissions

The program reads `/dev/input/event*` to listen for physical keyboard events. You may need to add your user to the `input` group:

```bash
sudo usermod -aG input $USER
```

A re-login is required for the change to take effect.
