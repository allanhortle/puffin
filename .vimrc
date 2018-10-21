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

Plug '~/localhost/code/vim-boring-javascript'
Plug '/usr/local/opt/fzf' | Plug 'junegunn/fzf.vim'
"Plug 'SirVer/ultisnips'
Plug 'airblade/vim-gitgutter'
Plug 'ajh17/VimCompletesMe'
Plug 'bentayloruk/vim-react-es6-snippets'
Plug 'cakebaker/scss-syntax.vim'
Plug 'christoomey/vim-sort-motion'
Plug 'docunext/closetag.vim'
Plug 'dyng/ctrlsf.vim'
Plug 'editorconfig/editorconfig-vim'
Plug 'freitass/todo.txt-vim'
Plug 'honza/vim-snippets'
Plug 'https://github.com/w0rp/ale.git'
Plug 'isRuslan/vim-es6'
Plug 'itchyny/lightline.vim'
Plug 'jceb/vim-orgmode'
Plug 'jparise/vim-graphql'
Plug 'jeffkreeftmeijer/vim-numbertoggle'
Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': './install --all' } 
Plug 'mattn/emmet-vim'
Plug 'mbbill/undotree'
Plug 'mhinz/vim-startify'
Plug 'nelstrom/vim-markdown-folding'
Plug 'retorillo/istanbul.vim'
Plug 'scrooloose/nerdcommenter'
Plug 'scrooloose/nerdtree', { 'on': ['NERDTreeToggle', 'NERDTreeFind'] }
Plug 'tpope/vim-abolish'
Plug 'tpope/vim-fugitive'
Plug 'tpope/vim-markdown'
Plug 'tpope/vim-repeat'
Plug 'tpope/vim-speeddating'
Plug 'tpope/vim-surround'
Plug 'unblevable/quick-scope'
let g:ale_statusline_format = ['☀️️ %d', '🕯️ %d', '']
nmap <silent> <C-k> <Plug>(ale_previous_wrap)
nmap <silent> <C-j> <Plug>(ale_next_wrap)
" let g:ale_sign_error = '❌'
" let g:ale_sign_warning = '⭕'e'


call plug#end() 

"
" General Vim
"
set autoindent                  " always set auto-indenting on
set background=dark
set backspace=indent,eol,start  " allow backspacing over everything in insert mode
set copyindent                  " copy the previous indentation on auto-indenting
set encoding=utf8
set expandtab
set hidden
set hlsearch
set ignorecase                  " ignore case when searching
set incsearch                   " show search matches as you type
set mouse=a
set number                      " always show line numbers
set rtp+=/usr/local/opt/fzf
set shiftround                  " use multiple of shiftwidth when indenting with '<' and '>'
set shiftwidth=4                " number of spaces to use for auto-indenting
set showmatch                   " set show matching 
set smartcase                   " ignore case if search pattern is all lower-case case-sensitive otherwise
set smarttab                    " insert tabs on the start of a line according to shiftwidth, not tabstop
set softtabstop=4
set splitright
set t_Co=256
set tabstop=4                   " a tab is four spaces
set updatetime=250
set wildmenu
set nowrap
set timeoutlen=1000 ttimeoutlen=0

set nobackup                    " No backups.
set nowritebackup               " No backups.
set noswapfile                  " No swap files; more hassle than they're worth."
set complete=.,w,b,u,i          " turn off tab completion for tags

"
" ## PLUGINS ##
"

" ALE
let g:ale_linters = {'javascript': ['eslint', 'flow']}

" Lightline
set noshowmode
set laststatus=2
let g:lightline = { 
    \ 'enable': {
    \   'tabline': 0
    \ },
    \ 'colorscheme': 'jellybeans',
    \ 'active': {
    \   'right': [ ['lineinfo'], ['filetype'] ]
    \ }
\ }

" markdown
let g:markdown_fold_style = 'nested'

" emmet
"let g:user_emmet_expandabbr_key='<Tab>'
"imap <expr> <tab> emmet#expandAbbrIntelligent("\<tab>")


