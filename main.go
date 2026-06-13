package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	portalDest  = "org.freedesktop.portal.Desktop"
	portalPath  = "/org/freedesktop/portal/desktop"
	portalIface = "org.freedesktop.portal.RemoteDesktop"
	reqIface    = "org.freedesktop.portal.Request"

	evKey = 0x01
)

// keyMap 键名 → evdev keycode 映射
var keyMap = map[string]uint32{
	"a": 30, "b": 48, "c": 46, "d": 32, "e": 18, "f": 33,
	"g": 34, "h": 35, "i": 23, "j": 36, "k": 37, "l": 38,
	"m": 50, "n": 49, "o": 24, "p": 25, "q": 16, "r": 19,
	"s": 31, "t": 20, "u": 22, "v": 47, "w": 17, "x": 45,
	"y": 21, "z": 44,
	"space": 57, "enter": 28, "esc": 1, "tab": 15,
	"shift": 42, "ctrl": 29, "alt": 56,
	"up": 103, "down": 108, "left": 105, "right": 106,
	"f1": 59, "f2": 60, "f3": 61, "f4": 62, "f5": 63,
	"f6": 64, "f7": 65, "f8": 66, "f9": 67, "f10": 68,
	"f11": 87, "f12": 88,
}

// parseKey 将键名或数字转为 evdev keycode
func parseKey(s string) (uint32, error) {
	// 先尝试数字 (直接指定 evdev code)
	var code uint32
	if _, err := fmt.Sscanf(s, "%d", &code); err == nil {
		return code, nil
	}
	// 再查映射表
	if code, ok := keyMap[strings.ToLower(s)]; ok {
		return code, nil
	}
	return 0, fmt.Errorf("未知按键: %s", s)
}

var counter int

func nextToken(prefix string) string {
	counter++
	return fmt.Sprintf("%s%d", prefix, counter)
}

