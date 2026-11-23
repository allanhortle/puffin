# puffin

A tmux-first home for agentic development. tmux is where the work happens and
GitHub is the pipeline: the pull request is the unit of change, and puffin
keeps a local database that maps pull requests to worktrees, tmux sessions and
agents.

It replaces `prstatusd` (PR status in the tmux bar) and most of `workmux`
(worktrees and agent windows).

## The model

- **A task** is what you sit down to do. It owns one tmux session and one
  directory, and belongs to no repository.
- **A worktree** is mounted inside the task's directory, one per Graphite
  stack. A task can have none (an exploration), one, or several across repos.
- **A change** is a branch and, once it exists, its pull request. Stacks are
  not stored; they are recomputed from each PR's base and head.
- **The bottom PR of a stack speaks for the task.** Its letter, whether the
  task is waiting on you, and where it sorts all come from that one PR.
- **Live tmux state is never stored.** Sessions carry `@puffin_task`, windows
  carry `@puffin_worktree` and `@puffin_agent`, and the daemon reads them back.

One binary: `puffin daemon` is the only writer to the SQLite database, and
every other command is a thin client talking to it over a unix socket.

## Kinds of task

| Start it with | What you get |
|---|---|
| `puffin new [--read REPO] prompt…` | Exploration: an agent at the task root, no branch |
| `puffin new --attach REPO prompt…` | Work: a branch, a worktree, an agent in it. `-e` writes the prompt in `$EDITOR`; `REPO` can be `.` |
| `puffin attach REPO` | Adds a worktree to the task you are in |
| `puffin pr N` | A task from an open PR: yours to carry on, or one to review (agent starts on `/code-review`) |
| `puffin adopt [PATH]` | Moves an existing worktree into a new task as it stands |
| `puffin here [PATH]` | In place: a session and agent in a directory puffin does not own. No worktree, no PR tracking |

Then: `puffin pick` (the list), `tasks [NAME]`, `switch NAME`, `rename`,
`close [--force]`, `sweep`. `puffin help` has the rest.

## Reading the list and the status bar

`prefix Space` opens `puffin pick`. A row is: `▲` · status · agent · name.

- `▲` marks a row waiting on you. The `▲N` in the status bar is the number of
  marked rows.
- The word is where the bottom PR of the task's stack stands. Your own:
  `ready`, `draft`, `building`, `queued`, `waiting` (out with a reviewer),
  `blocked` (held up on you), `failing`, `conflict`, `changes` (requested),
  `merged`, `closed`. Someone else's: `review` (asked of you), `updated` (the
  author pushed since you reviewed), `reviewed` (back with them). With no PR:
  `local`, `explore`, or `here` for a task in place. A red number after it is
  unresolved threads; `+N` is the rest of the stack.
- Agent: `~` working, `?` asking you something, `•` finished its turn.

A row is yours when its bottom PR is failing, conflicted, has changes
requested, is waiting on you, or has unresolved threads once its build has
settled; when a review is asked of you and no agent is working on it; or when
its agent is asking you something. Building or queued is the machine's move.

The left of the status bar shows the same word and agent icon in front of
the session name. Windows outside any task still get a single letter on their
tab (`G D B Q W F C R M X`), where there is no room for a word.

## Setup

```
go build -o ~/.local/bin/puffin ./cmd/puffin
puffin start
```

- **Config**: `~/.config/puffin/config.toml`; see `config.example.toml`.
  `roots` are scanned for clones, and each `[repo."owner/name"]` section says
  what a new worktree needs (`copy`, `setup`, `pre_remove`, `panes`).
- **tmux**: start the daemon from `.tmux.conf`, bind `puffin pick` to a key,
  and use `@puffin_status`, `@puffin_attention`, `@prreview` and `@pr` in the
  status formats.
- **Agent state**: Claude Code hooks call `puffin agent working|waiting|done|off`
  on `UserPromptSubmit`/`PostToolUse`, `Notification`, `Stop` and `SessionEnd`.
- **Auth**: puffin uses `gh auth token`; it has no credentials of its own.

Data lives in `~/.local/share/puffin/` (database, task directories) and
`~/.local/state/puffin/` (socket, lock, log).

## Developing

`go test ./...` covers status derivation, stacks, ranking and the store. To
run the whole thing without touching the live tmux server or database, point
it at throwaway locations: relative `XDG_STATE_HOME`, `XDG_DATA_HOME` and
`XDG_CONFIG_HOME` (the socket path must stay under 104 bytes), a config with
`tmux_socket = "something"`, and `GH_CONFIG_DIR=~/.config/gh` so `gh` still
finds its login.

What is not done is in [TODO.md](TODO.md).
