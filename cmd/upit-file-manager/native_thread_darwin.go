//go:build darwin

package main

import "runtime"

func lockNativeThread() {
	runtime.LockOSThread()
}
