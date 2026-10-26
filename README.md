# claude-utils

Small utilities for [Claude Code](https://docs.claude.com/en/docs/claude-code).

## claude-statusline

A fast, compact status line for Claude Code.

![claude-statusline](claude-statusline/screenshot.png)

From left to right: context window usage, model, project and working directories, git branch and dirty file count, session duration, cost, plan rate limits (5-hour and 7-day), lines added and removed, and the current time.

```sh
go install github.com/collinvandyck/claude-utils/claude-statusline@latest
claude-statusline install --scope user
```

See [claude-statusline/README.md](claude-statusline/README.md) for configuration and details.

## Development

```sh
just test     # run the tests
just install  # install claude-statusline from this checkout
```
