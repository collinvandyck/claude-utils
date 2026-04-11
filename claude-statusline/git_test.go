package main

import "testing"

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
