

## Oh My ZSH ##

export ZSH=$HOME/.oh-my-zsh
ENABLE_CORRECTION="true"
DISABLE_AUTO_TITLE="true"
plugins=(git brew tmux npm)
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

ZSH_THEME_GIT_PROMPT_PREFIX="${b}[${g}"
ZSH_THEME_GIT_PROMPT_SUFFIX="${b}]${res}"
ZSH_THEME_GIT_PROMPT_AHEAD="${r}⬆ NUM${res}"
ZSH_THEME_GIT_PROMPT_BEHIND="${g}⬇ NUM${res}"
ZSH_THEME_GIT_PROMPT_MERGING="${m}✕${res}"
ZSH_THEME_GIT_PROMPT_UNTRACKED="${r}●${res}"
ZSH_THEME_GIT_PROMPT_MODIFIED="${y}●${res}"
ZSH_THEME_GIT_PROMPT_STAGED="${g}●${res}"

PROMPT=$'${b}%2~$(git_prompt_info) ${b}=> ${res}'
RPROMPT="${y}%n@%M %T${res}"
ZLE_RPROMPT_INDENT=0
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
