package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

type (
	component interface {
		render() (string, error)
	}
	componentFunc func() (string, error)
	line          struct {
		left  []component
		right []component
	}
	renderer struct {
		status  Status
		state   *state
		styles  styles
		gitInfo *gitInfo
		lines   []component
	}
	parts struct {
		vs []string
	}
	style  = lipgloss.Style
	styles struct {
		dflt         style
		dir          style
		error        style
		model        style
		session      style
		duration     style
		time         style
		git          gitStyles
		context      contextStyles
		cost         style
		rates        rateLimitStyles
		linesChanged linesChangedStyles
	}
	rateLimitStyles struct {
		low    style
		medium style
		high   style
	}
	contextStyles struct {
		empty  style
		low    style
		medium style
		high   style
	}
	gitStyles struct {
		branchGlyph    style
		noChangesGlyph style
		branch         style
		delta          style
	}
	linesChangedStyles struct {
		added   style
		removed style
	}
)

var (
	defaultStyles = styles{
		error:    lipgloss.NewStyle().Foreground(lipgloss.Red),
		model:    lipgloss.NewStyle().Bold(false).Foreground(lipgloss.Green),
		dir:      lipgloss.NewStyle().Foreground(lipgloss.Blue),
		duration: lipgloss.NewStyle().Foreground(lipgloss.Cyan).Faint(true),
		time:     lipgloss.NewStyle().Foreground(lipgloss.Cyan).Faint(true),
		cost:     lipgloss.NewStyle().Foreground(lipgloss.Green),
		git: gitStyles{
			branchGlyph:    lipgloss.NewStyle().Foreground(lipgloss.Green).Bold(true),
			noChangesGlyph: lipgloss.NewStyle().Foreground(lipgloss.Green).Bold(false),
			branch:         lipgloss.NewStyle().Foreground(lipgloss.Magenta).Bold(false),
			delta:          lipgloss.NewStyle().Foreground(lipgloss.Green).Bold(false),
		},
		context: contextStyles{
			empty:  lipgloss.NewStyle().Foreground(lipgloss.Green).Faint(true),
			low:    lipgloss.NewStyle().Foreground(lipgloss.Green).Faint(true),
			medium: lipgloss.NewStyle().Foreground(lipgloss.Yellow).Faint(false),
			high:   lipgloss.NewStyle().Foreground(lipgloss.Red).Faint(false),
		},
		rates: rateLimitStyles{
			low:    lipgloss.NewStyle().Foreground(lipgloss.Green).Faint(true),
			medium: lipgloss.NewStyle().Foreground(lipgloss.Yellow).Faint(false),
			high:   lipgloss.NewStyle().Foreground(lipgloss.Red).Faint(false),
		},
		linesChanged: linesChangedStyles{
			added:   lipgloss.NewStyle().Foreground(lipgloss.Green).Faint(true),
			removed: lipgloss.NewStyle().Foreground(lipgloss.Red).Faint(true),
		},
	}
)

func (f componentFunc) render() (string, error) { return f() }

func (l *line) render() (string, error) {
	var left parts
	for _, c := range l.left {
		if err := left.addErr(c.render()); err != nil {
			return "", err
		}
	}
	leftStr := left.join()
	if len(l.right) == 0 {
		return leftStr, nil
	}
	var right parts
	for _, c := range l.right {
		if err := right.addErr(c.render()); err != nil {
			return "", err
		}
	}
	rightStr := right.join()
	if rightStr == "" {
		return leftStr, nil
	}
	tw := termWidth()
	leftWidth := lipgloss.Width(leftStr)
	rightWidth := lipgloss.Width(rightStr)
	padding := 4 // account for Claude Code statusline chrome
	gap := tw - leftWidth - rightWidth - padding
	if gap < 1 {
		return leftStr, nil
	}
	return leftStr + strings.Repeat(" ", gap) + rightStr, nil
}

