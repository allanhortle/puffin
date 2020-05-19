"
"   ██████╗ ██╗   ██╗███████╗███████╗██╗███╗   ██╗
"   ██╔══██╗██║   ██║██╔════╝██╔════╝██║████╗  ██║
"   ██████╔╝██║   ██║█████╗  █████╗  ██║██╔██╗ ██║
"   ██╔═══╝ ██║   ██║██╔══╝  ██╔══╝  ██║██║╚██╗██║
"   ██║     ╚██████╔╝██║     ██║     ██║██║ ╚████║
"   ╚═╝      ╚═════╝ ╚═╝     ╚═╝     ╚═╝╚═╝  ╚═══╝
"
"   File:   .vimrc
"

" {{{ Plugins
set nocompatible
if empty(glob('~/.vim/autoload/plug.vim'))
    silent !curl -fLo ~/.vim/autoload/plug.vim --create-dirs
        \ https://raw.githubusercontent.com/junegunn/vim-plug/master/plug.vim
    autocmd VimEnter * PlugInstall | source $MYVIMRC
endif
call plug#begin('~/.vim/plugged')

" Vim
Plug '/usr/local/opt/fzf' | Plug 'junegunn/fzf.vim'
Plug 'airblade/vim-gitgutter'
Plug 'chrisbra/csv.vim'
Plug 'docunext/closetag.vim'
Plug 'dyng/ctrlsf.vim'
Plug 'junegunn/fzf', { 'dir': '~/.fzf', 'do': './install --all' } 
Plug 'junegunn/vim-peekaboo'
Plug 'mattn/emmet-vim'
Plug 'mbbill/undotree'
Plug 'mhinz/vim-startify'
Plug 'mzlogin/vim-markdown-toc'
Plug 'neoclide/coc.nvim', {'branch': 'release'}
Plug 'scrooloose/nerdcommenter'
Plug 'scrooloose/nerdtree'
Plug 'tpope/vim-abolish'
Plug 'tpope/vim-fugitive'
Plug 'tpope/vim-markdown'
Plug 'tpope/vim-repeat'
Plug 'tpope/vim-surround'
Plug 'calviken/vim-gdscript3'
Plug 'aquach/vim-http-client'

" Syntax
Plug 'cakebaker/scss-syntax.vim'
Plug 'editorconfig/editorconfig-vim'
Plug 'jparise/vim-graphql'
Plug 'maxmellon/vim-jsx-pretty'
Plug 'pangloss/vim-javascript'

call plug#end() 

" }}}

" {{{ General Vim

set autoindent                  " always set auto-indenting on
set background=dark
set backspace=indent,eol,start  " allow backspacing over everything in insert mode
set complete=.,w,b,u,i          " turn off tab completion for tags
set copyindent                  " copy the previous indentation on auto-indenting
set encoding=utf8
set expandtab
set foldlevel=20
set hidden
set hlsearch
set ignorecase                  " ignore case when searching
set incsearch                   " show search matches as you type
set lazyredraw                  " dont redraw in the middle of a macro
set mouse=a
set nowrap
set number                      " always show line numbers
set rtp+=/usr/local/opt/fzf
set shiftround                  " use multiple of shiftwidth when indenting with '<' and '>'
set shiftwidth=4                " number of spaces to use for auto-indenting
set showmatch                   " set show matching 
set smartcase                   " ignore case if search pattern is all lower-case case-sensitive otherwise
set smarttab                    " insert tabs on the start of a line according to shiftwidth, not tabstop
set softtabstop=4
set splitright                  " set vertical splits to the right
set suffixesadd=.jsx,.md,.js
set t_Co=256
set t_BE=
set tabstop=4                   " a tab is four spaces
set timeoutlen=1000 ttimeoutlen=0
set updatetime=250
set wildmenu

set writebackup                 " protect against crash-during-write
set nobackup                    " but do not persist backup after successful write
set backupcopy=auto             " use rename-and-write-new method whenever safe
set undofile                    " persist the undo tree for each file
set backupdir^=~/.vim/backup//  " keep all the backup files in .vim
set undodir^=~/.vim/undo//
set noswapfile                  " dont have swap files, they are lame.



" plain text type file options
augroup WritingFiles
    autocmd!
    autocmd FileType markdown set wrap
    autocmd FileType markdown setlocal spell
    autocmd FileType gitcommit setlocal spell
augroup END

" Disable auto commenting
autocmd FileType * setlocal formatoptions-=c formatoptions-=r formatoptions-=o

filetype plugin on

" }}}

" {{{ Plugins

" ALE
let g:ale_linters = {'javascript': ['eslint', 'flow']}

" markdown
let g:markdown_fold_style = 'nested'

