
## Oh My ZSH ##

export ZSH=$HOME/.oh-my-zsh
ENABLE_CORRECTION="true"
DISABLE_AUTO_TITLE="true"
plugins=(git brew tmux npm react-native)
export PATH=$HOME/bin:/usr/local/bin:$PATH
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

git_changes() {
    local NUM="$(git status --short 2> /dev/null | grep $1 | wc -l | xargs echo)"
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
        GIT_STATE+="⬆ $NUM_AHEAD"
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
    STAGED+="$(git_changes '^[AC]' '+')"
    STAGED+="$(git_changes '^M' '~')"
    STAGED+="$(git_changes '^D' '-')"
    STAGED+="$(git_changes '^R' '>')"

    if [[ -n $STAGED ]]; then
        GIT_STATE+=" S$STAGED"
    fi

    # Unstaged
    UNSTAGED+="$(git_changes '^ M' '~')"
    UNSTAGED+="$(git_changes '^ D' '-')"

    if [[ -n $UNSTAGED ]]; then
        GIT_STATE+=" U$UNSTAGED"
    fi

    # Untracked
    local NUM_UNTRACKED="$(git status --short 2> /dev/null | grep '^[??]' | wc -l | xargs echo)"
    if [[ "$NUM_UNTRACKED" -gt 0 ]]; then
        GIT_STATE+=" ?$NUM_UNTRACKED"
    fi

    if [[ -n $GIT_STATE ]]; then
        echo "$GIT_STATE"
    fi
}


# Show Git branch/tag, or name-rev if on detached head
parse_git_branch() {
    (git symbolic-ref -q HEAD || git name-rev --name-only --no-undefined --always HEAD) 2> /dev/null
}

git_prompt_info() {
    local git_where="$(parse_git_branch)"

    if [[ -n "$git_where" ]]; then
        echo "${g}${git_where#(refs/heads/|tags/)} $(parse_git_state)"
    fi
}


puffin_prompt() {
    RPROMPT="${y}%n@%M %D{%R}${res}"
    PROMPT=$'${b}%~ $(git_prompt_info) ${y}$(puffin_prompt_extra &>/dev/null && puffin_prompt_extra)  
${b}=> ${res}'
} 
puffin_prompt

export FZF_DEFAULT_COMMAND='ag --hidden --ignore .git -g ""'
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
