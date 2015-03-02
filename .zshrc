# Path to your oh-my-zsh installation.
export ZSH=$HOME/.oh-my-zsh

ENABLE_CORRECTION="true"

plugins=(git git-flow brew tmux)

export PATH=$HOME/bin:/usr/local/bin:$PATH

source $ZSH/oh-my-zsh.sh







b="%{$fg[blue]%}"
bl="%{$fg[black]%}"
g="%{$fg[green]%}"
c="%{$fg[cyan]%}"
r="%{$fg[red]%}"
y="%{$fg[yellow]%}"
w="%{$fg[white]%}"
m="%{$fg[magenta]%}"
res="%{$reset_color%}"

# Modify the colors and symbols in these variables as desired.
ZSH_THEME_GIT_PROMPT_PREFIX="${b}[${g}"
ZSH_THEME_GIT_PROMPT_SUFFIX="${b}]${res}"
ZSH_THEME_GIT_PROMPT_AHEAD="${r}⬆ NUM${res}"
ZSH_THEME_GIT_PROMPT_BEHIND="${g}⬇ NUM${res}"
ZSH_THEME_GIT_PROMPT_MERGING="${m}✕${res}"
ZSH_THEME_GIT_PROMPT_UNTRACKED="${r}●${res}"
ZSH_THEME_GIT_PROMPT_MODIFIED="${y}●${res}"
ZSH_THEME_GIT_PROMPT_STAGED="${g}●${res}"

PROMPT=$'${b}%c$(git_prompt_info) ${b}=> ${res}'
RPROMPT="${g}%M [%D{%H:%M:%S}]${res}"




source ~/.zshrc.local


# Load the shell dotfiles, and then some:
# * ~/.path can be used to extend `$PATH`.
# * ~/.extra can be used for other settings you don’t want to commit.
for file in ~/.{path,exports,aliases,functions,extra}; do
	[ -r "$file" ] && [ -f "$file" ] && source "$file";
done;
unset file;
