//go:build !linux && !windows && !ios

package services

// Desktops Nova doesn't ship for yet (macOS, the BSDs) still build; they use
// the Wails updater without any workarounds.

func packageManaged() bool { return false }

func stageNextToApp() {}

func cleanupAfterUpdate() {}
