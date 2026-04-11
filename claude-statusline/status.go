package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/spf13/cobra"
)

type (
	Status struct {
		Dir               string        `json:"cwd"`
		SessionID         string        `json:"session_id"`
		SessionName       string        `json:"session_name"`
		TranscriptPath    string        `json:"transcript_path"`
		Model             Model         `json:"model"`
		Workspace         Workspace     `json:"workspace"`
		Version           string        `json:"version"`
		OutputStyle       OutputStyle   `json:"output_style"`
		Cost              Cost          `json:"cost"`
		ContextWindow     ContextWindow `json:"context_window"`
		Exceeds200KTokens bool          `json:"exceeds_200k_tokens"`
		RateLimits        RateLimits    `json:"rate_limits"`
		Vim               Vim           `json:"vim"`
		Agent             Agent         `json:"agent"`
		Worktree          Worktree      `json:"worktree"`
	}
	Model struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	}
	Workspace struct {
		CurrentDir  string   `json:"current_dir"`
		ProjectDir  string   `json:"project_dir"`
		AddedDirs   []string `json:"added_dirs"`
		GitWorktree string   `json:"git_worktree"`
	}
	OutputStyle struct {
		Name string `json:"name"`
	}
	Cost struct {
		TotalCostUSD       float64 `json:"total_cost_usd"`
		TotalDurationMs    int64   `json:"total_duration_ms"`
		TotalAPIDurationMs int64   `json:"total_api_duration_ms"`
		TotalLinesAdded    int64   `json:"total_lines_added"`
		TotalLinesRemoved  int64   `json:"total_lines_removed"`
	}
	ContextWindow struct {
		TotalInputTokens    int64        `json:"total_input_tokens"`
		TotalOutputTokens   int64        `json:"total_output_tokens"`
		ContextWindowSize   int64        `json:"context_window_size"`
		UsedPercentage      int          `json:"used_percentage"`
		RemainingPercentage int          `json:"remaining_percentage"`
		CurrentUsage        CurrentUsage `json:"current_usage"`
	}
	CurrentUsage struct {
		InputTokens              int64 `json:"input_tokens"`
		OutputTokens             int64 `json:"output_tokens"`
		CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	}
	RateLimits struct {
		FiveHour WindowedRateLimits `json:"five_hour"`
		SevenDay WindowedRateLimits `json:"seven_day"`
	}
	WindowedRateLimits struct {
		UsedPercentage float64 `json:"used_percentage"`
		ResetsAtEpoch  int64   `json:"resets_at"`
	}
	Vim struct {
		Mode string `json:"mode"`
	}
	Agent struct {
		Name string `json:"name"`
	}
	Worktree struct {
		Name           string `json:"name"`
		Path           string `json:"path"`
		Branch         string `json:"branch"`
		OriginalCWD    string `json:"original_cwd"`
		OriginalBranch string `json:"original_branch"`
	}
)

func init() {
	lipgloss.Writer.Profile = colorprofile.ANSI256
}

func main() {
	cmd := cobra.Command{
		SilenceUsage: true,
	}
	cmd.AddCommand(pprofCmd, installCmd)
	cmd.Run = func(cmd *cobra.Command, _ []string) {
		result, err := render(cmd.Context())
		if err != nil {
			result = lipgloss.NewStyle().Foreground(lipgloss.Red).Render("error: " + err.Error())
		}
		fmt.Println(lipgloss.Sprint(result))
	}
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func render(ctx context.Context) (rendered string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	var status Status
	if err := json.NewDecoder(os.Stdin).Decode(&status); err != nil {
		return "", err
	}
	state, err := newState(status)
	if err != nil {
		return "", err
	}
	defer func() { _ = state.close() }()
	renderer, err := newRenderer(status, state)
	if err != nil {
		return "", err
	}
	defer renderer.close()
	return renderer.render(ctx)
}
