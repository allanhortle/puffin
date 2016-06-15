"
" Plugins
"
set nocompatible
if empty(glob('~/.vim/autoload/plug.vim'))
    silent !curl -fLo ~/.vim/autoload/plug.vim --create-dirs
        \ https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim
    autocmd VimEnter * PlugInstall | source $MYVIMRC
endif
call plug#begin('~/.vim/plugged')

Plug 'airblade/vim-gitgutter'
Plug 'cakebaker/scss-syntax.vim'
Plug 'docunext/closetag.vim'
Plug 'editorconfig/editorconfig-vim'
Plug 'gabrielelana/vim-markdown'
Plug 'honza/vim-snippets'
Plug 'isRuslan/vim-es6'
Plug 'jeffkreeftmeijer/vim-numbertoggle'
Plug 'jiangmiao/auto-pairs'
Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': './install --all' } 
Plug 'junegunn/fzf.vim'
Plug 'junegunn/goyo.vim'
Plug 'mattn/emmet-vim'
Plug 'mxw/vim-jsx'
Plug 'morhetz/gruvbox'
Plug 'pangloss/vim-javascript'
Plug 'ddrscott/vim-side-search'
Plug 'scrooloose/nerdcommenter'
Plug 'scrooloose/nerdtree', { 'on': 'NERDTreeToggle' }
Plug 'sheerun/vim-polyglot'
Plug 'tpope/vim-surround'
Plug 'tpope/vim-fugitive'
Plug 'tpope/vim-speeddating'
Plug 'unblevable/quick-scope'
Plug 'bentayloruk/vim-react-es6-snippets'


call plug#end() 

"
" General Vim
"
set encoding=utf8
set rtp+=~/.fzf
set wrap linebreak nolist
set background=dark
set expandtab
set splitright
set tabstop=4                   " a tab is four spaces
set softtabstop=4
set backspace=indent,eol,start  " allow backspacing over everything in insert mode
set autoindent                  " always set auto-indenting on
set copyindent                  " copy the previous indentation on auto-indenting
set number                      " always show line numbers
set shiftwidth=4                " number of spaces to use for auto-indenting
set shiftround                  " use multiple of shiftwidth when indenting with '<' and '>'
set showmatch                   " set show matching 
set ignorecase                  " ignore case when searching
set smartcase                   " ignore case if search pattern is all lower-case case-sensitive otherwise
set smarttab                    " insert tabs on the start of a line according to shiftwidth, not tabstop
set incsearch                   " show search matches as you type
set t_Co=256
set updatetime=250
set mouse=a

set nobackup                    " No backups.
set nowritebackup               " No backups.
set noswapfile                  " No swap files; more hassle than they're worth."

"
" ## PLUGINS ##
"

" Nerd Tree
map <C-o> :NERDTreeToggle<CR>
let NERDTreeShowHidden=1
let NERDTreeQuitOnOpen=1
" close if nerd tree is the only buffer
autocmd bufenter * if (winnr("$") == 1 && exists("b:NERDTree") && b:NERDTree.isTabTree()) | q | endif

" Quick Scope
let g:qs_first_occurrence_highlight_color = 244       " terminal vim
let g:qs_second_occurrence_highlight_color = 245         " terminal vim

" Syntastic
let g:syntastic_always_populate_loc_list = 1
let g:syntastic_auto_loc_list = 1
let g:syntastic_check_on_open = 1
let g:syntastic_check_on_wq = 0
let g:syntastic_javascript_checkers = ['eslint']


"
" ## Keyboard Mappings ##
"
map <F10> :echo "hi<" . synIDattr(synID(line("."),col("."),1),"name") . '> trans<' . synIDattr(synID(line("."),col("."),0),"name") . "> lo<" . synIDattr(synIDtrans(synID(line("."),col("."),1)),"name") . ">"<CR>
set pastetoggle=<F2>
map <F5> :setlocal spell! spelllang=en_au<CR>
map <F3> :set wrap!<CR>

" Normal mode
nnoremap <C-p> :Files<CR>
nnoremap <C-f> :Ag<CR>
nnoremap <CR> :noh<CR><CR>
nnoremap <Leader>p :let @" = expand("%")<CR>
nnoremap <Leader>l :ls<CR>
nnoremap <Leader>b :bp<CR>
nnoremap <Leader>f :bn<CR>
nnoremap <Leader>g :e#<CR>
nnoremap <Leader>1 :1b<CR>
nnoremap <Leader>2 :2b<CR>
nnoremap <Leader>3 :3b<CR>
nnoremap <Leader>4 :4b<CR>
nnoremap <Leader>5 :5b<CR>
nnoremap <Leader>6 :6b<CR>
nnoremap <Leader>7 :7b<CR>
nnoremap <Leader>8 :8b<CR>
nnoremap <Leader>9 :9b<CR>
nnoremap <Leader>0 :10b<CR>

