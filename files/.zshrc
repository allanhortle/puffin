# Brew Completions
if type brew &>/dev/null; then
  FPATH=$(brew --prefix)/share/zsh/site-functions:$FPATH

  autoload -Uz compinit
  compinit
fi

autoload -Uz vcs_info

## Oh My ZSH ##
export ZSH=$HOME/.oh-my-zsh
export NVM_LAZY_LOAD=true
export NVM_COMPLETION=true
ENABLE_CORRECTION="true"
DISABLE_AUTO_TITLE="true"
plugins=(git brew tmux yarn npm aws vi-mode history-substring-search zsh-nvm fzf)
export PATH=$HOME/bin:/usr/local/bin:$PATH
export KEYTIMEOUT=1
source $ZSH/oh-my-zsh.sh

unsetopt correct_all  
setopt correct






## Prompt ##
b="%{$fg[blue]%}"
bl="%{$fg[black]%}"
g="%{$fg[green]%}"
c="%{$fg[cyan]%}"
r="%{$fg[red]%}"
y="%{$fg[yellow]%}"
w="%{$fg[white]%}"
m="%{$fg[magenta]%}"
res="%{$reset_color%}"

puffin_prompt() {
    local pathPrompt="%~  "
    local gitPrompt="$(git_prompt_info)"
    local extraPrompt="$(puffin_prmpt_extra &>/dev/null && puffin_prompt_extra)"
    PROMPT="${b}${pathPrompt}${g}${vcs_info_msg_0_}${y}${extraPrompt}${y}%D{%R}${b}
=>%{$reset_color%} "
} 


export FZF_DEFAULT_COMMAND='rg --files --hidden --follow --glob "!.git"'
export FZF_DEFAULT_OPTS='--color=16,bg+:-1,pointer:2,prompt:2,hl+:2,hl:2,fg+:2'
export EDITOR="/usr/local/bin/vim"
eval "$(fasd --init auto)"

zstyle ':vcs_info:*' enable git
zstyle ':vcs_info:*' check-for-changes true
zstyle ':vcs_info:git*' formats "%b%m  "
zstyle ':vcs_info:git*' actionformats "%a%b%m  "
zstyle ':vcs_info:*+*:*' debug false
zstyle ':vcs_info:git*+set-message:*' hooks git-extras

git_changes() {
    local NUM="$(echo $1 | grep $2 | wc -l | xargs echo)"
    if [[ "$NUM" -gt 0 ]]; then
        echo "$3$NUM"
    fi
}

function +vi-git-extras(){
    local changes=$(git status --short --branch -u 2> /dev/null);
    local STAGED=""
    local MISC=""
    local UNSTAGED=""

    # Staged
    STAGED+="$(git_changes "${changes}" "^[AC]" "+")"
    STAGED+="$(git_changes "${changes}" "^M" "~")"
    STAGED+="$(git_changes "${changes}" "^D" "-")"
    STAGED+="$(git_changes "${changes}" "^R" ">")"
    if [[ -n $STAGED ]]; then
        hook_com[misc]+=" [${STAGED}]"
    fi

    # Unstaged
    UNSTAGED+="$(git_changes "${changes}" "^.M" "~")"
    UNSTAGED+="$(git_changes "${changes}" "^.D" "-")"
    UNSTAGED+="$(git_changes "${changes}" "^??" "?")"
    if [[ -n $UNSTAGED ]]; then
        hook_com[misc]+=" $UNSTAGED";
    fi

    # Ahead/Behind/Conficts
    #local NUM_AHEAD="$(git log --oneline @{u}.. 2> /dev/null | wc -l | xargs echo)"
    local NUM_AHEAD=$(echo $changes | head -1 | pcregrep -io1 "ahead (\d+)")
    if [ "$NUM_AHEAD" -gt 0 ]; then
        hook_com[misc]+=" ⬆ $NUM_AHEAD"
    fi

    #local NUM_BEHIND="$(git log --oneline ..@{u} 2> /dev/null | wc -l | xargs echo)"
    local NUM_BEHIND=$(echo $changes | head -1 | pcregrep -io1 "behind (\d+)")
    if [ "$NUM_BEHIND" -gt 0 ]; then
        hook_com[misc]+=" ⬇ $NUM_BEHIND"
    fi

    MISC+="$(git_changes "${changes}" '^UU' "${r}✕")"
    if [[ -n $MISC ]]; then
        hook_com[misc]+=" ${MISC}"
    fi
}

zstyle ':completion:*' verbose yes
zstyle ':completion:*' menu select
zstyle ':completion:*' list-colors ${(s.:.)LS_COLORS}
zstyle ':completion:*' verbose yes
zstyle ':completion:*:descriptions' format "$fg[yellow]%B[%d]%b"
zstyle ':completion:*:messages' format '%d'
zstyle ':completion:*:warnings' format "$fg[red]No matches for:$reset_color %d"
zstyle ':completion:*:corrections' format '%B%d (errors: %e)%b'
zstyle ':completion:*' group-name ''
zstyle ':completion:::::' completer _complete _approximate
zstyle ':completion:*:approximate:*' max-errors 2

precmd() {
    vcs_info;
    puffin_prompt;
}

# Loading Configs ##
if [ -f ~/.zshrc.local ]; then
    source ~/.zshrc.local
fi

# Load the shell dotfiles
for file in ~/.{aliases,functions}; do
    [ -r "$file" ] && [ -f "$file" ] && source "$file";
done;
unset file;

