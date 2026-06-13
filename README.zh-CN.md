# tuborkey

Linux 键盘连发工具。**同时支持 Wayland 和 X11**，基于 XDG RemoteDesktop Portal，无需 XTest、无需 root、不依赖特定显示服务器。

适用于 DFO/DNF 等游戏。

## 工作原理

大多数连发工具依赖 X11 的 XTest 扩展，**在 Wayland 下无法工作**。tuborkey 使用 [XDG RemoteDesktop Portal](https://flatpak.github.io/xdg-desktop-portal/) —— Linux 上官方的、与显示服务器无关的输入注入方式。因此它可以在 GNOME、KDE、Sway、Hyprland 等任意 Wayland 合成器上运行。

首次启动时会弹出一次权限确认对话框。

## 使用方法

```bash
go build -o tuborkey ./src/...
./tuborkey
```

按住配置的按键触发连发，松开停止。Ctrl+C 退出。

### 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-lang` | `en` | 语言：`en` (English) 或 `zh` (中文) |
| `-key` | `j` | 连发按键，支持键名 (j/k/space/enter 等) 或 evdev code (如 36) |
| `-interval` | `50` | 连发间隔，单位毫秒 |
| `-toggle` | `0` | 开关快捷键，支持键名 (f1/scrolllock/0 等) 或 evdev code |

### 示例

```bash
# 默认：0 键开关连发，按住 J 连发，50ms 间隔
./tuborkey

# F1 开关连发，按住 K 连发
./tuborkey -toggle f1 -key k

# ScrollLock 开关，按住空格连发，30ms 间隔
./tuborkey -toggle scrolllock -key space -interval 30

# 使用中文界面
./tuborkey -lang zh
```

## 权限

程序需要读取 `/dev/input/event*` 来监听物理键盘事件，可能需要将用户加入 `input` 组：

```bash
sudo usermod -aG input $USER
```

修改后需要重新登录才能生效。
