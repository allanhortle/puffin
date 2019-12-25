set background=dark
hi clear
if exists("syntax_on")
  syntax reset
endif
let g:colors_name = "puffin"

"
" Basic
"
hi Constant ctermfg=blue cterm=none
hi Identifier ctermfg=white cterm=none
hi PreProc ctermfg=white cterm=none
hi Type ctermfg=white cterm=none
hi Statement ctermfg=white cterm=none
hi Special ctermfg=white cterm=none
hi Title ctermfg=white cterm=none

"
" vim UI
"
hi ColorColumn ctermbg=238
hi Comment ctermfg=242
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

"
" vim
"
hi vimCommand ctermfg=red
hi vimString ctermfg=green
hi vimHiCtermColor ctermfg=blue
hi vimFgBgAttrib ctermfg=blue
hi vimHiAttrib ctermfg=blue
hi vimNotation ctermfg=yellow
hi vimBracket ctermfg=yellow
hi vimMapModKey ctermfg=yellow


"
" js
"

" keyword
hi jsVariable ctermfg=red
hi jsFunction ctermfg=red
hi jsImport ctermfg=red
hi jsExport ctermfg=red
hi jsFrom ctermfg=red
hi jsReturn ctermfg=red
hi jsStorageClass ctermfg=red
hi jsExportDefault ctermfg=red
hi jsClassKeyword ctermfg=red
hi jsExtendsKeyword ctermfg=red
hi jsSuper ctermfg=red
hi jsGlobalObjects ctermfg=red
hi jsGlobalNodeObjects ctermfg=red
hi jsOperatorKeyword ctermfg=red
hi jsBuiltins ctermfg=red
hi jsThis ctermfg=red
hi jsAsyncKeyword ctermfg=red
hi jsException ctermfg=red
" string
hi jsTemplateString ctermfg=green
hi jsTemplateBraces ctermfg=green
hi jsString ctermfg=green
hi jsxString ctermfg=green
hi jsObjectStringKey ctermfg=green
hi jsRegexpOr ctermfg=green
hi jsRegexpMod ctermfg=green
hi jsRegexpGroup ctermfg=green
hi jsRegexpString ctermfg=green
hi jsRegexpBackRef ctermfg=green
" constant
hi jsSpecial ctermfg=blue
hi jsUndefined ctermfg=blue
" meta
hi jsxOpenPunct ctermfg=yellow
hi jsxComponentName ctermfg=yellow
hi jsxAttrib ctermfg=yellow
hi jsxEqual ctermfg=yellow
hi jsxBraces ctermfg=yellow
hi jsxCloseString ctermfg=yellow
hi jsxClosePunct ctermfg=yellow
" types
hi jsFlowTypeStatement ctermfg=magenta
hi jsFlowNoise ctermfg=magenta
hi jsFlowObject ctermfg=magenta
hi jsFlowMaybe ctermfg=magenta
hi jsFlowType ctermfg=magenta
hi jsFlowDefinition ctermfg=magenta
hi jsFlowTypeCustom ctermfg=magenta
hi jsFlowTypeValue ctermfg=magenta
hi jsFlowClassGroup ctermfg=magenta
hi jsFlowGeneric ctermfg=magenta
hi jsFlowOrOperator ctermfg=magenta
hi jsFlowFunctionGroup ctermfg=magenta
hi jsFlowArgumentDef ctermfg=magenta


"
" json
"
hi jsonString ctermfg=green
hi jsonEscape ctermfg=blue
hi jsonKeyword ctermfg=yellow
hi jsonQuote ctermfg=green


"
" yaml
"
hi yamlBlockMappingKey ctermfg=yellow
hi yamlPlainScalar ctermfg=green
hi yamlFlowString ctermfg=green
hi yamlFlowStringDelimiter ctermfg=green



"
" graphql
"
"keyword
hi graphqlStructure ctermfg=red
"string
hi graphqlString ctermfg=green
"constant
"meta
hi graphqlVariable ctermfg=yellow
"type
hi graphqlConstant ctermfg=magenta
hi graphqlType ctermfg=magenta
hi graphqlOperator ctermfg=magenta


"
" scss
"

" string
hi cssStringQ ctermfg=green
hi scssInterpolationDelimiter ctermfg=green
" constant
hi scssAttribute ctermfg=blue
" keyword
hi scssMixin ctermfg=red
hi scssContent ctermfg=red
hi cssBoxProp ctermfg=red
hi cssFlexibleBoxProp ctermfg=red
hi cssMediaProp ctermfg=red
hi cssBorderProp ctermfg=red
hi cssPositioningProp ctermfg=red
hi cssBackgroundProp ctermfg=red
" meta
hi scssSelectorName ctermfg=yellow
hi scssSelectorChar ctermfg=yellow
hi scssAmpersand ctermfg=yellow



"
" diff
"
hi DiffAdd ctermfg=green ctermbg=NONE
hi DiffDelete ctermfg=red ctermbg=NONE
hi DiffChange ctermfg=blue ctermbg=NONE
hi DiffText ctermfg=white ctermbg=NONE

"
" todo.txt
"
hi TodoPriorityA ctermfg=yellow
hi TodoPriorityB ctermfg=green
hi TodoPriorityB ctermfg=blue
hi TodoProject ctermfg=cyan



