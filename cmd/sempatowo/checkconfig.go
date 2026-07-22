package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/semptpanick/sempatowo/internal/config"
)

// runCheckConfig validates the environment and every config file without
// connecting to Discord, and returns the process exit code. Config problems
// used to be discoverable only by starting the bot and watching what failed to
// happen; this makes them a command you can run before you care.
func runCheckConfig(env *config.Env) int {
	fmt.Println("Environment OK")
	fmt.Printf("  accounts:        %d token(s)\n", len(env.Tokens))
	fmt.Printf("  captcha solver:  %s\n", describeSolver(env.Captcha))
	fmt.Printf("  notifications:   %s\n", onOff(env.Notifications))
	fmt.Printf("  isolated chrome: %s\n", onOff(env.Browser.Isolated))
	fmt.Printf("  data root:       %s\n", env.Dirs.Root)
	fmt.Println()

	paths, err := configFiles(env.Dirs.Config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot read %s: %v\n", env.Dirs.Config, err)
		return 1
	}
	if len(paths) == 0 {
		fmt.Printf("No config files in %s yet — one is created on first login.\n", env.Dirs.Config)
		return 0
	}

	failed := 0
	for _, path := range paths {
		s, res, err := config.Inspect(path)
		name := filepath.Base(path)

		if err != nil {
			fmt.Printf("%s: FAILED\n", name)
			for _, line := range strings.Split(err.Error(), "\n") {
				fmt.Printf("  ✗ %s\n", line)
			}
			failed++
			continue
		}

		label := s.Label
		if label == "" {
			label = "unlabelled"
		}
		fmt.Printf("%s (%s): OK\n", name, label)
		if res.Migrated {
			fmt.Println("  → pre-1.0 format; it will be migrated on next start (run with a backup)")
		}
		for _, note := range res.Notes {
			fmt.Printf("  · %s\n", note)
		}
		for _, w := range s.Warnings() {
			fmt.Printf("  ! %s\n", w)
		}
	}

	if failed > 0 {
		fmt.Printf("\n%d of %d config file(s) failed validation.\n", failed, len(paths))
		return 1
	}
	return 0
}

func configFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var out []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		out = append(out, filepath.Join(dir, e.Name()))
	}
	sort.Strings(out)
	return out, nil
}

func describeSolver(c config.CaptchaEnv) string {
	if !c.AutoSolveEnabled() {
		return "none (manual solving)"
	}
	return fmt.Sprintf("%s, %s timeout", c.Service, c.SolveTimeout)
}

func onOff(v bool) string {
	if v {
		return "on"
	}
	return "off"
}
