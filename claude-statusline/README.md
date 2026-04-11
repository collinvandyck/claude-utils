# claude-statusline

A fast compact status line for claude code that tries to be fast. 

Includes:
- context window graph
- current model
- git status
- cost
- plan rate limits
- session duration
- lines changed

## Install

```sh
go install github.com/collinvandyck/claude-utils/claude-statusline@latest
claude-statusline install --scope 'user|project'
```

This will install the statusline with a configuration like this:

```shell
cat ~/.claude/settings.json | jq .statusLine
{
  "command": "claude-statusline",
  "refreshInterval": 1,
  "type": "command"
}
```
It by design installs with a fast refresh interval. More expensive operations like git changed files only run every 5s.


```sh
claude-statusline install --scope project
```

If you already have a `statusLine` configured, pass `--force` to overwrite it.

## General

Each bit of information comes from a 'component'. If the data is not available for a component (e.g. plan rate limits when you are using the api) they are not rendered. Eventually the components will have names and will be able to be ordered.

The terminal rendering is provided by https://github.com/charmbracelet/lipgloss. The statusline is built to support themes, although it only ships with a default one that I liked.

If you're using a modern terminal emulator, you should see that the branch name is clickable (opens the branch on github). This uses OSC 8 hyperlinks. If you're on an older terminal I'm not sure what your experience will be. I plan to add more clickable links in the future.

## Example

Claude code statuslines consume JSON from stdin and render the statusline to stdout, which claude then displays. This means that you can test it out by running against the example JSON that the Anthropic docs provide:

```sh
claude-statusline < claude-statusline/example.json
█░░░░░░░░░ 8% claude-opus-4-6 /current/working/directory ✓ 0m $0.01 rl ▂ ▄ 41% ↑156 ↓23  09:20 
```

## Future

- Command line flags or config that controls which components are rendered, in what order, and so on.