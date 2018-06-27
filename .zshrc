
## Oh My ZSH ##
export ZSH=$HOME/.oh-my-zsh
ENABLE_CORRECTION="true"
DISABLE_AUTO_TITLE="true"
plugins=(git brew tmux npm react-native vi-mode history-substring-search)
export PATH=$HOME/bin:/usr/local/bin:$PATH
source $ZSH/oh-my-zsh.sh

# todo.txt config
export TODO_DIR="$HOME/Dropbox/data/todo"
export TODO_FILE="$TODO_DIR/todo.txt"
export DONE_FILE="$TODO_DIR/done.txt"
export REPORT_FILE="$TODO_DIR/report.txt"
export TMP_FILE="/tmp/todo.tmp"
export TODOTXT_DEFAULT_ACTION=ls
export TODOTXT_AUTO_ARCHIVE=false
export TODOTXT_SORT_COMMAND='env LC_COLLATE=C sort -k 2,2 -k 1,1n'
export TODOTXT_GRAPH_THRESHOLD=3


unsetopt correct_all  
setopt correct



export KEYTIMEOUT=1
#bindkey '^[[1;3D' backward-word # opt + left
#bindkey '^[[1;3?' backward-delete-word # opt + backspace
#bindkey '^[[1;3~' backward-forward-word # opt + delete
#bindkey '^[[1;3C' forward-word # opt + right



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

git_changes() {
    local NUM="$(git status --short -u 2> /dev/null | grep $1 | wc -l | xargs echo)"
    if [[ "$NUM" -gt 0 ]]; then
        echo "$2$NUM"
    fi
}

parse_git_state() {
    local GIT_STATE=""
    local STAGED=""
    local UNSTAGED=""

    # Ahead/Behind/Conficts
    local NUM_AHEAD="$(git log --oneline @{u}.. 2> /dev/null | wc -l | tr -d ' ')"
    if [ "$NUM_AHEAD" -gt 0 ]; then
        GIT_STATE+=" ⬆ $NUM_AHEAD"
    fi

    local NUM_BEHIND="$(git log --oneline ..@{u} 2> /dev/null | wc -l | tr -d ' ')"
    if [ "$NUM_BEHIND" -gt 0 ]; then
        GIT_STATE+=" ⬇ $NUM_BEHIND"
    fi

    local NUM_CONFLICTS="$(git diff --name-only --diff-filter=U &> /dev/null | wc -l | xargs echo)"
    if [[ "$NUM_CONFLICTS" -gt 0 ]]; then
        GIT_STATE+=" ${r}✕$NUM_CONFLICTS${g}"
    fi


    # Staged
    STAGED+="$(git_changes '^[AC]' "${g}+")"
    STAGED+="$(git_changes '^M' "${g}~")"
    STAGED+="$(git_changes '^D' "${g}-")"
    STAGED+="$(git_changes '^R' "${g}>")"

    if [[ -n $STAGED ]]; then
        GIT_STATE+=" ${g}[$STAGED]"
    fi

    # Unstaged
    UNSTAGED+="$(git_changes '^ M' "${g}~")"
    UNSTAGED+="$(git_changes '^ D' "${g}-")"
    UNSTAGED+="$(git_changes '^??' "${g}?")"

    if [[ -n $UNSTAGED ]]; then
        GIT_STATE+=" ${g}$UNSTAGED"
    fi

    echo "$GIT_STATE"
}



# Show Git branch/tag, or name-rev if on detached head
parse_git_branch() {
    (git symbolic-ref -q HEAD || git name-rev --name-only --no-undefined --always HEAD) 2> /dev/null
}

git_prompt_info() {
    local git_where="$(parse_git_branch)"

    if [[ -n "$git_where" ]]; then
        echo "${g}${git_where#(refs/heads/|tags/)}$(parse_git_state)  "
    fi
}

puffin_prompt() {
    local pathPrompt="%3~  "
    local gitPrompt="$(git_prompt_info)"
    local extraPrompt="$(puffin_prompt_extra &>/dev/null && puffin_prompt_extra)"
    local rightPrompt="%D{%R}"

    #Magic regex to remove unseen characaters
    #local zero='%([BSUbfksu]|([FK]|){*})'
    #Magic zsh to apply the regex and count the size
    #local pathSize=${${(S%%)pathPrompt//$~zero/}} 
    #local gitSize=${${(S%%)gitPrompt//$~zero/}}
    #local extraSize=${${(S%%)extraPrompt//$~zero/}}
    #local rightSize=${${(S%%)rightPrompt//$~zero/}}

     #echo $gitSize

    #local leftOver=$(expr $COLUMNS - $pathSize - $gitSize - $extraSize - $rightSize)


    #PROMPT="${b}${pathPrompt}${g}${gitPrompt}${y}${extraPrompt}$(printf %-${leftOver}s)${y}${rightPrompt}${b}
    PROMPT="${b}${pathPrompt}${g}${gitPrompt}${y}${extraPrompt}${y}${rightPrompt}${b}
=>${res} "
} 

precmd() {
    puffin_prompt;
}



export FZF_DEFAULT_COMMAND='rg --files --hidden --follow --glob "!.git"'
export FZF_DEFAULT_OPTS='--color=16,bg+:-1,pointer:2,prompt:2,hl+:2,hl:2,fg+:2'
eval "$(fasd --init auto)"

# Loading Configs ##
if [ -f ~/.fzf.zsh ] ; then
    source ~/.fzf.zsh
fi

if [ -f ~/.zshrc.local ]; then
    source ~/.zshrc.local
fi

# Load the shell dotfiles
for file in ~/.{aliases,functions}; do
    [ -r "$file" ] && [ -f "$file" ] && source "$file";
done;
unset file;


# puffin_prompt
add-zsh-hook precmd puffin_prompt
