# tuborkey

DNF 连发工具，基于 XDG RemoteDesktop Portal，兼容 Wayland。

## 使用方法

```bash
go build -o tuborkey .
./tuborkey
```

按住配置的按键触发连发，松开停止。Ctrl+C 退出。

### 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-key` | `j` | 连发按键，支持键名 (j/k/space/enter 等) 或 evdev code (如 36) |
| `-interval` | `50` | 连发间隔，单位毫秒 |

### 示例

```bash
# 默认：按住 J 连发，50ms 间隔
./tuborkey

# 按住 K 连发
./tuborkey -key k

# 按住空格连发，30ms 间隔
./tuborkey -key space -interval 30

# 直接指定 evdev code
./tuborkey -key 36
```

## 权限

程序需要读取 `/dev/input/event*` 来监听物理键盘事件，可能需要将用户加入 `input` 组：

```bash
sudo usermod -aG input $USER
```

修改后需要重新登录才能生效。
