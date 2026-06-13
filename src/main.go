package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/godbus/dbus/v5"
)

func main() {
	lang := flag.String("lang", "en", "language: en (English) or zh (Chinese)")
	keyName := flag.String("key", "j", "key to fire (e.g. j/k/space, or evdev code like 36)")
	interval := flag.Int("interval", 50, "fire interval in milliseconds")
	toggleName := flag.String("toggle", "\\", "toggle hotkey (e.g. f1/scrolllock/\\, or evdev code)")
	flag.Parse()

	msg = getMsg(*lang)

	// apply localized flag descriptions (re-parse for help text)
	flag.VisitAll(func(f *flag.Flag) {
		switch f.Name {
		case "key":
			f.DefValue = "j"
		case "interval":
			f.DefValue = "50"
		case "toggle":
			f.DefValue = "\\"
		}
	})

	keyCode, err := parseKey(*keyName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg.errPrefix, err)
		fmt.Fprintf(os.Stderr, "available keys: ")
		names := make([]string, 0, len(keyMap))
		for k := range keyMap {
			names = append(names, k)
		}
		fmt.Fprintf(os.Stderr, "%s\n", strings.Join(names, ", "))
		os.Exit(1)
	}

	toggleCode, err := parseKey(*toggleName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", msg.errPrefix, err)
		os.Exit(1)
	}
	if toggleCode == keyCode {
		fmt.Fprintln(os.Stderr, msg.errPrefix+": "+msg.errSameKey)
		os.Exit(1)
	}

	fmt.Println(msg.title)
	fmt.Printf(msg.keyInfo+"\n", strings.ToLower(*keyName), keyCode, *interval)
	fmt.Printf(msg.toggleInfo+"\n", strings.ToLower(*toggleName), toggleCode)
	fmt.Println(msg.hint)
	fmt.Println("---")

	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		fmt.Fprintf(os.Stderr, msg.errDbus+"\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	uniqueName := strings.TrimPrefix(conn.Names()[0], ":")

	sessionPath, err := createSession(conn, uniqueName)
	if err != nil {
		fmt.Fprintf(os.Stderr, msg.errSession+"\n", err)
		os.Exit(1)
	}
	fmt.Printf("Session: %s\n", sessionPath)

	if err := selectDevices(conn, sessionPath, uniqueName); err != nil {
		fmt.Fprintf(os.Stderr, msg.errDevices+"\n", err)
		os.Exit(1)
	}

	fmt.Println(msg.waitPermission)
	if err := startSession(conn, sessionPath, uniqueName); err != nil {
		fmt.Fprintf(os.Stderr, msg.errStart+"\n", err)
		os.Exit(1)
	}
	fmt.Println(msg.sessionStarted)

	kbdDevices := findKeyboardDevices()
	if len(kbdDevices) == 0 {
		fmt.Fprintln(os.Stderr, msg.noKeyboard)
		os.Exit(1)
	}
	fmt.Printf("%s\n", msg.listening)
	for _, d := range kbdDevices {
		fmt.Printf("  %s\n", d)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan struct{})
	rf := &rapidFireCtrl{}

	for _, dev := range kbdDevices {
		go monitorKeyboard(dev, conn, sessionPath, done, rf, keyCode, toggleCode, *interval)
	}

	fmt.Printf(msg.waitToggle+"\n", strings.ToLower(*toggleName), strings.ToLower(*keyName))

	<-sigCh
	fmt.Println(msg.exiting)
	close(done)
	rf.stop()
}
