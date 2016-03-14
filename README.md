Puffin
======

What's up? Puffins, that's what.

A CLI tool for keeping dotfiles and brew dependencies in sync. The goal is to have one consistent dev environment across any computer.

Requires:
* [homebrew]
* [node]
* [oh-my-zsh]


## Getting Started

**WARNING: currently Puffin will override your current dotfiles. MAKE A BACKUP BEFORE RUNNING THE COMMANDS BELOW**

```sh
npm install -g puffin
puffin install
puffin reload
. ~/.zshrc # I cant find a good way to reload zsh via node
```

## Puffin Install
Uses [homebrew] to install dependencies.

* massren
* reattach-to-user-namespace
* tmux
* [tig]
* [fasd]
* git
* ansiweather
* node (Just to make sure you dont need sudo)

## Puffin Reload
Rsyncs dotfiles from puffin to your `$HOME` folder.

* `.aliases`
* `.functions`
* `.tigrc`
* `.tmux.conf`
* `.vimrc`
* `.zshrc`

## Config Environment Variables
`PUFFIN_WEATHER_LOCATION` Sets ansiweather location. Defaults to 'Melbourne, AU'.

Each file will try to source `.xxx.local` at the very end. This lets you add any computer specific configs to each file.
For example you may choose to place some environment variables in `.zshrc.local`.


## TODO

* Backup before rsyncing
* check for updated before each command
* default `puffin` command to check for updates and install/reload


[homebrew]: http://brew.sh/
[fasd]: https://github.com/clvv/fasd
[tig]: https://github.com/jonas/tig
[node]: https://nodejs.org/
[oh-my-zsh]: https://github.com/robbyrussell/oh-my-zsh
