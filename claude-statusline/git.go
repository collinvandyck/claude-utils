package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-billy/v6/osfs"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing/format/gitignore"
)

type (
	gitInfo struct {
		status   Status
		repo     *git.Repository
		worktree *git.Worktree
	}
	headInfo struct {
		isBranch   bool
		isTag      bool
		isDetached bool
		name       string
	}
	gitChanges struct {
		Status git.Status
	}
)

func newGitInfo(status Status) (*gitInfo, error) {
	repo, err := git.PlainOpenWithOptions(status.Workspace.ProjectDir, &git.PlainOpenOptions{DetectDotGit: true})
	res := &gitInfo{status: status, repo: repo}
	if err != nil {
		return res, nil // not a git repo
	}
	res.worktree, err = repo.Worktree()
	if err != nil {
		return nil, err
	}
	fs := osfs.New("/")
	if global, err := gitignore.LoadGlobalPatterns(fs); err == nil {
		res.worktree.Excludes = append(res.worktree.Excludes, global...)
	}
	if system, err := gitignore.LoadSystemPatterns(fs); err == nil {
		res.worktree.Excludes = append(res.worktree.Excludes, system...)
	}
	return res, nil
}

func (g *gitInfo) changes() (*gitChanges, error) {
	if g.worktree != nil {
		status, err := g.worktree.StatusWithOptions(git.StatusOptions{Strategy: git.Empty})
		if err != nil {
			return nil, err
		}
		changes := &gitChanges{Status: status}
		return changes, nil
	}
	return nil, nil
}

func (g *gitInfo) close() error {
	if g.repo != nil {
		return g.repo.Close()
	}
	return nil
}

func (g *gitInfo) projectRoot() string {
	if g.worktree == nil {
		return ""
	}
	return g.worktree.Filesystem.Root()
}

func (g *gitInfo) head() (info headInfo, err error) {
	if g.repo != nil {
		ref, err := g.repo.Head()
		if err != nil {
			return info, err
		}
		info.name = ref.Name().Short()
		switch {
		case ref.Name().IsBranch():
			info.isBranch = true
		case ref.Name().IsTag():
			info.isTag = true
		default:
			info.name = ref.Hash().String()
			info.name = info.name[:min(len(info.name), 8)]
			info.isDetached = true
		}
	}
	return info, nil
}

func hyperlink(url, text string) string {
	return fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", url, text)
}

// remoteURL returns the base web URL for the origin remote (e.g. "https://github.com/owner/repo").
// Returns empty string if not available.
func (g *gitInfo) remoteURL() string {
	if g.repo == nil {
		return ""
	}
	remote, err := g.repo.Remote("origin")
	if err != nil {
		return ""
	}
	urls := remote.Config().URLs
	if len(urls) == 0 {
		return ""
	}
	return gitRemoteToHTTPS(urls[0])
}

// gitRemoteToHTTPS converts a git remote URL to an HTTPS web URL.
// Handles ssh (user@host:owner/repo.git) and https URLs.
func gitRemoteToHTTPS(raw string) string {
	// user@github.com:owner/repo.git — any SSH-style URL with @...:.../
	if idx := strings.Index(raw, "@"); idx >= 0 {
		if colonIdx := strings.Index(raw[idx:], ":"); colonIdx >= 0 {
			host := raw[idx+1 : idx+colonIdx]
			path := raw[idx+colonIdx+1:]
			raw = "https://" + host + "/" + path
		}
	}
	raw = strings.TrimSuffix(raw, ".git")
	return raw
}

// branchURL returns a web URL for the given branch/ref name, or empty string.
func (g *gitInfo) branchURL(name string) string {
	base := g.remoteURL()
	if base == "" {
		return ""
	}
	return base + "/tree/" + name
}

type dirtyStats struct {
	modified int
	deleted  int
}

func (d dirtyStats) total() int {
	return d.modified + d.deleted
}

func (g *gitInfo) fastDirtyCount() (dirtyStats, error) {
	if g.repo == nil {
		return dirtyStats{}, nil
	}
	idx, err := g.repo.Storer.Index()
	if err != nil {
		return dirtyStats{}, err
	}
	rootDir := g.projectRoot()
	var stats dirtyStats
	for _, entry := range idx.Entries {
		fi, err := os.Lstat(filepath.Join(rootDir, entry.Name))
		if err != nil {
			if os.IsNotExist(err) {
				stats.deleted++
			}
			continue
		}
		if entry.Size != uint32(fi.Size()) {
			stats.modified++
			continue
		}
		if !entry.ModifiedAt.Equal(fi.ModTime()) {
			stats.modified++
			continue
		}
		if fi.Mode().IsRegular() != entry.Mode.IsFile() {
			stats.modified++
			continue
		}
	}
	return stats, nil
}

type changedFiles map[git.StatusCode]int

func (g *gitChanges) changedFiles() (res changedFiles) {
	res = make(changedFiles)
	for _, fs := range g.Status {
		sc := fs.Worktree
		if sc == git.Unmodified {
			continue
		}
		res[sc] += 1
	}
	return
}

func (cf changedFiles) dirtyCount() (res int) {
	for k, v := range cf {
		if k == git.Unmodified {
			continue
		}
		res += v
	}
	return
}

func (cf changedFiles) untracked() int {
	return cf[git.Untracked]
}

func (cf changedFiles) modified() int {
	return cf[git.Modified]
}

func (cf changedFiles) added() int {
	return cf[git.Added]
}

func (cf changedFiles) deleted() int {
	return cf[git.Deleted]
}
