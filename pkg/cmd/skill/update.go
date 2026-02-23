package skill

import (
	"context"
	"fmt"
	"os"
	"time"
)

type UpdateCmd struct{}

func (cmd *UpdateCmd) Run(ctx context.Context) error {
	currentVersion, err := installedVersion()
	if err != nil {
		return fmt.Errorf("skill not installed. Run 'bt skill add' first")
	}

	fmt.Println("Checking for updates...")
	remoteVersion, err := fetchRemoteVersion(10 * time.Second)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	if remoteVersion == currentVersion {
		fmt.Printf("Already up to date (v%s)\n", currentVersion)
		updateLastCheck()
		return nil
	}

	fmt.Printf("Updating from v%s to v%s...\n", currentVersion, remoteVersion)
	if _, err := fetchAndStore(); err != nil {
		return fmt.Errorf("failed to update skill: %w", err)
	}

	linked, err := createSymlinks(false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to update symlinks: %v\n", err)
	} else if agents, err := detectAgents(); err == nil && len(linked) < len(agents) {
		fmt.Fprintf(os.Stderr, "Warning: linked %d of %d agents\n", len(linked), len(agents))
	}

	updateLastCheck()
	fmt.Printf("Updated to v%s\n", remoteVersion)
	return nil
}
