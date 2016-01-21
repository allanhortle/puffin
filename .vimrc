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

Plug 'gmarik/Vundle.vim'
Plug 'mattn/emmet-vim'
Plug 'isRuslan/vim-es6'
Plug 'scrooloose/nerdtree'
Plug 'scrooloose/nerdcommenter'
Plug 'pangloss/vim-javascript'
Plug 'mxw/vim-jsx'
Plug 'bling/vim-airline'
Plug 'tpope/vim-surround'
Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': './install --all' } 
Plug 'junegunn/fzf.vim'
Plug 'cakebaker/scss-syntax.vim'
Plug 'godlygeek/tabular'
Plug 'gabrielelana/vim-markdown'

call plug#end() 

"
" General Vim
"
set rtp+=~/.fzf
set spell spelllang=en_au
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
set hlsearch                    " highlight search terms
set incsearch                   " show search matches as you type

"
" Airline
"
let g:airline_left_sep=''
let g:airline_right_sep=''
let g:airline_theme='bubblegum'
set laststatus=2

" Enable the list of buffers
let g:airline#extensions#tabline#enabled = 1

" Show just the filename
let g:airline#extensions#tabline#fnamemod = ':t'


"
" Nerd Tree
"
map <C-o> :NERDTreeToggle<CR>
let NERDTreeShowHidden=1
let NERDTreeQuitOnOpen=1

" close if nerd tree is the only buffer
autocmd bufenter * if (winnr("$") == 1 && exists("b:NERDTree") && b:NERDTree.isTabTree()) | q | endif


"
" Keyboard Mappings
"
map <F10> :echo "hi<" . synIDattr(synID(line("."),col("."),1),"name") . '> trans<' . synIDattr(synID(line("."),col("."),0),"name") . "> lo<" . synIDattr(synIDtrans(synID(line("."),col("."),1)),"name") . ">"<CR>

" Normal mode
nnoremap <C-p> :Files<CR>
nnoremap <CR> :noh<CR><CR>
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

"
"Insert Mode
imap <c-x><c-l> <plug>(fzf-complete-line)

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
set statusline=%t[%{strlen(&fenc)?&fenc:'none'},%{&ff}]%h%m%r%y%=%c,%l/%L\ %P



syntax reset
syntax on
colorscheme galea

if filereadable("~/.vimrc.local")
    so '~/.vimrc.local'
endif
