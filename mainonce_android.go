//go:build android

package main

import "sync/atomic"

var mainStarted atomic.Bool

// firstMain reports whether this is the first main() in the process. Android
// often destroys the activity but keeps the process cached; reopening the app
// then creates a new activity whose nativeInit starts main() again. A second
// app.Run() fails with "application is running" and log.Fatal kills the
// process, so the app closed instantly after being away a while. The Go app
// lives as long as the process; nativeInit already points the JNI bridge at
// the new activity.
func firstMain() bool {
	return mainStarted.CompareAndSwap(false, true)
}
