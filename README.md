# puffin

puffin is a PR-first workflow for tmux.

Instead of organising your work around agents, puffin organises it around
pull requests: the ones you created, and the ones you have been asked to
review. It gives you a unified view of all the things you need to pay
attention to, and lets you jump quickly to the next thing that requires your
attention.

When you create a new task, puffin gives it a tmux session, git worktrees and
an agent, and keeps track of its state all the way from draft to merged.

When someone requests a review from you, puffin adds it to your task list
just like your other work, so it's ready for you to explore the code or ask
an agent about.

## Starting a task

```
puffin new [-e] [prompt…]
```

This makes a task for the repository you are standing in. The repository's
entry in the config decides what kind of task you get:

| The repository                     | What `puffin new` gives you                                                                                 |
| ---------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| has `worktree: true`               | A branch, a worktree, and an agent in it working on the prompt.                                             |
| anything else                      | A session in the clone itself, worked on in place. There is one per clone, so asking again takes you to it. |
| none (you are not in a repository) | A task for `default_repo` from the config. Without that, an agent with no repository.                       |

Options:

- `-r REPO` names a repository other than the one you are in. Repeat it for
  work across several. Each gets a worktree in the task's directory, and the
  session is one window with one agent at the root.
- `--bare` skips the worktree. The agent can read the repository and nothing
  more, which suits a question that will not become a change.
- `-e` opens `$EDITOR` to write the prompt.
- `--name NAME` names the task yourself.

Without `--name`, a quick model names the task and its branch from the prompt,
with any issue id in front. The model is `name_command` in the config (Claude
Haiku by default) and its instructions are `name_prompt` (workmux's
`auto_name` system prompt by default). With no model, the name is the prompt's
first words.

Other ways to start:

| Command               | What you get                                                                             |
| --------------------- | ---------------------------------------------------------------------------------------- |
| `puffin attach REPO`  | Another worktree in the task you are in, with its own window and agent.                  |
| `puffin pr N`         | A task for an open pull request: your own to carry on with, or someone else's to review. |
| `puffin adopt [PATH]` | A new task made from an existing worktree, which is moved into it as it stands.          |
| `puffin here [PATH]`  | A session in any directory, git or not. Closing it leaves the directory alone.           |

## Switching tasks

At the heart of the puffin workflow is a fuzzy finder for your current work. It's a list of your open prs and reviews.
The list is sorted by state, and the things you need to act on now are at the top.

`prefix Space` opens `puffin pick`, to show the list of tasks and reviews.

- **`▲`** marks a row that is waiting on you.
- **Agent states** are `~` while it working, `?` when it is asking you something,
  and `•` when it has finished its turn.

The status words:

| Whose           | Word               | Meaning                                         |
| --------------- | ------------------ | ----------------------------------------------- |
| Yours           | `ready`            | Nothing is in the way of merging.               |
|                 | `draft`            | Ready, and still a draft.                       |
|                 | `building`         | Checks are running.                             |
|                 | `queued`           | In the merge queue.                             |
|                 | `blocked`          | Held up on you.                                 |
|                 | `failing`          | A check failed.                                 |
|                 | `conflict`         | It conflicts with its base.                     |
|                 | `review`           | A review is asked of you.                       |
|                 | `changes`          | A reviewer requested changes.                   |
| Someone else's  | `review`           | A review is asked of you.                       |
|                 | `waiting`          | Out with a reviewer.                            |
|                 | `updated`          | The author pushed since you reviewed.           |
|                 | `reviewed`         | You reviewed it and it is back with the author. |
| No pull request | `local`            | A branch that has no pull request yet.          |
|                 | `explore`          | A task with no worktree.                        |
|                 | `here`             | Adhoc tasks that have no syncronised state      |
|                 | `merged`, `closed` | Finished.                                       |

A row is waiting on you in three cases:

- Its bottom pull request is failing, conflicted, blocked on you, or has changes requested.
- A review is asked of you and no agent is working on it.
- Its agent is asking you something.

## Reviewing a pull request

`puffin pr N` on someone else's pull request checks out its branch in a new
task and starts an agent on `review_prompt` (`/code-review` by default). The
commands below take a number or a link. With nothing, they use the pull
request of the branch you are on.

### `puffin read [PR]`

Shows what to know before the diff: title, author, status, branches, size,
reviewers and labels, then the description rendered as Markdown. Ghostty and
kitty draw the description's pictures in place. Other terminals, and
`--links`, show them as links and send the text through the pager.

### `puffin comments [PR]`

Lists what has been said:

1. The conversation under the pull request, with each review's verdict.
2. Every thread on the diff, with the lines it was started on and whether it
   is resolved.

`--unresolved` keeps only the open threads. Comments that bots post under the
pull request, and Graphite's stack comment, are left out unless you pass
`--all`. A bot's threads on the diff always stay, because that is where
review bots report what they found.

### `puffin diff [PR]`

