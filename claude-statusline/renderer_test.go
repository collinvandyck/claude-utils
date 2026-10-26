package main

import "testing"

// TestDisplayPath checks git-relative display, /private trimming, and abbreviation of long paths.
func TestDisplayPath(t *testing.T) {
	tests := []struct {
		name    string
		gitRoot string
		path    string
		want    string
	}{
		{
			name:    "inside git root",
			gitRoot: "/src/saas-components",
			path:    "/src/saas-components/charts/prom",
			want:    "saas-components/charts/prom",
		},
		{
			name:    "git root itself",
			gitRoot: "/src/saas-components",
			path:    "/src/saas-components",
			want:    "saas-components",
		},
		{
			name:    "outside git root does not climb with dot-dots",
			gitRoot: "/src/saas-components",
			path:    "/opt/scratch",
			want:    "/opt/scratch",
		},
		{
			name:    "sibling sharing a name prefix is outside the root",
			gitRoot: "/src/repo",
			path:    "/src/repo-other/x",
			want:    "/src/repo-other/x",
		},
		{
			name: "private tmp is shown as tmp",
			path: "/private/tmp/foo",
			want: "/tmp/foo",
		},
		{
			name: "long tmp path keeps root and last two dirs",
			path: "/private/tmp/claude-501/-Users-collin-code-temporal/956bfd4d-80b1-430a-ac15-1c11997668de/scratchpad/sc-3512",
			want: "/tmp/.../scratchpad/sc-3512",
		},
		{
			name:    "long git-relative path keeps repo and last two dirs",
			gitRoot: "/src/saas-components",
			path:    "/src/saas-components/deploy/charts/temporal/templates/prometheus/rules",
			want:    "saas-components/.../prometheus/rules",
		},
		{
			name: "long path with too few segments is left alone",
			path: "/a-very-long-directory-name/another-very-long-directory-name",
			want: "/a-very-long-directory-name/another-very-long-directory-name",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := displayPath(tt.gitRoot, tt.path)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("displayPath(%q, %q) = %q, want %q", tt.gitRoot, tt.path, got, tt.want)
			}
		})
	}
}