" tig
nnoremap <Leader>tt :!tig<CR><CR>
nnoremap <Leader>ts :!tig status<CR><CR>
nnoremap <Leader>th :!tig %<CR><CR>
nnoremap <Leader>tb :!tig blame %<CR><CR>

" windows
nnoremap <Tab> <C-W>w
nnoremap <S-Tab> <C-W>W

"
" window/buffer splitting
"

nmap <leader>s<left>   :leftabove  vnew<CR>
nmap <leader>s<right>  :rightbelow vnew<CR>
nmap <leader>s<up>     :leftabove  new<CR>
nmap <leader>s<down>   :rightbelow new<CR>

"
"Insert Mode
imap <c-x><c-l> <plug>(fzf-complete-line)
inoremap <leader>; <C-o>A;

function! s:Saving_scroll(cmd)
  let save_scroll = &scroll
  execute 'normal! ' . a:cmd
  let &scroll = save_scroll
endfunction
nnoremap <C-J> :call <SID>Saving_scroll("1<C-V><C-D>")<CR>
vnoremap <C-J> <Esc>:call <SID>Saving_scroll("gv1<C-V><C-D>")<CR>
nnoremap <C-K> :call <SID>Saving_scroll("1<C-V><C-U>")<CR>
vnoremap <C-K> <Esc>:call <SID>Saving_scroll("gv1<C-V><C-U>")<CR>

"
" Commands
"
command! WQ wq
command! Wq wq
command! W w
command! Q q

"
" Custom Hardmode
"
noremap <Up> <NOP>
noremap <Down> <NOP>
noremap <Left> <NOP>
noremap <Right> <NOP>
noremap <BS> <NOP>
"
" Syntax Highlighting
"

" jsx files
let g:jsx_ext_required = 0
autocmd BufNewFile,BufRead *.json set syntax=javascript


"
" User Interface
"

" status line
let g:currentmode={
    \ 'n': 'Normal ',
    \ 'no': 'N·Operator Pending ',
    \ 'v': 'Visual ',
    \ 'V': 'V·Line ',
    \ '^V': 'V·Block ',
    \ 's': 'Select ',
    \ 'S': 'S·Line ',
    \ '^S': 'S·Block ',
    \ 'i': 'Insert ',
    \ 'R': 'Replace ',
    \ 'Rv': 'V·Replace ',
    \ 'c': 'Command ',
    \ 'cv': 'Vim Ex ',
    \ 'ce': 'Ex ',
    \ 'r': 'Prompt ',
    \ 'rm': 'More ',
    \ 'r?': 'Confirm ',
    \ '!': 'Shell ',
    \ 't': 'Terminal '
\}

" Automatically change the statusline color depending on mode
function! ChangeStatuslineColor()
    if (mode() =~# '\v(n|no)')
        exe 'hi! StatusLine ctermfg=002 cterm=reverse,bold'
        exe 'hi! StatusLineNC ctermfg=8'
    elseif (mode() =~# '\v(v|V)' || g:currentmode[mode()] ==# 'V·Block' || get(g:currentmode, mode(), '') ==# 't')
        exe 'hi! StatusLine ctermfg=004'
    elseif (mode() ==# 'i')
        exe 'hi! StatusLine ctermfg=003'
    else
        exe 'hi! StatusLine ctermfg=008'
    endif
    return ''
endfunction

   
"set noruler
"set noshowmode
set laststatus=2
set statusline=
set statusline+=%{ChangeStatuslineColor()} " Changing the statusline color
set statusline+=\ %{toupper(g:currentmode[mode()])} " Current mode
set statusline+=%-.50F " Full file path, max 50 chars
set statusline+=%m 
set statusline+=%=                                  " Right Right side
set statusline+=\ %Y                                " FileType
set statusline+=\ %l:%c                " Rownumber/total (%)
set statusline+=\ 

syntax reset
syntax on
colorscheme galea

"hi User1 ctermbg=000 ctermfg=007

if filereadable("~/.vimrc.local")
    so '~/.vimrc.local'
endif