Opens the changes in [hunk](https://github.com/modem-dev/hunk). If you have
reviewed the pull request, it shows only what the author changed since your
last review. Otherwise it shows the whole change.

`--since` starts from somewhere else:

| Value         | Diffs from                                             |
| ------------- | ------------------------------------------------------ |
| `base`        | The base branch, which is the whole change.            |
| `reviewed`    | The commit your last review was made on.               |
| `v1`, `v2`, … | A version: what the branch was before each force push. |
| a commit      | That commit.                                           |

`--pick` chooses the starting point from a list, and `--versions` prints the
list.

What the diff leaves out:

- **Changes from the base.** If the branch was rebased, or the base was merged
  into it, a plain diff would also show everything that landed on the base in
  between. puffin replays the old version onto the new base in memory and
  diffs against the result. This works in a stack too, where the branch below
  was rewritten as well.
- **Pushes that changed nothing.** After a restack, puffin says nothing
  changed and opens nothing.

If the replay conflicts, puffin says so. The conflicted files carry conflict
markers on the old side, so their diff shows how the author resolved the
conflict.

Notes you write in hunk (press `c`) are offered as comments on your pending
review when hunk closes. A pending review is a draft that only you can see
until you submit it.

### `puffin approve [PR] [-m TEXT]`

Approves the pull request at the commit it is on now. If you have a pending
review, puffin submits that, so the notes from `puffin diff` go out with the
approval.

## Managing tasks

| Command                    | What it does                                                                                                                    |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `puffin tasks [NAME]`      | Prints every task, or one, with its worktrees and pull requests.                                                                |
| `puffin switch NAME`       | Goes to a task's session, rebuilding it if it has gone.                                                                         |
| `puffin rename [NAME] NEW` | Renames a task and its session. The directory and branches keep their names.                                                    |
| `puffin close [NAME]`      | Removes a task's worktrees, session and directory. A worktree with uncommitted work stops it unless you pass `--force`.         |
| `puffin sweep`             | Closes every task whose pull requests have all merged or closed.                                                                |
| `puffin reopen [NAME]`     | Keeps a task open after its pull requests have merged, because there is more to do. A swept task comes back with its worktrees. |

`puffin help` lists every command.

## The daemon

puffin is one binary. `puffin daemon` runs in the background and does the
watching:

- It asks GitHub about your pull requests and the reviews asked of you. It
  checks GitHub's notification feed about once a minute and refetches when
  something moved. It also refetches every 5 minutes regardless, or every 30
  seconds while a build is running or a pull request is in the merge queue.
- It reads tmux to see which sessions and windows belong to which task, and
  writes the status words and counts back to the status bar.
- It is the only writer to the SQLite database. Every other command is a
  client that talks to it over a unix socket.

| Command          | What it does                                                  |
| ---------------- | ------------------------------------------------------------- |
| `puffin start`   | Starts the daemon in the background, if it is not running.    |
| `puffin stop`    | Stops it and clears the status bar.                           |
| `puffin restart` | Stops and starts it. Do this after rebuilding.                |
| `puffin toggle`  | Stops it if it is running, starts it if it is not.            |
| `puffin status`  | Shows whether it is running, and when it last fetched.        |
| `puffin refresh` | Runs one poll now. `--force` refetches even if nothing moved. |
| `puffin daemon`  | Runs it in the foreground.                                    |

## Setup

```
go build -o ~/.local/bin/puffin ./cmd/puffin
puffin start
```

puffin has no credentials of its own. It uses the token from `gh auth token`.

### Config

The config is `~/.config/puffin/config.yaml`. `config.example.yaml` shows
every setting.

- `roots` are the directories puffin scans for clones.
- Each entry under `repos` says whether the repository uses worktrees, and
  what a new worktree needs: files to `copy`, commands to run for `setup` and
  `pre_remove`, and the `panes` to lay out.
- `panes` are set per type of task, `work` and `review`, so a review can open
  on `puffin read`. A type you leave out uses the `work` layout.
- `read: hide:` is a list of regular expressions. Whatever they match is cut
  from descriptions and comments before `puffin read` and `puffin comments`
  draw them. It has no default. The example hides Cursor's summary block and
  the buttons and signature under each Cursor Bugbot comment.

### tmux

In `.tmux.conf`:

- Start the daemon with `puffin start`.
- Bind `puffin pick` to a key.
- Use `@puffin_status`, `@puffin_attention`, `@prreview` and `@pr` in the
  status formats.

### Agent state

Claude Code hooks tell puffin what the agent in a pane is doing by calling
`puffin agent STATE`:

| Hook                              | State     |
| --------------------------------- | --------- |
| `UserPromptSubmit`, `PostToolUse` | `working` |
| `Notification`                    | `waiting` |
| `Stop`                            | `done`    |
| `SessionEnd`                      | `off`     |

### Where data lives

| Directory                | Contents                               |
| ------------------------ | -------------------------------------- |
| `~/.local/share/puffin/` | The database and the task directories. |
| `~/.local/state/puffin/` | The socket, lock and log.              |
