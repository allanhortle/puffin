# Puffin

What's up? Puffins, that's what.

A CLI tool for keeping dotfiles and brew dependencies in sync. The goal is to have one consistent dev environment across any computer.

Requires:
* [homebrew]
* [node]
* [oh-my-zsh]


## Getting Started

**WARNING: Puffin will override your current dotfiles. Make a backup if you want to preserve existing files**

```sh
# Install Homebrew
/usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"

# Install oh-my-zsh
sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"

# Install Node
brew install node

# Install Puffin
npm install -g puffin

puffin install
puffin install-apps
puffin reload
. ~/.zshrc # I cant find a good way to reload zsh via node
```

## Puffin Install
Uses [homebrew] to install dependencies.

* fasd
* git
* htop
* massren
* node
* reattach-to-user-namespace
* tig
* tmux
* ansiweather
* fzf
* ripgrep
* sops
* todo-txt
* vim
* visidata
* yarn

## `puffin install-apps`
Uses `brew cask` to install apps.

* 1password
* alfred
* bettertouchtool
* dropbox
* firefox
* google-chrome
* iterm2
* slack
* spotify
* the-unarchiver
* transmit
* visual-studio-code

## Puffin Reload
Rsyncs dotfiles from puffin to your `$HOME` folder.

* `.aliases`
* `.functions`
* `.tigrc`
* `.tmux.conf`
* `.vimrc`
* `.zshrc`
* `.vim/`

## Config Environment Variables
`PUFFIN_WEATHER_LOCATION` Sets ansiweather location. Defaults to 'Melbourne, AU'.

Each file will try to source `.xxx.local` at the very end. This lets you add any computer specific configs to each file.
For example you may choose to place some environment variables in `.zshrc.local`.


[homebrew]: http://brew.sh/
[tig]: https://github.com/jonas/tig
[node]: https://nodejs.org/
[oh-my-zsh]: https://github.com/robbyrussell/oh-my-zsh