" emmet
"let g:user_emmet_expandabbr_key='<Tab>'
"imap <expr> <tab> emmet#expandAbbrIntelligent("\<tab>")



" FZF
let g:fzf_preview_window = ''


" Nerd Tree
map <C-o> :NERDTreeToggle<CR>
map <C-l> :NERDTreeFind<CR>
let NERDTreeShowHidden = 1
let NERDTreeQuitOnOpen = 1
let NERDTreeMinimalUI = 1
let NERDTreeDirArrows = 1
" close if nerd tree is the only buffer
autocmd bufenter * if (winnr("$") == 1 && exists("b:NERDTree") && b:NERDTree.isTabTree()) | q | endif

" netrw
let g:netrw_liststyle = 3
let g:netrw_banner = 0


" Startify
let g:startify_change_to_dir = 0
let g:startify_lists = [
    \ { 'type': 'dir',       'header': ['   MRU '. getcwd()] },
    \ { 'type': 'files',     'header': ['   MRU']            },
    \ { 'type': 'bookmarks', 'header': ['   Bookmarks']      },
    \ { 'type': 'commands',  'header': ['   Commands']       },
    \ ]


" ctrlsf
let g:ctrlsf_auto_focus = { "at": "start" }


" coc
call coc#add_extension(
    \ 'coc-flow',
    \ 'coc-tsserver',
    \ 'coc-eslint'
\)

function! s:check_back_space() abort
  let col = col('.') - 1
  return !col || getline('.')[col - 1]  =~# '\s'
endfunction

" shift tab navigate fowards
inoremap <silent><expr> <TAB> pumvisible() ? "\<C-n>" : <SID>check_back_space() ? "\<TAB>" : coc#refresh()
" shift tab navigate backwards
inoremap <expr><S-TAB> pumvisible() ? "\<C-p>" : "\<C-h>"
" cr accept completion
inoremap <expr> <cr> pumvisible() ? "\<C-y>" : "\<C-g>u\<CR>"
" Use `[g` and `]g` to navigate diagnostics
nmap <silent> [g <Plug>(coc-diagnostic-prev)
nmap <silent> ]g <Plug>(coc-diagnostic-next)

" Remap keys for gotos
nmap <silent> gd <Plug>(coc-definition)
nmap <silent> gy <Plug>(coc-type-definition)
nmap <silent> gi <Plug>(coc-implementation)
nmap <silent> gr <Plug>(coc-references)

" Use K to show documentation in preview window
nnoremap <silent> K :call CocAction('doHover')<CR>

" Highlight symbol under cursor on CursorHold
autocmd CursorHold * silent call CocActionAsync('highlight')
"autocmd CursorHold * silent call CocAction('doHover');

" Remap for rename current word
nmap <leader>rn <Plug>(coc-rename)


" }}}

" {{{ Keyboard Mapping

" print file path
nmap <F1> :echo expand('%:p')<cr>
" insert file path
imap <F1> <c-r>=expand("%:p")<cr>
" toggle pastemode
set pastetoggle=<F2>
" toggle wrap
map <F3> :set wrap!<CR>:set linebreak!<CR>
" toggle spell check
map <F6> :setlocal spell! spelllang=en_au<CR>
" print syntax names
map <F10> :echo "hi<" . synIDattr(synID(line("."),col("."),1),"name") . '> trans<' . synIDattr(synID(line("."),col("."),0),"name") . "> lo<" . synIDattr(synIDtrans(synID(line("."),col("."),1)),"name") . ">"<CR>
" save and run last terminal command
map <F12> <Esc>:w<CR>:!!<CR>

