# Outstanding

Written 2026-11-23, at the end of the session that built puffin.

## Loose ends outside this repo

- **Dotfiles are uncommitted.** `.tmux.conf`,
  `.local/bin/ctx-fzf.mjs` and the new `.config/puffin/config.toml` all carry
  puffin changes.
- **`~/.claude/settings.json` was edited by hand**: workmux's four status hooks
  removed, puffin's five added. There is no `puffin` command that installs
  them, so a new machine needs them added again.
- **`wmt`/`wmn` in `.zshrc` still run workmux.** The replacement is
  `puffin new -e`.
- **`prstatusd` and `prstatusd.mjs` are still in dotfiles**, unused, and
  `~/.cache/prstatus/` is stale. `prstatusd.mjs` had uncommitted edits, which
  is why it was not deleted.
- **The GitHub repo still has the old project's branches and tags**
  (`develop`, `feature/*`, 30 tags) and its old description.

## Still on workmux

- Existing workmux worktrees are legacy by decision: move one over with
  `puffin adopt PATH` when picking it back up.

## Known gaps

- **Replies without a push are invisible.** A reviewed PR only comes back when
  the author re-requests review or its head commit changes.
- **A restack counts as a push**, so `new` can appear with nothing changed.
- **PRs from forks cannot be reviewed** as tasks.
- **A leftover local branch holds a task open.** The branch a task started on,
  if it never got a PR, stops the task going `done` until it is deleted.
- **`puffin open N` and the click range carry only a number**, which is
  ambiguous across repositories. Your own PR wins.
- **Rebuilding a session drops a root agent**: a task that began as an
  exploration and later had a repo attached comes back with only the worktree
  window.
- **Agents restart without their conversation** when a session is rebuilt, and
  `adopt` loses the old worktree's conversation, because Claude keys history
  by directory.
- **Names are word-chopped**, not model-generated as workmux's `auto_name` was,
  and an explicit `--name` or `rename` is cut to four words like any other.
- **`-r` of the same repo twice in one `new` fails**: both want the same
  branch.
- **A task across several repos never runs a repo's `setup` commands or pane
  layout**, because its worktrees get no windows. Files are still copied.
- **`puffin stats` knows nothing from before 2026-11-26 about pull requests
  that had no task.** Their rows were deleted then, and their events point at
  nothing. Reviews in particular start from that date.
- **A review counts as handed back when it stops being asked for**, which is
  also what a merge without me looks like.
- **A branch name used twice overwrites the first pull request's row**, so the
  log keeps only the later one.
- **GitHub's mergeability flickers to UNKNOWN and back** many times a day and
  each is logged. `stats` ignores them; the event table still fills with them.
- **`puffin read` smuggles its pictures past tmux.** tmux 3.8 does not know
  Kitty images, so they go through `allow-passthrough` straight to Ghostty:
  they are not kept inside the pane, a redraw or a window switch loses them,
  and they do not scroll back. tmux master has Kitty images (from 2026-10-29,
  built with `--enable-images`); once a release has them `read` writes them
  plainly and tmux keeps them, with no change here. Then `allow-passthrough`
  can come out of `.tmux.conf`. Pictures also mean no pager.
- **No notifications.** The only signals are the list and the status bar.
- **The left status needs the daemon.** With it stopped, the status bar stops
  updating.

## Not built

- Browsing every open PR in the org, and viewing a PR's diff without a
  worktree. Both went with `prq`, which was removed on 2026-11-23.
  `puffin pr N` only knows your own PRs and your review requests.
- `tmux -C` control mode. Windows outside a task are still matched to PRs by
  polling pane directories, as prstatusd did.
- The `agent_run` table is in the schema and unused: agent state lives in tmux
  options. Either use it for history or drop it.
- A way to keep a per-window agent icon on tabs for sessions with several
  agents.
- Whether `▲` (waiting on you) and `•` (review requests) should become one
  count was raised and left undecided. They overlap: a re-requested review
  with a task counts in both.

## Not yet seen on real work

- The review states after the first: author pushed, re-requested, merged.
  Unit tested only.
- `puffin adopt` on a real monorepo worktree, in particular whether
  `node_modules` survives the move.
- Clicking a letter or the review count in the status bar.
