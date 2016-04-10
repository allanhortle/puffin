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

Plug 'vim-airline/vim-airline'
Plug 'vim-airline/vim-airline-themes'
Plug 'cakebaker/scss-syntax.vim'
Plug 'docunext/closetag.vim'
Plug 'editorconfig/editorconfig-vim'
Plug 'gabrielelana/vim-markdown'
Plug 'godlygeek/tabular'
Plug 'honza/vim-snippets'
Plug 'isRuslan/vim-es6'
Plug 'jceb/vim-orgmode'
Plug 'jeffkreeftmeijer/vim-numbertoggle'
Plug 'jiangmiao/auto-pairs'
Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': './install --all' } 
Plug 'junegunn/fzf.vim'
Plug 'junegunn/goyo.vim'
Plug 'mattn/emmet-vim'
Plug 'mxw/vim-jsx'
Plug 'morhetz/gruvbox'
Plug 'pangloss/vim-javascript'
Plug 'scrooloose/nerdcommenter'
Plug 'scrooloose/nerdtree'
Plug 'scrooloose/syntastic'
Plug 'sheerun/vim-polyglot'
Plug 'sirver/ultisnips'
Plug 'tpope/vim-surround'
Plug 'tpope/vim-fugitive'
Plug 'tpope/speeddating'
Plug 'vim-scripts/camelcasemotion'
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
set hlsearch                    " highlight search terms
set incsearch                   " show search matches as you type
set t_Co=256
set mouse=a


"
" ## PLUGINS ##
"

" Airline
let g:airline_left_sep=''
let g:airline_right_sep=''
let g:airline_theme='term'
set laststatus=2
let g:airline#extensions#tabline#enabled = 1
let g:airline#extensions#tabline#fnamemod = ':t'


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
