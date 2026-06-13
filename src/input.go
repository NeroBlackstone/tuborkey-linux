package main

import (
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/godbus/dbus/v5"
)

// inputEvent 对应 Linux 内核的 struct input_event
// 在不同架构上大小不同（x86_64 为 24 字节，32 位系统为 16 字节）
type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

const (
	evKey = 0x01

	// inputEventSize 是当前架构下 inputEvent 的字节大小
	inputEventSize = int(unsafe.Sizeof(inputEvent{}))
)

// keyMap 键名 → evdev keycode 映射
var keyMap = map[string]uint32{
	"a": 30, "b": 48, "c": 46, "d": 32, "e": 18, "f": 33,
	"g": 34, "h": 35, "i": 23, "j": 36, "k": 37, "l": 38,
	"m": 50, "n": 49, "o": 24, "p": 25, "q": 16, "r": 19,
	"s": 31, "t": 20, "u": 22, "v": 47, "w": 17, "x": 45,
	"y": 21, "z": 44,
	"0": 11, "1": 2, "2": 3, "3": 4, "4": 5,
	"5": 6, "6": 7, "7": 8, "8": 9, "9": 10,
	"space": 57, "enter": 28, "esc": 1, "tab": 15,
	"shift": 42, "ctrl": 29, "alt": 56,
	"up": 103, "down": 108, "left": 105, "right": 106,
	"f1": 59, "f2": 60, "f3": 61, "f4": 62, "f5": 63,
	"f6": 64, "f7": 65, "f8": 66, "f9": 67, "f10": 68,
	"f11": 87, "f12": 88,
}

// enabled 控制连发是否开启
var enabled atomic.Bool

// parseKey 将键名或数字转为 evdev keycode
func parseKey(s string) (uint32, error) {
	// 优先从 keyMap 中查找键名
	if code, ok := keyMap[strings.ToLower(s)]; ok {
		return code, nil
	}
	// 如果 keyMap 中没有，尝试作为数字解析
	var code uint32
	if _, err := fmt.Sscanf(s, "%d", &code); err == nil {
		return code, nil
	}
	return 0, fmt.Errorf("unknown key: %s", s)
}

// findKeyboardDevices 从 /proc/bus/input/devices 查找键盘设备
func findKeyboardDevices() []string {
	data, err := os.ReadFile("/proc/bus/input/devices")
	if err != nil {
		return nil
	}

	var devices []string
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "H: Handlers=") && strings.Contains(line, "kbd") {
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "event") {
					devices = append(devices, "/dev/input/"+part)
				}
			}
		}
	}
	return devices
}

// monitorKeyboard 监听键盘设备的按键事件
func monitorKeyboard(devicePath string, conn *dbus.Conn, sessionPath dbus.ObjectPath, done <-chan struct{}, rf *rapidFireCtrl, keyCode, toggleCode uint32, intervalMs int) {
	fd, err := syscall.Open(devicePath, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, msg.errOpen+"\n", devicePath, err)
		return
	}
	defer syscall.Close(fd)

	for {
		select {
		case <-done:
			return
		default:
		}

		var ev inputEvent
		n, err := syscall.Read(fd, (*[inputEventSize]byte)(unsafe.Pointer(&ev))[:])
		if err != nil || n < inputEventSize {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		if ev.Type == evKey {
			// 开关快捷键：按下时切换
			if ev.Code == uint16(toggleCode) && ev.Value == 1 {
				newState := !enabled.Load()
				enabled.Store(newState)
				if newState {
					fmt.Printf(msg.toggleOn+"\n", time.Now().Format("15:04:05"))
				} else {
					rf.stop()
					fmt.Printf(msg.toggleOff+"\n", time.Now().Format("15:04:05"))
				}
			}

			// 连发键：仅在 enabled 时响应
			if ev.Code == uint16(keyCode) && enabled.Load() {
				switch ev.Value {
				case 1: // 按下
					rf.start(conn, sessionPath, keyCode, intervalMs)
				case 0: // 松开
					rf.stop()
				}
			}
		}
	}
}