func termWidth() int {
	if s := os.Getenv("COLUMNS"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			return n
		}
	}
	if tty, err := os.Open("/dev/tty"); err == nil {
		defer tty.Close()
		if w, _, err := term.GetSize(int(tty.Fd())); err == nil && w > 0 {
			return w
		}
	}
	return 120
}

func newRenderer(status Status, state *state) (*renderer, error) {
	gitInfo, err := newGitInfo(status)
	if err != nil {
		return nil, err
	}
	res := &renderer{
		status:  status,
		state:   state,
		styles:  defaultStyles,
		gitInfo: gitInfo,
	}
	res.lines = []component{
		&line{
			left: []component{
				componentFunc(res.renderContext),
				componentFunc(res.renderModel),
				componentFunc(res.renderPath),
				componentFunc(res.renderBranch),
				componentFunc(res.renderDuration),
				componentFunc(res.renderCost),
				componentFunc(res.renderRateLimits),
				componentFunc(res.renderLinesChanged),
			},
			right: []component{
				componentFunc(res.renderTime),
			},
		},
	}
	return res, nil
}

func (r *renderer) render(_ context.Context) (string, error) {
	var p parts
	for _, l := range r.lines {
		if err := p.addErr(l.render()); err != nil {
			return "", err
		}
	}
	return p.joinSep("\n"), nil
}

func (r *renderer) renderLinesChanged() (string, error) {
	var parts parts
	la := r.renderLineChange("↑", r.status.Cost.TotalLinesAdded, r.styles.linesChanged.added)
	parts.add(la)
	lr := r.renderLineChange("↓", r.status.Cost.TotalLinesRemoved, r.styles.linesChanged.removed)
	parts.add(lr)
	return parts.joinErr()
}

func (r *renderer) renderLineChange(glyph string, delta int64, style lipgloss.Style) string {
	if delta == 0 {
		return ""
	}
	var amt = fmt.Sprintf("%d", delta)
	switch {
	case delta > 1000*1000:
		amt = fmt.Sprintf("%dM", delta/(1000*1000))
	case delta > 1000:
		amt = fmt.Sprintf("%dK", delta/1000)
	}
	return style.Render(fmt.Sprintf("%s%s", glyph, amt))
}

func (r *renderer) renderRateLimits() (string, error) {
	rl := r.status.RateLimits
	var parts parts
	parts.add(r.renderRateLimit(rl.FiveHour))
	parts.add(r.renderRateLimit(rl.SevenDay))
	res := parts.join()
	if res != "" {
		res = "rl " + res
	}
	return res, nil
}

func (r *renderer) renderRateLimit(limit WindowedRateLimits) string {
	var vertiBars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	if limit.UsedPercentage == 0 {
		return ""
	}
	idx := int(float64(len(vertiBars)) * (limit.UsedPercentage / 100.0))
	res := string(vertiBars[min(idx, len(vertiBars)-1)])
	if limit.UsedPercentage > 30 {
		res = res + fmt.Sprintf(" %d%%", int(limit.UsedPercentage))
	}
	var style lipgloss.Style
	switch {
	// TODO: define common constants for thresholds
	case limit.UsedPercentage > 70:
		style = r.styles.rates.high
	case limit.UsedPercentage > 30:
		style = r.styles.rates.medium
	default:
		style = r.styles.rates.low
	}
	res = style.Render(res)
	return res
}

func (r *renderer) renderCost() (string, error) {
	usd := r.status.Cost.TotalCostUSD
	if usd > 0 {
		return r.styles.cost.Render(fmt.Sprintf("$%0.2f", usd)), nil
	}
	return "", nil
}

func (r *renderer) renderContext() (string, error) {
	w := r.status.ContextWindow
	var build strings.Builder
	for i := range 10 {
		if w.UsedPercentage/10 >= i {
			var style lipgloss.Style
			switch {
			case w.UsedPercentage > 70:
				style = r.styles.context.high
			case w.UsedPercentage > 30:
				style = r.styles.context.medium
			default:
				style = r.styles.context.low
			}
			build.WriteString(style.Render("█"))
		} else {
			build.WriteString(r.styles.context.empty.Render("░"))
		}
	}
	var parts parts
	parts.add(build.String())
	parts.add(fmt.Sprintf("%d%%", w.UsedPercentage))
	return parts.joinErr()
}

