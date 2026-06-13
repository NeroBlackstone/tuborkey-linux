package main

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	portalDest  = "org.freedesktop.portal.Desktop"
	portalPath  = "/org/freedesktop/portal/desktop"
	portalIface = "org.freedesktop.portal.RemoteDesktop"
	reqIface    = "org.freedesktop.portal.Request"

	keyJ = 36 // evdev keycode
)

var counter int

func nextToken(prefix string) string {
	counter++
	return fmt.Sprintf("%s%d", prefix, counter)
}

func main() {
	fmt.Println("tuborkey - 连发程序 (Portal/EI 方式)")
	fmt.Println("每 3 秒自动按 J 键，Ctrl+C 退出")
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

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Println("\n退出")
			return
		case <-ticker.C:
			if err := sendKey(conn, sessionPath, keyJ); err != nil {
				fmt.Fprintf(os.Stderr, "按键失败: %v\n", err)
			} else {
				fmt.Printf("[%s] 按下 J\n", time.Now().Format("15:04:05"))
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
