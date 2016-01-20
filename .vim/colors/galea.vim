highlight clear
syntax reset
let g:colors_name = "Galea"

let RED = 1
let GREEN = 2 
let YELLOW = 3
let BLUE = 4
let ORANGE = 6
let BLACK = 0
let WHITE = "white"

highlight VertSplit ctermfg=DarkGray ctermbg=DarkGray
highlight Constant ctermfg=Blue
highlight Cursor ctermbg=Green

highlight Comment 	ctermfg=DarkGray	
highlight LineNr        ctermfg=DarkGray

" Variables
highlight Identifier ctermfg=6		
highlight Function ctermfg=WHITE

highlight Statement 	ctermfg=RED
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


" Markdown
highlight markdownHeadingDelimiter ctermfg=YELLOW
highlight Title ctermfg=YELLOW


"
" CSS
"

highlight scssProperty ctermfg=RED
highlight scssComment ctermfg=DarkGray
"Yellow
highlight scssAmpersand ctermfg=YELLOW
highlight scssSelector ctermfg=YELLOW
highlight scssSelectorChar ctermfg=YELLOW
highlight scssSelectorName ctermfg=YELLOW
highlight cssPseudoClassId ctermfg=YELLOW
highlight cssNoise ctermfg=YELLOW

"Red
highlight scssInclude ctermfg=RED
highlight scssFunctionName ctermfg=RED
highlight scssContent ctermfg=RED
highlight scssReturn ctermfg=RED
highlight scssExtend ctermfg=RED
highlight scssImport ctermfg=RED
highlight scssDebug ctermfg=RED
highlight scssWarn ctermfg=RED
highlight scssError ctermfg=RED
highlight scssIf ctermfg=RED
highlight scssElse ctermfg=RED
highlight scssElse ctermfg=RED
highlight scssWhile ctermfg=RED
highlight scssFor ctermfg=RED
highlight scssForKeyword ctermfg=RED
highlight scssAtRoot ctermfg=RED
highlight scssMixinName ctermfg=RED
highlight cssTextProp ctermfg=RED
highlight cssAnimationProp ctermfg=RED
highlight cssAuralProp ctermfg=RED
highlight cssBackgroundProp ctermfg=RED
highlight cssBorderProp ctermfg=RED
highlight cssBoxProp ctermfg=RED
highlight cssColorProp ctermfg=RED
highlight cssContentForPagedMediaProp ctermfg=RED
highlight cssDimensionProp ctermfg=RED
highlight cssFlexibleBoxProp ctermfg=RED
highlight cssFontDescriptorProp ctermfg=RED
highlight cssFontProp ctermfg=RED
highlight cssGeneratedContentProp ctermfg=RED
highlight cssGridProp ctermfg=RED
highlight cssHyerlinkProp ctermfg=RED
highlight cssIEUIProp ctermfg=RED
highlight cssLineboxProp ctermfg=RED
highlight cssListProp ctermfg=RED
highlight cssMarqueeProp ctermfg=RED
highlight cssMediaProp ctermfg=RED
highlight cssMobileTextProp ctermfg=RED
highlight cssMultiColumnProp ctermfg=RED
highlight cssPagedMediaProp ctermfg=RED
highlight cssPageProp ctermfg=RED
highlight cssPositioningProp ctermfg=RED
highlight cssPrintProp ctermfg=RED
highlight cssProp ctermfg=RED
highlight cssRenderProp ctermfg=RED
highlight cssRubyProp ctermfg=RED
highlight cssSpeechProp ctermfg=RED
highlight cssTableProp ctermfg=RED
highlight cssTextProp ctermfg=RED
highlight cssTransformProp ctermfg=RED
highlight cssTransitionProp ctermfg=RED
highlight cssUIProp ctermfg=RED
