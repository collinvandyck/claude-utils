package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func init() {
	installCmd.Flags().String("scope", "user", "where should the statusline be installed (user|project)")
	installCmd.Flags().Bool("force", false, "overwrite existing statusLine config")
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the statusline",
	RunE: func(cmd *cobra.Command, args []string) error {
		scope, _ := cmd.Flags().GetString("scope")
		force, _ := cmd.Flags().GetBool("force")

		var paths []string
		switch scope {
		case "user":
			hd, err := os.UserHomeDir()
			if err != nil {
				return err
			}
			paths = append(paths, filepath.Join(hd, ".claude", "settings.local.json"))
			paths = append(paths, filepath.Join(hd, ".claude", "settings.json"))
		case "project":
			paths = append(paths, filepath.Join(".claude", "settings.local.json"))
			paths = append(paths, filepath.Join(".claude", "settings.json"))
		default:
			return fmt.Errorf("unsupported scope: %v", scope)
		}

		// find the first path that exists, or use the first path
		var target string
		for _, p := range paths {
			if _, err := os.Stat(p); err == nil {
				target = p
				break
			}
		}
		if target == "" {
			return errors.New("no claude settings found")
		}

		// read existing settings or start fresh
		data := map[string]any{}
		bs, err := os.ReadFile(target)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("read %s: %w", target, err)
		}
		if len(bs) > 0 {
			if err := json.Unmarshal(bs, &data); err != nil {
				return fmt.Errorf("parse %s: %w", target, err)
			}
		}

		// check for existing statusLine
		if _, exists := data["statusLine"]; exists && !force {
			return fmt.Errorf("statusLine already configured in %s (use --force to overwrite)", target)
		}

		// set the statusLine config
		data["statusLine"] = map[string]any{
			"type":            "command",
			"command":         "claude-statusline",
			"refreshInterval": 1,
		}

		// write back
		out, err := json.MarshalIndent(data, "", "  ")
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(target, append(out, '\n'), 0644); err != nil {
			return err
		}

		fmt.Printf("statusLine configured in %s\n", target)
		return nil
	},
}