" Make help vertical
cnoreabbrev <expr> help ((getcmdtype() is# ':'    && getcmdline() is# 'help')?('vert help'):('help'))
cnoreabbrev <expr> h ((getcmdtype() is# ':'    && getcmdline() is# 'h')?('vert help'):('h'))


" Normal mode

" Wrapped navigation
nnoremap j gj
nnoremap k gk
nnoremap gj j
nnoremap gk k

nnoremap <Space> .
nnoremap <MiddleMouse> :call CocAction('doHover')<CR>
nnoremap <C-p> :Files<CR>
nnoremap <C-g> :Rg<CR>
nmap <C-f> <Plug>CtrlSFPrompt
vmap <C-f> <Plug>CtrlSFVwordExec
nnoremap <CR> :noh<CR><CR>
nnoremap <Leader>h :Startify<CR>
nnoremap <Leader>b :bp<CR>
nnoremap <Leader>e :CocList --normal -A diagnostics<CR>
nnoremap <Leader>f :bn<CR>
nnoremap <Leader>r :source $MYVIMRC<CR>
nnoremap <Leader>u :UndotreeToggle<CR>

nnoremap Q @@

" tig
nnoremap <Leader>tt :!tig<CR><CR>
nnoremap <Leader>ts :!tig status<CR><CR>
nnoremap <Leader>th :!tig %<CR><CR>
nnoremap <Leader>tb :execute '!tig blame % +' . line('.')<CR><CR>


" windows
nnoremap <Leader><Tab> <C-W>w

" window/buffer splitting
nnoremap <leader>s<left>   :leftabove  vnew<CR>
nnoremap <leader>s<right>  :rightbelow vnew<CR>
nnoremap <leader>s<up>     :leftabove  new<CR>
nnoremap <leader>s<down>   :rightbelow new<CR>


"
" Commands
"
command! -bang W w<bang>
command! -bang Q q<bang>
command! -bang Wq wq<bang>
command! -bang WQ wq<bang>
command! -bang Wqa wqa<bang>
command! -bang WQa wqa<bang>
command! -bang WQA wqa<bang>

"
" Custom Hardmode
"
noremap <Up> <NOP>
noremap <Down> <NOP>
noremap <Left> <NOP>
noremap <Right> <NOP>
noremap <BS> <NOP>
noremap <TAB> <NOP>


" }}}

" {{{ Syntax Highlighting

" jsx files
augroup filetypedetect
    au BufRead,BufNewFile *.jsx,*.mdx set filetype=javascript
augroup END
let g:javascript_plugin_flow = 1

syntax reset
syntax on
if &diff
    syntax off
endif
colorscheme puffin


" }}}


"

" 100 Column Ruler
set colorcolumn=100


"" statusline
let g:currentmode={
    \ 'n'      : 'normal',
    \ 'no'     : 'normal op',
    \ 'v'      : 'visual',
    \ 'V'      : 'vline',
    \ 's'      : 'select',
    \ 'S'      : 'sline',
    \ '\<C-S>' : 'sblock',
    \ 'i'      : 'insert',
    \ 'R'      : 'replace',
    \ 'Rv'     : 'vreplace',
    \ 'c'      : 'command',
    \ 'cv'     : 'vim ex',
    \ 'ce'     : 'ex',
    \ 'r'      : 'prompt',
    \ 'rm'     : 'more',
    \ 'r?'     : 'confirm',
    \ '!'      : 'shell',
    \ 't'      : 'terminal'
    \}

function! CurrentMode() abort
    let l:modecurrent = mode()
    let l:modelist = get(g:currentmode, l:modecurrent, 'vblock')
    let l:current_status_mode = l:modelist
    return l:current_status_mode
endfunction

function! ChangeStatuslineColor()
  let m = CurrentMode()
  if (m ==# "normal")
    exe 'hi! User1 ctermbg=blue'
elseif (m ==# "insert")
    exe 'hi! User1 ctermbg=green'
  elseif (m ==# 'vblock' || m ==# 'visual' || m ==# 'vline')
    exe 'hi! User1 ctermbg=magenta'
  else
    exe 'hi! User1 ctermbg=yellow'
  endif
  return ''
endfunction

function! ActiveStatus()
  let statusline=""
  let statusline.="%{ChangeStatuslineColor()}"
  let statusline.="%1* %{CurrentMode()} "
  let statusline.="%1* %(%-0.75f %M%)"
  let statusline.="%="
  let statusline.="%1*%( %r%w%y%)"
  let statusline.="%1* %v:%l "
  return statusline
endfunction

function! InactiveStatus()
  let statusline=""
  let statusline.="%{ChangeStatuslineColor()}"
  let statusline.="%2* %(%-0.75f %M%)"
  let statusline.="%="
  let statusline.="%2*%( %r%w%y%)"
  let statusline.="%2* %v:%l\ "
  return statusline
endfunction

set noshowmode " dont show --insert--
set laststatus=2 " ?
hi StatusLine ctermbg=black ctermfg=white cterm=none 
hi StatusLineNC ctermbg=black cterm=NONE
hi User1 ctermbg=blue ctermfg=black

" Inactive Styles
hi User2 ctermbg=grey ctermfg=black
set statusline=%!ActiveStatus()


augroup status
  autocmd!
  autocmd WinEnter * setlocal statusline=%!ActiveStatus()
  autocmd WinLeave * setlocal statusline=%!InactiveStatus()
  " more reliable insert color change
  autocmd InsertEnter * hi User1 ctermbg=green
  autocmd InsertLeave * hi User1 ctermbg=blue
augroup END

" Add local vimrc
if filereadable(expand("~/.vimrc.local"))
    so ~/.vimrc.local
endif


" vim:foldmethod=marker:foldlevel=0

" }}}



