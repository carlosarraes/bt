package skill

import (
	"context"
	"fmt"
	"time"
)

type StatusCmd struct{}

func (cmd *StatusCmd) Run(ctx context.Context) error {
	currentVersion, err := installedVersion()
	if err != nil {
		fmt.Println("bt skill: not installed")
		fmt.Println("Run 'bt skill add' to install")
		return nil
	}

	fmt.Printf("bt skill v%s\n\n", currentVersion)

	statuses, err := symlinkStatus()
	if err == nil {
		fmt.Println("Agents:")
		for _, s := range statuses {
			fmt.Printf("  %-8s %s\n", s.Name, s.Status)
		}
	}

	fmt.Println()
	remoteVersion, err := fetchRemoteVersion(5 * time.Second)
	if err != nil {
		fmt.Println("Update check: failed (network error)")
		return nil
	}

	if remoteVersion != currentVersion {
		fmt.Printf("Update available: v%s → v%s\n", currentVersion, remoteVersion)
		fmt.Println("Run 'bt skill update' to update")
	} else {
		fmt.Println("Up to date")
	}

	return nil
}
