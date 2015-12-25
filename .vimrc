"
" Vundle
"
set nocompatible              " be iMproved, required
filetype off                  " required
set rtp+=~/.vim/bundle/Vundle.vim
call vundle#begin()

Plugin 'gmarik/Vundle.vim'
Plugin 'mattn/emmet-vim'
Plugin 'isRuslan/vim-es6'
Plugin 'kien/ctrlp.vim'
Plugin 'Lokaltog/vim-easymotion'
Plugin 'terryma/vim-multiple-cursors'
Plugin 'scrooloose/nerdtree'
Plugin 'pangloss/vim-javascript'
Plugin 'mxw/vim-jsx'
Plugin 'bling/vim-airline'
Plugin 'tpope/vim-surround'

call vundle#end()            " required
filetype plugin indent on    " required


"
" Airline
"
let g:airline_left_sep=''
let g:airline_right_sep=''
let g:airline_theme='bubblegum'


"
" Ctrl P
"
let g:ctrlp_custom_ignore = 'node_modules\|DS_Store\|git'
let g:ctrlp_show_hidden = 1


"
" Nerd Tree
"
map <C-o> :NERDTreeToggle<CR>
let NERDTreeShowHidden=1
"let g:NERDTreeWinPos='right'
let NERDTreeQuitOnOpen=1

" close if nerd tree is the only buffer
autocmd bufenter * if (winnr("$") == 1 && exists("b:NERDTree") && b:NERDTree.isTabTree()) | q | endif

" Open Nerd tree on open if empty buffer
autocmd StdinReadPre * let s:std_in=1
autocmd VimEnter * if argc() == 0 && !exists("s:std_in") | NERDTree | endif


"
" Keyboard Mappings
"
nnoremap <CR> :noh<CR><CR>
map <F10> :echo "hi<" . synIDattr(synID(line("."),col("."),1),"name") . '> trans<' . synIDattr(synID(line("."),col("."),0),"name") . "> lo<" . synIDattr(synIDtrans(synID(line("."),col("."),1)),"name") . ">"<CR>


"
" Syntax Highlighting
"

" jsx files
let g:jsx_ext_required = 0
autocmd BufNewFile,BufRead *.json set syntax=javascript


"
" UI
"
set background=dark
set expandtab
set shiftwidth=4
set softtabstop=4
set number
set hlsearch
set splitright

" status line
set statusline=%t[%{strlen(&fenc)?&fenc:'none'},%{&ff}]%h%m%r%y%=%c,%l/%L\ %P

syntax reset
syntax on
colorscheme galea

if filereadable("~/.vimrc.local")
    so '~/.vimrc.local'
endif
