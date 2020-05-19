set background=dark
hi clear
if exists("syntax_on")
  syntax reset
endif
let g:colors_name = "puffin-basic"

"
" Basic
"
" resets
hi Identifier ctermfg=white cterm=none
hi Special ctermfg=white cterm=none

" colors
hi Comment ctermfg=242

hi Constant ctermfg=green cterm=none
hi String ctermfg=green cterm=none
hi Character ctermfg=green cterm=none
hi Number ctermfg=blue cterm=none
hi Boolean ctermfg=blue cterm=none
hi Float ctermfg=blue cterm=none

hi Statement ctermfg=red cterm=none
hi Conditional ctermfg=red cterm=none
hi Repeat ctermfg=red cterm=none
hi Label ctermfg=red cterm=none
hi Operator ctermfg=red cterm=none
hi Keyword ctermfg=red cterm=none
hi Exception ctermfg=red cterm=none

hi PreProc ctermfg=yellow cterm=none
hi Include ctermfg=yellow cterm=none
hi Define ctermfg=yellow cterm=none
hi Macro ctermfg=yellow cterm=none
hi PreCondit ctermfg=yellow cterm=none

hi Type ctermfg=magenta cterm=none
hi StorageClass ctermfg=magenta cterm=none
hi Structure ctermfg=magenta cterm=none
hi Typedef ctermfg=magenta cterm=none



"hi Statement ctermfg=white cterm=none
"hi Identifier ctermfg=white cterm=none
"hi PreProc ctermfg=white cterm=none
"hi Type ctermfg=white cterm=none
"hi Special ctermfg=white cterm=none
"hi Statement ctermfg=white cterm=none
"hi Title ctermfg=white cterm=none



"
" vim UI
"
hi ColorColumn ctermbg=238
hi Directory ctermfg=white
hi Error ctermbg=red ctermfg=black
hi ErrorMsg ctermfg=red ctermbg=black
hi FoldColumn ctermbg=none ctermfg=1
hi Folded ctermbg=none ctermfg=242
hi LineNr ctermfg=242
hi MatchParen ctermfg=white ctermbg=grey
hi ModeMsg ctermfg=white
hi MoreMsg ctermfg=white
hi NonText ctermfg=242
hi PMenu ctermfg=white ctermbg=darkgrey
hi PMenuSel ctermfg=black ctermbg=green
hi Question ctermfg=white
hi Search ctermbg=grey
hi SignColumn ctermfg=white
hi SpecialKey ctermfg=white
hi SpellBad ctermfg=red ctermbg=none cterm=underline
hi SpellCap ctermfg=yellow ctermbg=none cterm=underline
hi SpellLocal ctermfg=none ctermbg=none cterm=none
hi TabLine ctermfg=none ctermbg=none cterm=underline
hi TabLineFill ctermfg=none ctermbg=none cterm=underline
hi TabLineSel ctermfg=black ctermbg=grey
hi VertSplit ctermfg=0 ctermbg=white
hi Visual ctermbg=240
hi WarningMsg ctermfg=yellow ctermbg=black
hi StartifyHeader ctermfg=yellow 
hi StartifySection ctermfg=242 

