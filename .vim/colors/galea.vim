set background=dark
hi clear
if exists("syntax_on")
  syntax reset
endif
let g:colors_name = "Galea"

let RED = 1
let GREEN = 2 
let YELLOW = 3
let BLUE = 4
let MAGENTA = 5
let TEAL = 6
let BLACK = 0
let WHITE = "white"
let NONE = "NONE"

let GREY = 242
let LIGHT_GREY= 244

fun! <SID>GALEA_HI(group, fg, bg, attr)
    if a:fg != ""
        exec "hi " . a:group . " ctermfg=" . a:fg
    endif
    if a:bg != ""
        exec "hi " . a:group . " ctermbg=" . a:bg
    endif
    if a:attr != ""
        exec "hi " . a:group . " cterm=" . a:attr
    endif
endfun

call <SID>GALEA_HI('DiffAdd'      , GREEN  , NONE, '')
call <SID>GALEA_HI('DiffDelete'   , RED    , NONE, '')
call <SID>GALEA_HI('DiffChange'   , BLUE   , NONE, '')
call <SID>GALEA_HI('DiffText'     , WHITE  , NONE, '')

" Vim
call <SID>GALEA_HI('ColorColumn'  , RED    , ''         , '')
call <SID>GALEA_HI('ColorColumn'  , WHITE  , ''         , '')
call <SID>GALEA_HI('Comment'      , GREY   , ''         , '')
call <SID>GALEA_HI('Constant'     , BLUE   , ''         , '')
call <SID>GALEA_HI('CursorColumn' , WHITE  , ''         , '')
call <SID>GALEA_HI('CursorLine'   , WHITE  , ''         , '')
call <SID>GALEA_HI('Directory'    , WHITE  , ''         , '')
call <SID>GALEA_HI('FoldColumn'   , WHITE  , ''         , '')
call <SID>GALEA_HI('Folded'       , WHITE  , ''         , '')
call <SID>GALEA_HI('LineNr'       , GREY   , ''         , '')
call <SID>GALEA_HI('MatchParen'   , WHITE  , ''         , '')
call <SID>GALEA_HI('ModeMsg'      , WHITE  , ''         , '')
call <SID>GALEA_HI('MoreMsg'      , WHITE  , ''         , '')
call <SID>GALEA_HI('NonText'      , GREY   , ''         , '')
call <SID>GALEA_HI('PMenu'        , WHITE  , GREY       , '')
call <SID>GALEA_HI('PMenuSel'     , GREY   , GREEN      , '')
call <SID>GALEA_HI('Question'     , ''     , GREY       , '')
call <SID>GALEA_HI('Search'       , NONE   , LIGHT_GREY , '')
call <SID>GALEA_HI('SignColumn'   , WHITE  , ''         , '')
call <SID>GALEA_HI('SpecialKey'   , WHITE  , ''         , '')
call <SID>GALEA_HI('StatusLine'   , WHITE  , ''         , '')
call <SID>GALEA_HI('StatusLineNC' , WHITE  , ''         , '')
call <SID>GALEA_HI('SpellBad'     , RED    , NONE       , '')
call <SID>GALEA_HI('TabLine'      , WHITE  , ''         , '')
call <SID>GALEA_HI('VertSplit'    , GREY   , GREY       , '')
call <SID>GALEA_HI('Visual'       , NONE   , LIGHT_GREY , '')
call <SID>GALEA_HI('WarningMsg'   , WHITE  , ''         , '')

" Basic
call <SID>GALEA_HI('Identifier'  , TEAL      , '' , 'NONE')
call <SID>GALEA_HI('Function'    , WHITE, '' , '')
call <SID>GALEA_HI('Statement'   , WHITE  , '' , '')
call <SID>GALEA_HI('PreProc'     , WHITE    , '' , '')
call <SID>GALEA_HI('Conditional' , RED    , '' , '')
call <SID>GALEA_HI('Repeat'      , RED    , '' , '')
call <SID>GALEA_HI('Keyword'      , RED    , '' , '')
call <SID>GALEA_HI('Label'       , RED    , '' , '')
call <SID>GALEA_HI('Exception'   , RED    , '' , '')
call <SID>GALEA_HI('Special'     , RED    , '' , '')
call <SID>GALEA_HI('Constant'    , BLUE   , '' , '')
call <SID>GALEA_HI('Type'        , MAGENTA , '' , '')
call <SID>GALEA_HI('Meta'        , YELLOW , '' , '')
call <SID>GALEA_HI('String'      , 2      , '' , '')
call <SID>GALEA_HI('Character'   , 2      , '' , '')
call <SID>GALEA_HI('Number'      , BLUE   , '' , '')



