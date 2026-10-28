package app

import (
	"syscall"
	"time"
)

// uptimeMS and sleepMS keep split_windows.go free of time arithmetic (it runs on a goroutine and
// only needs a coarse clock and a short pause).
func uptimeMS() int64 { return time.Now().UnixMilli() }

func sleepMS(ms int64) { time.Sleep(time.Duration(ms) * time.Millisecond) }

// windowsPID is a helper for tests: the pid of this process as a plain number.
func windowsPID() uint32 { return uint32(syscall.Getpid()) }
