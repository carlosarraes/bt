package skill

import (
	"fmt"
	"os"
	"path/filepath"
)

type agent struct {
	Name     string
	SkillDir string
}

func agentList() ([]agent, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return []agent{
		{Name: "Claude", SkillDir: filepath.Join(homeDir, ".claude", "skills")},
		{Name: "Cursor", SkillDir: filepath.Join(homeDir, ".cursor", "skills")},
		{Name: "Codex", SkillDir: filepath.Join(homeDir, ".codex", "skills")},
	}, nil
}

func detectAgents() ([]agent, error) {
	all, err := agentList()
	if err != nil {
		return nil, err
	}
	var detected []agent
	for _, a := range all {
		if info, err := os.Stat(a.SkillDir); err == nil && info.IsDir() {
			detected = append(detected, a)
		}
	}
	return detected, nil
}

func createSymlinks(force bool) ([]string, error) {
	dir, err := skillDir()
	if err != nil {
		return nil, err
	}
	target := filepath.Join(dir, "bt")

	agents, err := detectAgents()
	if err != nil {
		return nil, err
	}

	var linked []string
	for _, a := range agents {
		linkPath := filepath.Join(a.SkillDir, "bt")

		info, err := os.Lstat(linkPath)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				if err := os.Remove(linkPath); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to remove existing symlink for %s: %v\n", a.Name, err)
					continue
				}
			} else if force {
				if err := os.RemoveAll(linkPath); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to remove existing directory for %s: %v\n", a.Name, err)
					continue
				}
			} else {
				fmt.Fprintf(os.Stderr, "Warning: %s exists and is not a symlink (use --force to replace)\n", linkPath)
				continue
			}
		}

		if err := os.Symlink(target, linkPath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to symlink %s: %v\n", a.Name, err)
			continue
		}
		linked = append(linked, a.Name)
	}
	return linked, nil
}

func removeSymlinks() ([]string, error) {
	agents, err := agentList()
	if err != nil {
		return nil, err
	}

	var removed []string
	for _, a := range agents {
		linkPath := filepath.Join(a.SkillDir, "bt")
		info, err := os.Lstat(linkPath)
		if err != nil {
			continue
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if err := os.Remove(linkPath); err == nil {
				removed = append(removed, a.Name)
			}
		}
	}
	return removed, nil
}

func symlinkStatus() ([]struct {
	Name   string
	Status string
}, error) {
	agents, err := agentList()
	if err != nil {
		return nil, err
	}

	var results []struct {
		Name   string
		Status string
	}
	for _, a := range agents {
		linkPath := filepath.Join(a.SkillDir, "bt")
		info, err := os.Lstat(linkPath)
		if err != nil {
			if _, statErr := os.Stat(a.SkillDir); statErr != nil {
				results = append(results, struct {
					Name   string
					Status string
				}{a.Name, "not installed"})
			} else {
				results = append(results, struct {
					Name   string
					Status string
				}{a.Name, "not linked"})
			}
			continue
		}

		if info.Mode()&os.ModeSymlink == 0 {
			results = append(results, struct {
				Name   string
				Status string
			}{a.Name, "directory (not managed by bt)"})
			continue
		}

		if _, err := os.Stat(linkPath); err != nil {
			results = append(results, struct {
				Name   string
				Status string
			}{a.Name, "broken symlink"})
		} else {
			results = append(results, struct {
				Name   string
				Status string
			}{a.Name, "linked"})
		}
	}
	return results, nil
}
