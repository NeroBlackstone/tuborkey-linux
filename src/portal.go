package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	portalDest  = "org.freedesktop.portal.Desktop"
	portalPath  = "/org/freedesktop/portal/desktop"
	portalIface = "org.freedesktop.portal.RemoteDesktop"
	reqIface    = "org.freedesktop.portal.Request"
)

var tokenCounter int64

func nextToken(prefix string) string {
	n := atomic.AddInt64(&tokenCounter, 1)
	return fmt.Sprintf("%s%d", prefix, n)
}

// waitForPortalResponse 等待 portal 的 Response signal
func waitForPortalResponse(conn *dbus.Conn, requestPath dbus.ObjectPath, timeout time.Duration) (uint32, map[string]dbus.Variant, error) {
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
		return 0, nil, fmt.Errorf(msg.errTimeout, timeout)
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
		return "", fmt.Errorf(msg.errCode, code)
	}

	sv, ok := results["session_handle"]
	if !ok {
		return "", fmt.Errorf(msg.errSessionHandle, results)
	}
	path, ok := sv.Value().(string)
	if !ok {
		return "", fmt.Errorf(msg.errSessionType, sv.Value())
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
		return fmt.Errorf(msg.errCode, code)
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
		return fmt.Errorf(msg.errCodeRejected, code)
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
