package main

type messages struct {
	flagKey, flagInterval, flagToggle             string
	errPrefix                                     string
	errUnknownKey, errSameKey                     string
	title, keyInfo, toggleInfo, hint              string
	errDbus, errSession, errDevices, errStart     string
	waitPermission, sessionStarted                string
	noKeyboard, listening                         string
	waitToggle                                    string
	exiting, errOpen, errFire                     string
	toggleOn, toggleOff                           string
	errCode, errSessionHandle, errSessionType     string
	errTimeout                                    string
	errCodeRejected                               string
}

var langMessages = map[string]messages{
	"en": {
		flagKey:         "key to turbo-fire (e.g. j/k/space, or evdev code like 36)",
		flagInterval:    "turbo-fire interval in milliseconds",
		flagToggle:      "toggle hotkey (e.g. f1/scrolllock/0, or evdev code)",
		errPrefix:       "error",
		errUnknownKey:   "unknown key: %s",
		errSameKey:      "toggle key cannot be the same as the fire key",
		title:           "tuborkey - keyboard autofire tool (Portal/EI)",
		keyInfo:         "fire key: %s (code %d), interval: %dms",
		toggleInfo:      "toggle key: %s (code %d)",
		hint:            "press toggle key to enable/disable, hold fire key to autofire, Ctrl+C to quit",
		errDbus:         "failed to connect D-Bus: %v",
		errSession:      "failed to create session: %v",
		errDevices:      "failed to select devices: %v",
		errStart:        "failed to start session: %v",
		waitPermission:  "waiting for permission (dialog may appear)...",
		sessionStarted:  "session started!",
		noKeyboard:      "no keyboard devices found, check /dev/input/ permissions",
		listening:       "listening on keyboard devices:",
		waitToggle:      "press %s to toggle, hold %s to autofire...",
		exiting:         "\nexiting",
		errOpen:         "failed to open %s: %v",
		errFire:         "autofire failed: %v",
		toggleOn:        "[%s] autofire ENABLED",
		toggleOff:       "[%s] autofire DISABLED",
		errCode:         "error code: %d",
		errSessionHandle:"session_handle not found in response: %v",
		errSessionType:  "session_handle has wrong type: %T",
		errTimeout:      "timeout (%v)",
		errCodeRejected: "error code: %d (rejected?)",
	},
	"zh": {
		flagKey:         "连发按键 (键名如 j/k/space，或 evdev code 如 36)",
		flagInterval:    "连发间隔 (毫秒)",
		flagToggle:      "开关快捷键 (键名如 f1/scrolllock/0，或 evdev code)",
		errPrefix:       "错误",
		errUnknownKey:   "未知按键: %s",
		errSameKey:      "开关快捷键不能和连发按键相同",
		title:           "tuborkey - 连发程序 (Portal/EI 方式)",
		keyInfo:         "连发键: %s (code %d), 间隔: %dms",
		toggleInfo:      "开关快捷键: %s (code %d)",
		hint:            "按开关快捷键切换连发，按住连发键触发连发，Ctrl+C 退出",
		errDbus:         "连接 D-Bus 失败: %v",
		errSession:      "创建 session 失败: %v",
		errDevices:      "选择设备失败: %v",
		errStart:        "启动 session 失败: %v",
		waitPermission:  "等待权限确认 (可能弹出对话框)...",
		sessionStarted:  "Session 已启动!",
		noKeyboard:      "未找到键盘设备，请确认 /dev/input/ 权限",
		listening:       "监听键盘设备:",
		waitToggle:      "按 %s 开关连发，按住 %s 触发连发...",
		exiting:         "\n退出",
		errOpen:         "打开 %s 失败: %v",
		errFire:         "连发失败: %v",
		toggleOn:        "[%s] 连发已开启",
		toggleOff:       "[%s] 连发已关闭",
		errCode:         "错误码: %d",
		errSessionHandle:"响应中没有 session_handle: %v",
		errSessionType:  "session_handle 类型错误: %T",
		errTimeout:      "超时 (%v)",
		errCodeRejected: "错误码: %d (用户拒绝?)",
	},
}

var msg messages

func getMsg(lang string) messages {
	if m, ok := langMessages[lang]; ok {
		return m
	}
	return langMessages["en"]
}
