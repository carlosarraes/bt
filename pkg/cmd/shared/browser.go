package shared

import (
	"fmt"
	"os/exec"
	"runtime"
)

// LaunchBrowser opens the given URL in the user's default browser.
func LaunchBrowser(url string) error {
	var cmdName string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmdName = "cmd"
		args = []string{"/c", "start", url}
	case "darwin":
		cmdName = "open"
		args = []string{url}
	case "linux":
		cmdName = "xdg-open"
		args = []string{url}
	default:
		return fmt.Errorf("unsupported platform")
	}

	return exec.Command(cmdName, args...).Start()
}
