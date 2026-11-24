# A GUI for puffin

Notes from 2026-11-24. Nothing here is built. The question was how to give
puffin a window, for people who do not much like the terminal, with the least
effort.

## Where it landed

A thin window over what puffin already is:

- **tmux stays as the engine, hidden.** No prefix key, no status bar. Sessions,
  panes, agents and persistence work as they do now, and the daemon needs no
  second backend.
- **The window is two things**: a sidebar that renders the list from the puffin
  socket (the data behind `puffin pick`), and a terminal view.
- **Start with Go + Wails + xterm.js.** It stays in this module, the design is
  plain HTML and CSS, and it runs on macOS, Linux and Windows.

Quitting the window leaves agents running in tmux. Anyone who does not know
tmux is there needs to be shown that.

## tmux as a server

tmux is already a server with clients. A GUI can be the client in two ways.

1. **An ordinary client in a pty.** The terminal view runs `tmux attach` and
   the sidebar drives it with `switch-client`. A day's work. The cost is that
   panes are tmux's: borders drawn in characters, no native divider drag, and
   scrollback and selection go through copy mode.
2. **Control mode (`tmux -C`).** tmux stops drawing and speaks a text protocol
   (`%output`, `%layout-change`, `%window-add`, `%session-changed`). The GUI
   draws each pane as its own terminal view, with native splits, scrollback and
   selection. This is how iTerm2's tmux integration works. More work: one view
   per pane, fed from `%output`, keys sent with `send-keys`, tmux's layout tree
   mirrored. It would also give the daemon push events in place of polling
   pane directories.

Do the first to find out whether the thing is worth having. The second is what
makes tmux invisible, and nothing from the first is thrown away.

## The terminal view

| | Wails + xterm.js | Swift + libghostty |
|---|---|---|
| Language | Go, HTML/CSS | Swift, a new codebase |
| Effort | Days | Weeks |
| Terminal quality | Good, not Ghostty's | Ghostty's |
| Platforms | macOS, Linux, Windows | macOS |

Go and libghostty:

- **libghostty-vt** (parser and terminal state) is a plain C library and binds
  from Go through cgo. It hands back a grid of cells; drawing, input and the
  pty are still ours, and Go has no strong toolkit to draw text with.
- **Full libghostty** (the GPU renderer) wants a native view from its host, an
  `NSView` with a Metal layer on macOS. From Go that is cgo plus Objective-C
  against an API that is not stable. Swift is the lower-effort way to this
  layer, with the Go daemon unchanged behind it.
- **In a webview**: libghostty-vt compiled to WebAssembly behind an
  xterm.js-compatible API would give Ghostty's emulation inside Wails, drawn by
  the browser's canvas.

Not checked: that Go bindings for libghostty-vt (`go-libghostty`) exist and are
current; that Coder's `ghostty-web` is the WebAssembly build described above;
that libghostty-vt reached 1.0 on 2026-11-06, which only secondary sources say.
The full embedding API was documented as not stabilised.

## cmux, tried and dropped

cmux is a macOS terminal built on libghostty, driven by a CLI and a unix
socket. The idea was a second backend: cmux in place of tmux, its sidebar in
place of `puffin pick`. A probe on 0.65.2 showed the CLI can do all of it:

| puffin needs | cmux 0.65.2 |
|---|---|
| A session with panes and an agent | `new-workspace --name --cwd --command --layout <json> --env` |
| A status word on the row | `set-status <key> <value> --color --icon --priority`, read back with `list-status` |
| To know which task a workspace is | `--env PUFFIN_TASK=…`, read back with `workspace env`; `description` comes back in `list-workspaces` |
| Order by rank, rename, close | `reorder-workspaces --order`, `rename-workspace`, `close-workspace` |
| Push instead of polling | `cmux events`, NDJSON: `workspace.created`, `renamed`, `reordered`, `notification.created` |

What would have shaped a backend:

- **Socket access.** By default only processes started inside a cmux terminal
  may connect. A daemon started elsewhere is refused unless cmux is put in
  password mode.
- **The status lane is cmux's.** `workspace status set` is an override that
  clears when cmux's own inference moves. Pills stay put.
- **cmux's own pull request pill** uses the workspace's first PR, not the
  bottom of the stack. It can be hidden, for the whole app only, with
  `sidebar.showPullRequests` in `~/.config/cmux/cmux.json`. `showBranchDirectory`,
  `showWorkspaceDescription`, `showNotificationMessage`, `showPorts` and
  `showLog` trim the rest of a row.
- **Custom sidebars** (beta) are JavaScript files in `~/.config/cmux/sidebars/`,
  interpreted at runtime. The bundled templates read title, PR, progress,
  agents and unread; none reads a status pill.

Dropped because the look is cmux's to decide, not because anything failed.