func main() {
	keyName := flag.String("key", "j", "连发按键 (键名如 j/k/space，或 evdev code 如 36)")
	interval := flag.Int("interval", 50, "连发间隔 (毫秒)")
	flag.Parse()

	keyCode, err := parseKey(*keyName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "错误: %v\n", err)
		fmt.Fprintf(os.Stderr, "可用键名: ")
		names := make([]string, 0, len(keyMap))
		for k := range keyMap {
			names = append(names, k)
		}
		fmt.Fprintf(os.Stderr, "%s\n", strings.Join(names, ", "))
		os.Exit(1)
	}

	fmt.Println("tuborkey - 连发程序 (Portal/EI 方式)")
	fmt.Printf("按键: %s (code %d), 间隔: %dms\n", strings.ToLower(*keyName), keyCode, *interval)
	fmt.Println("按住按键触发连发，Ctrl+C 退出")
	fmt.Println("---")

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		fmt.Fprintf(os.Stderr, "连接 D-Bus 失败: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	uniqueName := strings.TrimPrefix(conn.Names()[0], ":")

	sessionPath, err := createSession(conn, uniqueName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建 session 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Session: %s\n", sessionPath)

	if err := selectDevices(conn, sessionPath, uniqueName); err != nil {
		fmt.Fprintf(os.Stderr, "选择设备失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("等待权限确认 (可能弹出对话框)...")
	if err := startSession(conn, sessionPath, uniqueName); err != nil {
		fmt.Fprintf(os.Stderr, "启动 session 失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Session 已启动!")

	// 查找键盘设备
	kbdDevices := findKeyboardDevices()
	if len(kbdDevices) == 0 {
		fmt.Fprintln(os.Stderr, "未找到键盘设备，请确认 /dev/input/ 权限")
		os.Exit(1)
	}
	fmt.Printf("监听键盘设备:\n")
	for _, d := range kbdDevices {
		fmt.Printf("  %s\n", d)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan struct{})
	rf := &rapidFireCtrl{}

	// 监听所有键盘设备的按键事件
	for _, dev := range kbdDevices {
		go monitorKeyboard(dev, conn, sessionPath, done, rf, keyCode, *interval)
	}

	fmt.Printf("按住 %s 键开始连发...\n", strings.ToLower(*keyName))

	<-sigCh
	fmt.Println("\n退出")
	close(done)
	rf.stop()
}

// rapidFireCtrl 控制连发的启停
type rapidFireCtrl struct {
	mu     sync.Mutex
	active bool
	stopCh chan struct{}
}

func (r *rapidFireCtrl) start(conn *dbus.Conn, sessionPath dbus.ObjectPath, keyCode uint32, intervalMs int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.active {
		return
	}
	r.active = true
	r.stopCh = make(chan struct{})
	go rapidFireLoop(conn, sessionPath, r.stopCh, keyCode, intervalMs)
}

func (r *rapidFireCtrl) stop() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.active {
		return
	}
	r.active = false
	close(r.stopCh)
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

// monitorKeyboard 监听键盘设备的按键事件，按住时触发连发
func monitorKeyboard(devicePath string, conn *dbus.Conn, sessionPath dbus.ObjectPath, done <-chan struct{}, rf *rapidFireCtrl, keyCode uint32, intervalMs int) {
	fd, err := syscall.Open(devicePath, syscall.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开 %s 失败: %v\n", devicePath, err)
		return
	}
	defer syscall.Close(fd)

	// sizeof(struct input_event) on amd64 = 24 bytes
	buf := make([]byte, 24)

	for {
		select {
		case <-done:
			return
		default:
		}

		n, err := syscall.Read(fd, buf)
		if err != nil || n < 24 {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		// 解析 input_event (little-endian, amd64)
		// offset 0-15: struct timeval (16 bytes)
		// offset 16-17: type (uint16)
		// offset 18-19: code (uint16)
		// offset 20-23: value (int32)
		evType := uint16(buf[16]) | uint16(buf[17])<<8
		evCode := uint16(buf[18]) | uint16(buf[19])<<8
		evValue := int32(buf[20]) | int32(buf[21])<<8 | int32(buf[22])<<16 | int32(buf[23])<<24

		if evType == evKey && evCode == uint16(keyCode) {
			if evValue == 1 {
				rf.start(conn, sessionPath, keyCode, intervalMs)
				fmt.Printf("[%s] 按下 - 开始连发\n", time.Now().Format("15:04:05"))
			} else if evValue == 0 {
				rf.stop()
				fmt.Printf("[%s] 松开 - 停止连发\n", time.Now().Format("15:04:05"))
			}
		}
	}
}

// rapidFireLoop 快速连发按键
func rapidFireLoop(conn *dbus.Conn, sessionPath dbus.ObjectPath, stop <-chan struct{}, keyCode uint32, intervalMs int) {
	ticker := time.NewTicker(time.Duration(intervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			if err := sendKey(conn, sessionPath, keyCode); err != nil {
				fmt.Fprintf(os.Stderr, "连发失败: %v\n", err)
			}
		}
	}
}

// waitForPortalResponse 等待 portal 的 Response signal
func waitForPortalResponse(conn *dbus.Conn, requestPath dbus.ObjectPath, timeout time.Duration) (uint32, map[string]dbus.Variant, error) {
	// 使用字符串 match rule
	rule := fmt.Sprintf("type='signal',interface='%s',path='%s',member='Response'", reqIface, string(requestPath))
	conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, rule)

	ch := make(chan *dbus.Signal, 10)
	conn.Signal(ch)
	defer conn.RemoveSignal(ch)

	select {
	case sig := <-ch:
		if len(sig.Body) < 2 {
			return 0, nil, fmt.Errorf("unexpected response: %v", sig.Body)
		}
		code, _ := sig.Body[0].(uint32)
		results, _ := sig.Body[1].(map[string]dbus.Variant)
		return code, results, nil
	case <-time.After(timeout):
		return 0, nil, fmt.Errorf("超时 (%v)", timeout)
	}
}

func createSession(conn *dbus.Conn, uniqueName string) (dbus.ObjectPath, error) {
	token := nextToken("tb")
	sessionToken := nextToken("sess")
	reqPath := dbus.ObjectPath(fmt.Sprintf("/org/freedesktop/portal/desktop/request/%s/%s", uniqueName, token))

	obj := conn.Object(portalDest, portalPath)
	options := map[string]dbus.Variant{
		"handle_token":        dbus.MakeVariant(token),
		"session_handle_token": dbus.MakeVariant(sessionToken),
	}
	call := obj.Call(portalIface+".CreateSession", 0, options)
	if call.Err != nil {
		return "", call.Err
	}

	code, results, err := waitForPortalResponse(conn, reqPath, 10*time.Second)
	if err != nil {
		return "", err
	}
	if code != 0 {
		return "", fmt.Errorf("错误码: %d", code)
	}

	sv, ok := results["session_handle"]
	if !ok {
		return "", fmt.Errorf("响应中没有 session_handle: %v", results)
	}
	path, ok := sv.Value().(string)
	if !ok {
		return "", fmt.Errorf("session_handle 类型错误: %T", sv.Value())
	}
	return dbus.ObjectPath(path), nil
}

func selectDevices(conn *dbus.Conn, sessionPath dbus.ObjectPath, uniqueName string) error {
	token := nextToken("tb")
	reqPath := dbus.ObjectPath(fmt.Sprintf("/org/freedesktop/portal/desktop/request/%s/%s", uniqueName, token))

	obj := conn.Object(portalDest, portalPath)
	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token),
		"types":        dbus.MakeVariant(uint32(1)), // keyboard
	}
	call := obj.Call(portalIface+".SelectDevices", 0, sessionPath, options)
	if call.Err != nil {
		return call.Err
	}

	code, _, err := waitForPortalResponse(conn, reqPath, 10*time.Second)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("错误码: %d", code)
	}
	return nil
}

func startSession(conn *dbus.Conn, sessionPath dbus.ObjectPath, uniqueName string) error {
	token := nextToken("tb")
	reqPath := dbus.ObjectPath(fmt.Sprintf("/org/freedesktop/portal/desktop/request/%s/%s", uniqueName, token))

	obj := conn.Object(portalDest, portalPath)
	options := map[string]dbus.Variant{
		"handle_token": dbus.MakeVariant(token),
	}
	call := obj.Call(portalIface+".Start", 0, sessionPath, "", options)
	if call.Err != nil {
		return call.Err
	}

	code, _, err := waitForPortalResponse(conn, reqPath, 30*time.Second)
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("错误码: %d (用户拒绝?)", code)
	}
	return nil
}

func sendKey(conn *dbus.Conn, sessionPath dbus.ObjectPath, keycode uint32) error {
	obj := conn.Object(portalDest, portalPath)
	options := map[string]dbus.Variant{}

	// 按下
	call := obj.Call(portalIface+".NotifyKeyboardKeycode", 0, sessionPath, options, int32(keycode), uint32(1))
	if call.Err != nil {
		return fmt.Errorf("press: %w", call.Err)
	}

	time.Sleep(10 * time.Millisecond)

	// 释放
	call = obj.Call(portalIface+".NotifyKeyboardKeycode", 0, sessionPath, options, int32(keycode), uint32(0))
	if call.Err != nil {
		return fmt.Errorf("release: %w", call.Err)
	}
	return nil
}
