# Path to your oh-my-zsh installation.
export ZSH=$HOME/.oh-my-zsh

ZSH_CUSTOM=$HOME/github/otter
ZSH_THEME="otter"
plugins=(git otter tmux)

source $ZSH/oh-my-zsh.sh




alias tat='tmux new -s `basename $PWD`'
alias tr='source-file ~/.tmux.conf'

source ~/.zshrc.local

