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

## Starting a task

```
puffin new [-e] [prompt…]
```

makes a task for the repository you are standing in. What kind of task is the
repository's to say, in the config, not something to choose each time:

| The repository | What `puffin new` gives you |
|---|---|
| has `worktree: true` | a branch, a worktree, and an agent in it on the prompt |
| anything else | a session in the clone itself, worked on in place. One per clone: asking again goes to it |
| none (not in a repo) | `default_repo` from the config, or failing that an agent with no repo |

- `-r REPO` names a repository other than the one you are in. Repeat it for
  work across several: each gets a worktree side by side in the task's
  directory, and the session is one window with one agent at the root.
- `--bare` skips the worktree: an agent that can read the repository and
  nothing more, for a question that is not going to become a change.
- `-e` writes the prompt in `$EDITOR`.
- The task and its branch are named by a quick model from the prompt
  (`name_command`, Claude Haiku unless configured) following the
  instructions in `name_prompt` (workmux's `auto_name` system prompt by
  default), with any issue id in front. `--name` names it yourself; with no model, the prompt's first words.

Other ways in:

| Command | What you get |
|---|---|
| `puffin attach REPO` | Adds a worktree to the task you are in |
| `puffin pr N` | A task from an open PR: yours to carry on, or one to review (agent starts on `/code-review`) |
| `puffin adopt [PATH]` | Moves an existing worktree into a new task as it stands |
| `puffin here [PATH]` | A session in any directory, git or not |

Then: `puffin pick` (the list), `tasks [NAME]`, `switch NAME`, `rename`,
`close [--force]`, `sweep`, `reopen` (more to do on a task whose pull
requests all merged, even after a sweep). `puffin help` has the rest.

A repository's `panes` are laid out per type of task, `work` and `review`,
so a review can open on `puffin read`; a type left out uses `work`'s.
`read: hide:`
leaves out blocks named by the HTML comments around them, such as Cursor's
`CURSOR_SUMMARY` (the default).

`puffin read [PR]` is for sitting down to a review: the title, author,
status, branches, size, reviewers and labels, then the description rendered
as Markdown. In Ghostty or kitty its pictures are drawn in place; elsewhere,
or with `--links`, they are links and the text goes through the pager.

`puffin diff [PR]` shows the changes in [hunk](https://github.com/modem-dev/hunk).
A pull request you have reviewed opens on what its author changed since your
last review, not the whole of it again. `--since` starts somewhere else:
`base` (everything), `reviewed`, a version (`v1`, `v2`: what each force push
replaced), or any commit. `--pick` chooses from a list and `--versions`
prints it.

- A rebase in between does not bring in what landed on the base. The old
  version is replayed onto the new base in memory and the diff is against
  that, in a stack too, where the branch below was rewritten as well.
- Where the replay conflicts, those files carry conflict markers on the old
  side and so show how the author resolved them. puffin says when this happens.
- A push that changed nothing, such as a restack, says so and opens nothing.
- Notes written in hunk (`c`) are offered, when it closes, as comments on
  your pending review: a draft only you see until you submit it on GitHub or
  Graphite. GitHub only takes a comment on a line the pull request's whole
  diff shows, so any other note goes on the file with its line in the text.

`puffin approve [PR] [-m TEXT]` approves the pull request at the commit it
is on now. If you have a pending review, that is what is submitted, so the
notes from `puffin diff` go out with the approval.

`puffin stats [--days N]` reads the log of status changes: what merged and how
long it was open, how often CI went red and for how long, time spent with a
reviewer, reviews asked of you, and tasks started and finished. A pull request
is kept in the database after it closes, task or no task.

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

- **Config**: `~/.config/puffin/config.yaml`; see `config.example.yaml`.
  `roots` are scanned for clones, and each entry under `repos:` says
  whether the repository uses worktrees and what a new one needs (`copy`,
  `setup`, `pre_remove`, `panes`).
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
