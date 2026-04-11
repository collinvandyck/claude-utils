package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
)

var pprofCmd = &cobra.Command{
	Use:   "pprof [status.json]",
	Short: "Profile the statusline render with pprof",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		f, err := os.Open(args[0])
		if err != nil {
			return fmt.Errorf("open status json: %w", err)
		}
		defer f.Close()

		var status Status
		if err := json.NewDecoder(f).Decode(&status); err != nil {
			return fmt.Errorf("decode status json: %w", err)
		}

		// Override project dir if flag is set
		if dir, _ := cmd.Flags().GetString("dir"); dir != "" {
			status.Workspace.ProjectDir = dir
			status.Workspace.CurrentDir = dir
		}

		// CPU profile
		cpuFile, err := os.Create("/tmp/statusline-cpu.pprof")
		if err != nil {
			return err
		}
		defer cpuFile.Close()
		if err := pprof.StartCPUProfile(cpuFile); err != nil {
			return err
		}

		// Render
		ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
		defer cancel()

		state, err := newState(status)
		if err != nil {
			return err
		}
		defer func() { _ = state.close() }()

		renderer, err := newRenderer(status, state)
		if err != nil {
			return err
		}
		defer renderer.close()

		n, _ := cmd.Flags().GetInt("n")
		var result string
		for range n {
			result, err = renderer.render(ctx)
			if err != nil {
				return err
			}
		}
		fmt.Println(lipgloss.Sprint(result))

		// Stop CPU profile
		pprof.StopCPUProfile()

		// Snapshot profiles
		runtime.GC()
		for _, name := range []string{"heap", "allocs", "goroutine"} {
			f, err := os.Create(fmt.Sprintf("/tmp/statusline-%s.pprof", name))
			if err != nil {
				return err
			}
			defer f.Close()
			if err := pprof.Lookup(name).WriteTo(f, 0); err != nil {
				return err
			}
		}

		fmt.Fprintln(os.Stderr, "profiles written to /tmp/statusline-{cpu,heap,allocs,goroutine}.pprof")
		return nil
	},
}

func init() {
	pprofCmd.Flags().String("dir", "", "override project/current dir for git operations")
	pprofCmd.Flags().IntP("n", "n", 100, "number of render iterations")
}