" XML/HTML
call <SID>GALEA_HI('xmlTag'     , YELLOW , '' , '')
call <SID>GALEA_HI('xmlTagName' , YELLOW , '' , '')
call <SID>GALEA_HI('xmlEndTag'  , YELLOW , '' , '')
call <SID>GALEA_HI('xmlAttrib'  , YELLOW , '' , '')
call <SID>GALEA_HI('xmlEqual'   , YELLOW , '' , '')
call <SID>GALEA_HI('htmlTag'   , YELLOW , '' , '')
call <SID>GALEA_HI('htmlTagName'   , YELLOW , '' , '')
call <SID>GALEA_HI('htmlEndTag'   , YELLOW , '' , '')
call <SID>GALEA_HI('htmlTagN'   , BLUE , '' , '')


" Misc
call <SID>GALEA_HI('MatchParen', '', GREY, '')


" Javascript
call <SID>GALEA_HI('javaScriptReserved',  RED, '', '')
call <SID>GALEA_HI('javaScriptStatement', RED, '', '')
call <SID>GALEA_HI('javaScriptFuncKeyword', RED, '', '')


" Markdown
call <SID>GALEA_HI('htmlh1', YELLOW , '', '')
call <SID>GALEA_HI('htmlh2', YELLOW , '', '')
call <SID>GALEA_HI('htmlh3', YELLOW , '', '')
call <SID>GALEA_HI('Title', YELLOW , '' , '')
call <SID>GALEA_HI('markdownBlockquote', BLUE , '' , '')
call <SID>GALEA_HI('markdownBlockquoteDelimiter', BLUE , '' , '')
call <SID>GALEA_HI('markdownItemDelimiter', GREEN , '' , '')
call <SID>GALEA_HI('markdownItalic', TEAL , '' , '')
call <SID>GALEA_HI('markdownBold', TEAL , '' , '')
call <SID>GALEA_HI('markdownInlineDelimiter', TEAL , '' , '')


" CSS

