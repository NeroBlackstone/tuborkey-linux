package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

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
				fmt.Fprintf(os.Stderr, msg.errFire+"\n", err)
			}
		}
	}
}