" Nerd Tree
map <C-o> :NERDTreeToggle<CR>
let NERDTreeShowHidden=1
let NERDTreeQuitOnOpen=1
let NERDTreeMinimalUI = 1
let NERDTreeDirArrows = 1
" close if nerd tree is the only buffer
autocmd bufenter * if (winnr("$") == 1 && exists("b:NERDTree") && b:NERDTree.isTabTree()) | q | endif

" netrw
let g:netrw_liststyle = 3
let g:netrw_banner = 0


" RG
command! -bang -nargs=* Rg
  \ call fzf#vim#grep(
  \   'rg --column --line-number --no-heading --color=always --smart-case '.shellescape(<q-args>), 1,
  \   <bang>0 ? fzf#vim#with_preview('up:60%')
  \           : fzf#vim#with_preview('right:50%:hidden', '?'),
  \   <bang>0)

command! -bang -nargs=? -complete=dir Files
  \ call fzf#vim#files(<q-args>, fzf#vim#with_preview(), <bang>0)

" Startify
let g:startify_change_to_dir = 0

" UtilSnips

"
" ## Keyboard Mappings ##
"
nmap <F1> :echo expand('%:p')<CR>
imap <F1> :echo expand('%:p')<CR>
set pastetoggle=<F2>
map <F6> :setlocal spell! spelllang=en_au<CR>
map <F3> :set wrap!<CR>
map <F10> :echo "hi<" . synIDattr(synID(line("."),col("."),1),"name") . '> trans<' . synIDattr(synID(line("."),col("."),0),"name") . "> lo<" . synIDattr(synIDtrans(synID(line("."),col("."),1)),"name") . ">"<CR>

" Normal mode
nnoremap <C-p> :Files<CR>
nnoremap <C-f> :Rg<CR>
nnoremap <C-i> :NERDTreeFind<CR>
nnoremap <CR> :noh<CR><CR>
nnoremap <Leader>p :Welcome<CR
nnoremap <Leader>cp :let @* = expand("%")<CR>
nnoremap <Leader>l :ls<CR>
nnoremap <Leader>b :bp<CR>
nnoremap <Leader>e :ALENextWrap<CR>
nnoremap <Leader>E :ALEPreviousWrap<CR>
nnoremap <Leader>f :bn<CR>

" tig
nnoremap <Leader>tt :!tig<CR><CR>
nnoremap <Leader>ts :!tig status<CR><CR>
nnoremap <Leader>th :!tig %<CR><CR>
nnoremap <Leader>tb :!tig blame %<CR><CR>

" windows
nnoremap <Leader><Tab> <C-W>w

" window/buffer splitting
nmap <leader>s<left>   :leftabove  vnew<CR>
nmap <leader>s<right>  :rightbelow vnew<CR>
nmap <leader>s<up>     :leftabove  new<CR>
nmap <leader>s<down>   :rightbelow new<CR>

"Insert Mode
imap <c-x><c-l> <plug>(fzf-complete-line)
inoremap <leader>; <C-o>A;
inoremap jk <esc>


"
" Commands
"
command! WQa wqa
command! Wqa wqa
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
noremap <TAB> <NOP>
"
" Syntax Highlighting
"

" jsx files
augroup filetypedetect
    au BufRead,BufNewFile *.jsx set filetype=javascript
augroup END

"
" User Interface
"

syntax reset
syntax on
colorscheme galea
set colorcolumn=100
hi ColorColumn ctermbg=237
hi SpellBad ctermbg=1 ctermfg=0

"Tab Bar
hi TabLine ctermfg=none ctermbg=237 cterm=none
hi TabLineFill ctermfg=none ctermbg=237 cterm=none
hi TabLineSel ctermfg=none ctermbg=242 cterm=none

" Quick Scope
hi QuickScopePrimary cterm=underline cterm=underline
hi QuickScopeSecondary cterm=underline cterm=underline

" Folds
highlight Folded ctermbg=none ctermfg=242
highlight FoldColumn ctermbg=none ctermfg=1

" ALE
hi ALEError cterm=underline cterm=italic ctermfg=1

if filereadable(expand("~/.vimrc.local"))
    so ~/.vimrc.local
endif


