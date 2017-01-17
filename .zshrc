
## Oh My ZSH ##

export ZSH=$HOME/.oh-my-zsh
ENABLE_CORRECTION="true"
DISABLE_AUTO_TITLE="true"
plugins=(git brew tmux npm react-native)
export PATH=$HOME/bin:/usr/local/bin:$PATH
source $ZSH/oh-my-zsh.sh

unsetopt correct_all  
setopt correct


#--------------------------------------------------------------------------
# Git Prompt Functions
#--------------------------------------------------------------------------

# Show different symbols as appropriate for various Git repository states
parse_git_state() {



  # Compose this value via multiple conditional appends.
  local GIT_STATE=""

  local NUM_AHEAD="$(git log --oneline @{u}.. 2> /dev/null | wc -l | tr -d ' ')"
  if [ "$NUM_AHEAD" -gt 0 ]; then
    GIT_STATE=$GIT_STATE${ZSH_THEME_GIT_PROMPT_AHEAD//NUM/$NUM_AHEAD}
  fi

  local NUM_BEHIND="$(git log --oneline ..@{u} 2> /dev/null | wc -l | tr -d ' ')"
  if [ "$NUM_BEHIND" -gt 0 ]; then
    GIT_STATE=$GIT_STATE${ZSH_THEME_GIT_PROMPT_BEHIND//NUM/$NUM_BEHIND}
  fi

  local NUM_CONFLICTS="$(git diff --name-only --diff-filter=U &> /dev/null | wc -l | xargs echo)"
  if [[ "$NUM_CONFLICTS" -gt 0 ]]; then
    GIT_STATE=$GIT_STATE${ZSH_THEME_GIT_PROMPT_CONFLICTS//NUM/$NUM_CONFLICTS}
  fi

  local NUM_STAGED="$(git status --short 2> /dev/null | grep '^[AM]' | wc -l | xargs echo)"
  if [[ "$NUM_STAGED" -gt 0 ]]; then
    GIT_STATE=$GIT_STATE${ZSH_THEME_GIT_PROMPT_STAGED//NUM/$NUM_STAGED}
  fi

  local NUM_MODIFIED="$(git status --short 2> /dev/null | grep '^ M' | wc -l | xargs echo)"
  if [[ "$NUM_MODIFIED" -gt 0 ]]; then
    GIT_STATE=$GIT_STATE${ZSH_THEME_GIT_PROMPT_MODIFIED//NUM/$NUM_MODIFIED}
  fi

  local NUM_UNTRACKED="$(git status --short 2> /dev/null | grep '^[??]' | wc -l | xargs echo)"
  if [[ "$NUM_UNTRACKED" -gt 0 ]]; then
    GIT_STATE=$GIT_STATE${ZSH_THEME_GIT_PROMPT_UNTRACKED//NUM/$NUM_UNTRACKED}
    # GIT_STATE=$GIT_STATE$ZSH_THEME_GIT_PROMPT_UNTRACKED
  fi



  if [[ -n $GIT_STATE ]]; then
    echo "$GIT_STATE"
    #echo "$ZSH_THEME_GIT_PROMPT_PREFIX$GIT_STATE$ZSH_THEME_GIT_PROMPT_SUFFIX"
  fi

}

# If inside a Git repository, print its branch and state
git_prompt_info() {
	local git_where="$(parse_git_branch)"
	[ -n "$git_where" ] && echo "$ZSH_THEME_GIT_PROMPT_PREFIX${git_where#(refs/heads/|tags/)} $(parse_git_state)$ZSH_THEME_GIT_PROMPT_SUFFIX"
}

# Show Git branch/tag, or name-rev if on detached head
parse_git_branch() {
  (git symbolic-ref -q HEAD || git name-rev --name-only --no-undefined --always HEAD) 2> /dev/null
}


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

ZSH_THEME_GIT_PROMPT_PREFIX=" ${g}"
ZSH_THEME_GIT_PROMPT_SUFFIX=""
ZSH_THEME_GIT_PROMPT_AHEAD="${g}⬆ NUM "
ZSH_THEME_GIT_PROMPT_BEHIND="${g}⬇ NUM${g} "
ZSH_THEME_GIT_PROMPT_CONFLICTS="${r}✕NUM${g} "
ZSH_THEME_GIT_PROMPT_STAGED="${g}SNUM "
ZSH_THEME_GIT_PROMPT_MODIFIED="${g}MNUM "
ZSH_THEME_GIT_PROMPT_UNTRACKED="${g}UNUM "

puffin_prompt() {
    RPROMPT="${y}%n@%M %D{%R}${res}"
    PROMPT=$'${b}%~ $(git_prompt_info) ${y}$(puffin_prompt_extra &>/dev/null && puffin_prompt_extra)  
${b}=> ${res}'
} 
puffin_prompt
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
