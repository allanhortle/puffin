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
Plug 'airblade/vim-gitgutter'
Plug 'ajh17/VimCompletesMe'
Plug 'amiorin/vim-project'
Plug 'cakebaker/scss-syntax.vim'
Plug 'christoomey/vim-sort-motion'
Plug 'docunext/closetag.vim'
Plug 'dyng/ctrlsf.vim'
Plug 'editorconfig/editorconfig-vim'
Plug 'gabrielelana/vim-markdown'
Plug 'freitass/todo.txt-vim'
Plug 'honza/vim-snippets'
Plug 'isRuslan/vim-es6'
Plug 'itchyny/lightline.vim'
Plug 'jceb/vim-orgmode'
Plug 'jeffkreeftmeijer/vim-numbertoggle'
Plug 'jiangmiao/auto-pairs'
Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': './install --all' } 
Plug '/usr/local/opt/fzf' | Plug 'junegunn/fzf.vim'
Plug 'junegunn/goyo.vim'
Plug 'mattn/emmet-vim'
Plug 'scrooloose/nerdcommenter'
Plug 'scrooloose/nerdtree', { 'on': 'NERDTreeToggle' }
"Plug 'sheerun/vim-polyglot'
Plug 'sjl/gundo.vim'
Plug 'tpope/vim-surround'
Plug 'tpope/vim-fugitive'
Plug 'tpope/vim-speeddating'
Plug 'tpope/vim-repeat'
Plug 'unblevable/quick-scope'
Plug 'bentayloruk/vim-react-es6-snippets'
Plug 'https://github.com/w0rp/ale.git'
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
set rtp+=~/.fzf
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
set wrap linebreak nolist

set timeoutlen=1000 ttimeoutlen=0

set nobackup                    " No backups.
set nowritebackup               " No backups.
set noswapfile                  " No swap files; more hassle than they're worth."

"
" ## PLUGINS ##
"

" Lightline
set noshowmode
set laststatus=2
let g:lightline = { 'colorscheme': 'jellybeans', }

" vim-project
set rtp+=~/.vim/bundle/vim-project/
let g:project_enable_welcome = 0
let g:project_use_nerdtree = 1
call project#rc()

" emmet
"let g:user_emmet_expandabbr_key='<Tab>'
"imap <expr> <tab> emmet#expandAbbrIntelligent("\<tab>")


" Nerd Tree
map <C-o> :NERDTreeToggle<CR>
let NERDTreeShowHidden=1
let NERDTreeQuitOnOpen=1
" close if nerd tree is the only buffer
autocmd bufenter * if (winnr("$") == 1 && exists("b:NERDTree") && b:NERDTree.isTabTree()) | q | endif


" Gundo
nnoremap <leader>u :GundoToggle<CR>

" RG
command! -bang -nargs=* Find call fzf#vim#grep('rg --column --line-number --no-heading --fixed-strings --ignore-case --hidden --follow --glob "!.git/*" --color "always" '.shellescape(<q-args>), 1, <bang>0)


"
" ## Keyboard Mappings ##
"
set pastetoggle=<F2>
map <F6> :setlocal spell! spelllang=en_au<CR>
map <F3> :set wrap!<CR>
map <F10> :echo "hi<" . synIDattr(synID(line("."),col("."),1),"name") . '> trans<' . synIDattr(synID(line("."),col("."),0),"name") . "> lo<" . synIDattr(synIDtrans(synID(line("."),col("."),1)),"name") . ">"<CR>

" Normal mode
nnoremap <C-p> :Files<CR>
nnoremap <C-f> :Find<CR>
nnoremap <C-i> :NERDTreeFind<CR>
nnoremap <CR> :noh<CR><CR>
nnoremap <Leader>p :Welcome<CR>
nnoremap <Leader>cp :let @* = expand("%")<CR>
nnoremap <Leader>l :ls<CR>
nnoremap <Leader>b :bp<CR>
nnoremap <Leader>e :ALENext<CR>
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

" Quick Scope
highlight QuickScopePrimary cterm=underline cterm=underline
highlight QuickScopeSecondary cterm=underline cterm=underline

" ALE
highlight ALEError cterm=underline cterm=italic ctermfg=1

if filereadable(expand("~/.vimrc.local"))
    so ~/.vimrc.local
endif


