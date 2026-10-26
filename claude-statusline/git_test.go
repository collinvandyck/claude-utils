package main

import (
	"os/exec"
	"testing"
)

func TestHeadUnbornBranch(t *testing.T) {
	dir := t.TempDir()
	if out, err := exec.Command("git", "init", "-q", "-b", "main", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
	var status Status
	status.Workspace.CurrentDir = dir
	status.Workspace.ProjectDir = dir
	g, err := newGitInfo(status)
	if err != nil {
		t.Fatal(err)
	}
	defer g.close()
	got, err := g.head()
	if err != nil {
		t.Fatalf("head() error: %v", err)
	}
	if !got.isBranch || got.name != "main" {
		t.Errorf("head() = %+v, want branch main", got)
	}
}

func TestGitRemoteToHTTPS(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "standard ssh",
			raw:  "git@github.com:owner/repo.git",
			want: "https://github.com/owner/repo",
		},
		{
			name: "org credential ssh",
			raw:  "org-56493103@github.com:temporalio/temporal.git",
			want: "https://github.com/temporalio/temporal",
		},
		{
			name: "https with .git",
			raw:  "https://github.com/owner/repo.git",
			want: "https://github.com/owner/repo",
		},
		{
			name: "https without .git",
			raw:  "https://github.com/owner/repo",
			want: "https://github.com/owner/repo",
		},
		{
			name: "ssh with custom user",
			raw:  "deploy@gitlab.com:org/project.git",
			want: "https://gitlab.com/org/project",
		},
		{
			name: "nested path",
			raw:  "git@github.com:org/sub/repo.git",
			want: "https://github.com/org/sub/repo",
		},
		{
			name: "no .git suffix",
			raw:  "git@github.com:owner/repo",
			want: "https://github.com/owner/repo",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gitRemoteToHTTPS(tt.raw)
			if got != tt.want {
				t.Errorf("gitRemoteToHTTPS(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
