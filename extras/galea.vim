highlight clear
syntax reset
let g:colors_name = "Galea"

let RED = 1
let GREEN = 2 
let YELLOW = 3
let BLUE = 4
let ORANGE = 6
let WHITE = "white"

highlight Constant ctermfg=Blue
highlight Cursor ctermbg=Green

highlight Comment 	ctermfg=DarkGray	
highlight LineNr        ctermfg=DarkGray

" Variables
highlight Identifier ctermfg=6		
highlight Function ctermfg=WHITE

highlight Statement 	ctermfg=WHITE
highlight Conditional 	ctermfg=RED
highlight Repeat 	ctermfg=RED
highlight Label 	ctermfg=RED
highlight Exception 	ctermfg=RED

highlight Constant 	ctermfg=BLUE

"highlight Type 		ctermfg=DarkGray
highlight String	ctermfg=2
highlight Character	ctermfg=2
highlight Number	ctermfg=BLUE
" highlight Boolean	ctermfg=DarkGrey
" highlight Float		ctermfg=DarkGrey

" highlight Special 	ctermfg=DarkGray	
" highlight Todo 		ctermfg=DarkGray
"
" XML
highlight xmlTag        ctermfg=YELLOW
highlight xmlTagName     ctermfg=YELLOW
highlight xmlEndTag    ctermfg=YELLOW
highlight xmlAttrib    ctermfg=YELLOW
highlight xmlEqual    ctermfg=YELLOW

" Misc
highlight MatchParen ctermbg=GREEN 

" Javascript
highlight javaScriptReserved ctermfg=RED 