func (r *renderer) renderDuration() (string, error) {
	m := int(r.state.sessionDuration().Minutes())
	var res string
	switch {
	case m < 60:
		res = fmt.Sprintf("%dm", m)
	case m < 60*24:
		res = fmt.Sprintf("%dh%dm", m/60, m%60)
	default:
		res = fmt.Sprintf("%dd%dh", m/1440, (m%1440)/60)
	}
	return r.styles.duration.Render(res), nil
}

func (r *renderer) renderTime() (string, error) {
	return r.styles.time.Render(time.Now().Format("15:04")), nil
}

func (r *renderer) renderModel() (string, error) {
	return r.styles.model.Render(r.status.Model.ID), nil
}

func (r *renderer) renderPath() (string, error) {
	dir := r.status.Workspace.CurrentDir
	if gitRoot := r.gitInfo.projectRoot(); gitRoot != "" {
		rel, _ := filepath.Rel(gitRoot, dir)
		if rel != "" {
			dir = filepath.Join(filepath.Base(gitRoot), rel)
		}
	}
	return r.styles.dir.Render(dir), nil
}

func (r *renderer) renderBranch() (string, error) {
	head, err := r.gitInfo.head()
	if err != nil {
		return "", err
	}
	var parts parts
	if head.name != "" {
		glyph := r.styles.git.branchGlyph.Render("⎇")
		switch {
		case head.isTag:
			glyph = r.styles.git.branchGlyph.Render("🔖")
		case head.isDetached:
			glyph = r.styles.git.branchGlyph.Render("🙄")
		}
		name := r.styles.git.branch.Render(head.name)
		if url := r.gitInfo.branchURL(head.name); url != "" {
			name = hyperlink(url, name)
		}
		name = glyph + " " + name
		parts.add(name)
	}
	if s := r.renderChangedFiles(); s != "" {
		parts.add(s)
	} else {
		ok := r.styles.git.noChangesGlyph.Render("✓")
		parts.add(ok)
	}

	return parts.joinErr()
}

func (r *renderer) renderChangedFiles() string {
	cf := r.cachedChangedFiles()
	if cf == nil {
		return ""
	}
	count := cf.dirtyCount()
	if count == 0 {
		return ""
	}
	s := fmt.Sprintf("±%d", count)
	return r.styles.git.delta.Render(s)
}

func (r *renderer) cachedChangedFiles() changedFiles {
	cached := &r.state.session.ChangedFiles
	if time.Since(cached.UpdatedAt) < 5*time.Second {
		return cached.Counts
	}
	changes, err := r.gitInfo.changes()
	if err != nil || changes == nil {
		return nil
	}
	cf := changes.changedFiles()
	cached.Counts = cf
	cached.UpdatedAt = time.Now()
	return cf
}

// renderDirtyCount uses the fast index-only path (no untracked detection).
// Currently unused — kept as an alternative to the full changes() path.
func (r *renderer) renderDirtyCount() string {
	stats, err := r.gitInfo.fastDirtyCount()
	if err != nil || stats.total() == 0 {
		return ""
	}
	s := fmt.Sprintf("±%d", stats.total())
	return r.styles.git.delta.Render(s)
}

func (r *renderer) close() error {
	return errors.Join(r.gitInfo.close())
}

func (p *parts) add(val string) {
	if val != "" {
		p.vs = append(p.vs, val)
	}
}

func (p *parts) addErr(val string, err error) error {
	if err != nil {
		return err
	}
	p.add(val)
	return nil
}

func (p *parts) join() string {
	return strings.Join(p.vs, " ")
}

func (p *parts) joinSep(sep string) string {
	return strings.Join(p.vs, sep)
}

func (p *parts) joinErr() (string, error) {
	return p.join(), nil
}
