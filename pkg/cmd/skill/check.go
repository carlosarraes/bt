package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const checkInterval = 24 * time.Hour

func CheckForUpdate() {
	if !isInstalled() {
		return
	}

	dir, err := skillDir()
	if err != nil {
		return
	}

	checkPath := filepath.Join(dir, ".last_check")
	if !shouldCheck(checkPath) {
		return
	}

	remoteVersion, err := fetchRemoteVersion(2 * time.Second)
	if err != nil {
		return
	}

	currentVersion, err := installedVersion()
	if err != nil {
		return
	}

	updateLastCheckFile(checkPath)

	if remoteVersion != currentVersion {
		fmt.Fprintf(os.Stderr, "\nA new bt skill version is available (v%s → v%s). Run 'bt skill update'\n", currentVersion, remoteVersion)
	}
}

func shouldCheck(checkPath string) bool {
	data, err := os.ReadFile(checkPath)
	if err != nil {
		return true
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return true
	}
	return time.Since(time.Unix(ts, 0)) > checkInterval
}

func updateLastCheck() {
	dir, err := skillDir()
	if err != nil {
		return
	}
	updateLastCheckFile(filepath.Join(dir, ".last_check"))
}

func updateLastCheckFile(path string) {
	if err := os.WriteFile(path, []byte(fmt.Sprintf("%d\n", time.Now().Unix())), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write update check file: %v\n", err)
	}
}