call <SID>GALEA_HI('cssAnimationProp'            , RED    , '' , '')
call <SID>GALEA_HI('cssAuralProp'                , RED    , '' , '')
call <SID>GALEA_HI('cssBackgroundProp'           , RED    , '' , '')
call <SID>GALEA_HI('cssBorderProp'               , RED    , '' , '')
call <SID>GALEA_HI('cssBoxProp'                  , RED    , '' , '')
call <SID>GALEA_HI('cssColorProp'                , RED    , '' , '')
call <SID>GALEA_HI('cssContentForPagedMediaProp' , RED    , '' , '')
call <SID>GALEA_HI('cssDimensionProp'            , RED    , '' , '')
call <SID>GALEA_HI('cssFlexibleBoxProp'          , RED    , '' , '')
call <SID>GALEA_HI('cssFontDescriptorProp'       , RED    , '' , '')
call <SID>GALEA_HI('cssFontProp'                 , RED    , '' , '')
call <SID>GALEA_HI('cssGeneratedContentProp'     , RED    , '' , '')
call <SID>GALEA_HI('cssGridProp'                 , RED    , '' , '')
call <SID>GALEA_HI('cssHyerlinkProp'             , RED    , '' , '')
call <SID>GALEA_HI('cssIEUIProp'                 , RED    , '' , '')
call <SID>GALEA_HI('cssLineboxProp'              , RED    , '' , '')
call <SID>GALEA_HI('cssListProp'                 , RED    , '' , '')
call <SID>GALEA_HI('cssMarqueeProp'              , RED    , '' , '')
call <SID>GALEA_HI('cssMediaProp'                , RED    , '' , '')
call <SID>GALEA_HI('cssMobileTextProp'           , RED    , '' , '')
call <SID>GALEA_HI('cssMultiColumnProp'          , RED    , '' , '')
call <SID>GALEA_HI('cssNoise'                    , YELLOW , '' , '')
call <SID>GALEA_HI('cssPageProp'                 , RED    , '' , '')
call <SID>GALEA_HI('cssPagedMediaProp'           , RED    , '' , '')
call <SID>GALEA_HI('cssPositioningProp'          , RED    , '' , '')
call <SID>GALEA_HI('cssPrintProp'                , RED    , '' , '')
call <SID>GALEA_HI('cssProp'                     , RED    , '' , '')
call <SID>GALEA_HI('cssPseudoClassId'            , YELLOW , '' , '')
call <SID>GALEA_HI('cssRenderProp'               , RED    , '' , '')
call <SID>GALEA_HI('cssRubyProp'                 , RED    , '' , '')
call <SID>GALEA_HI('cssSpeechProp'               , RED    , '' , '')
call <SID>GALEA_HI('cssTableProp'                , RED    , '' , '')
call <SID>GALEA_HI('cssTagName'                  , YELLOW , '' , '')
call <SID>GALEA_HI('cssTextProp'                 , RED    , '' , '')
call <SID>GALEA_HI('cssTextProp'                 , RED    , '' , '')
call <SID>GALEA_HI('cssTransformProp'            , RED    , '' , '')
call <SID>GALEA_HI('cssTransitionProp'           , RED    , '' , '')
call <SID>GALEA_HI('cssUIProp'                   , RED    , '' , '')
call <SID>GALEA_HI('cssUnicodeEscape'            , GREEN    , '' , '')
call <SID>GALEA_HI('scssAmpersand'               , YELLOW , '' , '')
call <SID>GALEA_HI('scssAtRoot'                  , RED    , '' , '')
call <SID>GALEA_HI('scssComment'                 , GREY   , '' , '')
call <SID>GALEA_HI('scssContent'                 , RED    , '' , '')
call <SID>GALEA_HI('scssDebug'                   , RED    , '' , '')
call <SID>GALEA_HI('scssElse'                    , RED    , '' , '')
call <SID>GALEA_HI('scssElse'                    , RED    , '' , '')
call <SID>GALEA_HI('scssError'                   , RED    , '' , '')
call <SID>GALEA_HI('scssExtend'                  , RED    , '' , '')
call <SID>GALEA_HI('scssFor'                     , RED    , '' , '')
call <SID>GALEA_HI('scssForKeyword'              , RED    , '' , '')
call <SID>GALEA_HI('scssFunctionName'            , RED    , '' , '')
call <SID>GALEA_HI('scssIf'                      , RED    , '' , '')
call <SID>GALEA_HI('scssImport'                  , RED    , '' , '')
call <SID>GALEA_HI('scssInclude'                 , RED    , '' , '')
call <SID>GALEA_HI('scssMixinName'               , RED    , '' , '')
call <SID>GALEA_HI('scssProperty'                , RED    , '' , '')
call <SID>GALEA_HI('scssReturn'                  , RED    , '' , '')
call <SID>GALEA_HI('scssSelector'                , YELLOW , '' , '')
call <SID>GALEA_HI('scssSelectorChar'            , YELLOW , '' , '')
call <SID>GALEA_HI('scssSelectorName'            , YELLOW , '' , '')
call <SID>GALEA_HI('scssWarn'                    , RED    , '' , '')
call <SID>GALEA_HI('scssWhile'                   , RED    , '' , '')


" todo.txt

call <SID>GALEA_HI('TodoPriorityA', YELLOW, '', '')
call <SID>GALEA_HI('TodoPriorityB', GREEN, '', '')
call <SID>GALEA_HI('TodoPriorityC', BLUE, '', '')
call <SID>GALEA_HI('TodoProject', TEAL, '', '')
