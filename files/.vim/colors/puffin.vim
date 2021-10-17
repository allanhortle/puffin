set background=dark
hi clear
if exists("syntax_on")
  syntax reset
endif
let g:colors_name = "puffin"

" Make vert split stuff work
hi Normal cterm=none ctermfg=white ctermbg=none
"
" Basic
"
hi Comment ctermfg=242 cterm=italic
hi Constant ctermfg=blue cterm=none


hi Statement ctermfg=white cterm=none
hi Identifier ctermfg=white cterm=none
hi PreProc ctermfg=white cterm=none
hi Type ctermfg=white cterm=none
hi Special ctermfg=white cterm=none
hi Statement ctermfg=white cterm=none
hi Title ctermfg=white cterm=none

"
" vim UI
"
hi ColorColumn ctermbg=black
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
hi PMenu ctermfg=white ctermbg=234
hi PMenuSel ctermfg=black ctermbg=green
hi Question ctermfg=white
hi Search ctermbg=grey
hi SignColumn ctermbg=none
hi SpecialKey ctermfg=white
hi SpellBad ctermfg=red ctermbg=none cterm=underline
hi SpellCap ctermfg=yellow ctermbg=none cterm=underline
hi SpellLocal ctermfg=none ctermbg=none cterm=none
hi TabLine ctermfg=none ctermbg=none cterm=none
hi TabLineFill ctermfg=none ctermbg=none cterm=none
hi TabLineSel ctermfg=green ctermbg=none cterm=none
hi VertSplit ctermfg=white cterm=none ctermbg=none
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
hi jsEnvComment ctermfg=242

" keyword
hi jsAsyncKeyword ctermfg=red
hi jsBuiltins ctermfg=red
hi jsCatch ctermfg=red
hi jsClassKeyword ctermfg=red
hi jsClassMethodType ctermfg=red
hi jsConditional ctermfg=red
hi jsException ctermfg=red
hi jsExceptions ctermfg=red
hi jsExport ctermfg=red
hi jsExportDefault ctermfg=red
hi jsExtendsKeyword ctermfg=red
hi jsFlowImportType ctermfg=red
hi jsFrom ctermfg=red
hi jsFinally ctermfg=red
hi jsForAwait ctermfg=red
hi jsFunction ctermfg=red
hi jsFutureKeys ctermfg=red
hi jsGlobalNodeObjects ctermfg=red
hi jsGlobalObjects ctermfg=red
hi jsImport ctermfg=red
hi jsLabel ctermfg=red
hi jsModuleAs ctermfg=red
hi jsOperatorKeyword ctermfg=red
hi jsReturn ctermfg=red
hi jsRepeat ctermfg=red
hi jsStorageClass ctermfg=red
hi jsSuper ctermfg=red
hi jsStatement ctermfg=red
hi jsThis ctermfg=red
hi jsTry ctermfg=red
hi jsVariable ctermfg=red


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
hi jsNull ctermfg=blue
" meta
hi jsxOpenPunct ctermfg=yellow
hi jsxComponentName ctermfg=yellow
hi jsxAttrib ctermfg=yellow
hi jsxEqual ctermfg=yellow
hi jsxBraces ctermfg=yellow
hi jsxCloseString ctermfg=yellow
hi jsxClosePunct ctermfg=yellow
hi jsxTagName ctermfg=yellow
" types
hi jsFlowArgumentDef ctermfg=magenta
hi jsFlowArrow ctermfg=magenta
hi jsFlowClassGroup ctermfg=magenta
hi jsFlowDefinition ctermfg=magenta
hi jsFlowExactObject ctermfg=magenta
hi jsFlowFunctionGroup ctermfg=magenta
hi jsFlowClassFunctionGroup ctermfg=magenta
hi jsFlowGeneric ctermfg=magenta
hi jsFlowMaybe ctermfg=magenta
hi jsFlowNoise ctermfg=magenta
hi jsFlowObject ctermfg=magenta
hi jsFlowOrOperator ctermfg=magenta
hi jsFlowParens ctermfg=magenta
hi jsFlowParenAnnotation ctermfg=magenta
hi jsFlowType ctermfg=magenta
hi jsFlowTypeCustom ctermfg=magenta
hi jsFlowTypeStatement ctermfg=magenta
hi jsFlowTypeValue ctermfg=magenta
hi jsFlowWildcard ctermfg=magenta


"
" typescript
hi javaScriptLineComment ctermfg=242

hi typescriptAmbientDeclaration ctermfg=red
hi typescriptAsyncFuncKeyword ctermfg=red
hi typescriptBranch ctermfg=red
hi typescriptCase ctermfg=red
hi typescriptClassExtends ctermfg=red
hi typescriptClassKeyword ctermfg=red
hi typescriptClassStatic ctermfg=red
hi typescriptConditional ctermfg=red
hi typescriptDefault ctermfg=red
hi typescriptDeprecated ctermfg=red
hi typescriptExceptions ctermfg=red
hi typescriptExport ctermfg=red
hi typescriptFinally ctermfg=red
hi typescriptFuncKeyword ctermfg=red
hi typescriptGlobal ctermfg=red
hi typescriptGlobalObjects ctermfg=red
hi typescriptIdentifier ctermfg=red
hi typescriptImport ctermfg=red
hi typescriptInterfaceKeyword ctermfg=red
hi typescriptMessage ctermfg=red
hi typescriptMethodAccessor ctermfg=red
hi typescriptModule ctermfg=red
hi typescriptOperator ctermfg=red
hi typescriptPromiseStaticMethod ctermfg=red
hi typescriptReadonlyModifier ctermfg=red
hi typescriptRepeat ctermfg=red
hi typescriptReserved ctermfg=red
hi typescriptSource ctermfg=red
hi typescriptStatement ctermfg=red
hi typescriptStatementKeyword ctermfg=red
hi typescriptStorageClass ctermfg=red
hi typescriptVariable ctermfg=red
hi typescriptKeywordOp ctermfg=red
hi typescriptTry ctermfg=red
hi typescriptAccessibilityModifier ctermfg=red
hi typescriptForOperator ctermfg=red
hi typescriptExportType ctermfg=red
hi typescriptAsyncFor ctermfg=red

" constants
hi typescriptNull ctermfg=blue
hi typescriptBoolean ctermfg=blue
hi typescriptSpecial ctermfg=blue

hi typescriptString ctermfg=green
hi typescriptTemplate ctermfg=green
hi typescriptStringProperty ctermfg=green
hi typescriptTemplateSB ctermfg=green
hi typescriptStringLiteralType ctermfg=green
hi typescriptStringMember ctermfg=green
hi typescriptRegexpString ctermfg=green
hi tsxString ctermfg=green

hi typescriptTypeReference ctermfg=magenta
hi typescriptTypeBrackets ctermfg=magenta
hi typescriptPredefinedType ctermfg=magenta
hi typescriptAliasDeclaration ctermfg=magenta
hi typescriptAliasKeyword ctermfg=magenta
hi typescriptTypeAnnotation ctermfg=magenta
hi typescriptUnion ctermfg=magenta
hi typescriptUnion ctermfg=magenta
hi typescriptEndColons ctermfg=magenta
hi typescriptTypeBracket ctermfg=magenta
hi typescriptTypeParameter ctermfg=magenta
hi typescriptTypeQuery ctermfg=magenta
hi typescriptCastKeyword ctermfg=magenta
hi tsxTypeBraces ctermfg=magenta

hi tsxTagName ctermfg=yellow
hi tsxTag ctermfg=yellow
hi tsxAttrib ctermfg=yellow
hi tsxAttributeBraces ctermfg=yellow
hi tsxEqual ctermfg=yellow
hi tsxCloseTag ctermfg=yellow


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
hi graphqlStructure ctermfg=red
hi graphqlDirective ctermfg=red
hi graphqlString ctermfg=green
hi graphqlVariable ctermfg=yellow
hi graphqlConstant ctermfg=magenta
hi graphqlType ctermfg=magenta
hi graphqlOperator ctermfg=magenta


"
" scss
"

hi cssNoise ctermfg=white

" string
hi cssStringQ ctermfg=green
hi scssInterpolationDelimiter ctermfg=green

" constant
" keyword
hi cssBackgroundProp ctermfg=red
hi cssBorderProp ctermfg=red
hi cssBoxProp ctermfg=red
hi cssFlexibleBoxProp ctermfg=red
hi cssFontProp ctermfg=red
hi cssMediaProp ctermfg=red
hi cssPositioningProp ctermfg=red
hi cssTextProp ctermfg=red
hi cssTransformProp ctermfg=red
hi cssUIProp ctermfg=red
hi scssAtRoot ctermfg=red
hi scssContent ctermfg=red
hi scssMixin ctermfg=red
hi scssInclude ctermfg=red
" meta
hi cssPseudoClassId ctermfg=yellow
hi scssAmpersand ctermfg=yellow
hi scssSelectorChar ctermfg=yellow
hi scssSelectorName ctermfg=yellow



"
" diff

" git
hi diffAdded ctermfg=green ctermbg=NONE
hi diffRemoved ctermfg=red ctermbg=NONE
hi diffChanged ctermfg=blue ctermbg=NONE
hi diffText ctermfg=white ctermbg=NONE
hi diffFile ctermfg=yellow
hi diffIndexLine ctermfg=yellow
hi diffNewFile ctermfg=yellow
hi diffLine ctermfg=magenta
hi diffSubname ctermfg=magenta

" vimdiff
hi DiffAdd ctermfg=green ctermbg=none cterm=none
hi DiffDelete ctermfg=red ctermbg=none cterm=none
hi DiffChange ctermfg=white ctermbg=none cterm=none
hi DiffText ctermfg=blue ctermbg=none cterm=none



"
" markdown
hi markdownH1 ctermfg=yellow
hi markdownH1Delimiter ctermfg=yellow
hi markdownH2 ctermfg=green
hi markdownH2Delimiter ctermfg=green
hi markdownH3 ctermfg=blue
hi markdownH3Delimiter ctermfg=blue
hi markdownHeadingRule ctermfg=yellow

hi markdownCodeDelimiter ctermfg=magenta
hi markdownCode ctermfg=magenta

hi markdownItalic cterm=italic
hi markdownBold cterm=bold



"
" todo.txt
hi TodoPriorityA ctermfg=yellow
hi TodoPriorityB ctermfg=green
hi TodoPriorityC ctermfg=blue
hi TodoContext ctermfg=cyan
hi TodoProject ctermfg=magenta



"
" Vim Plug
hi plugName ctermfg=yellow
hi plugMessage ctermfg=blue
hi plugError ctermfg=red ctermbg=none


"
" SQL
hi sqlString ctermfg=green
hi sqlQuote ctermfg=green
hi sqlFold ctermfg=green
hi sqlStatement ctermfg=red
hi sqlKeyword ctermfg=red



"
" python
hi pythonStatement ctermfg=red
hi pythonRepeat ctermfg=red
hi pythonOperator ctermfg=red
hi pythonConditional ctermfg=red
hi pythonInclude ctermfg=red
hi pythonString ctermfg=green
hi pythonQuotes ctermfg=green


"
" xml
hi xmlTag ctermfg=yellow
hi xmlTagName ctermfg=yellow
hi xmlEqual ctermfg=yellow
hi xmlAttrib ctermfg=yellow
hi xmlDocTypeKeyword ctermfg=yellow
hi xmlDocTypeDecl ctermfg=yellow
hi xmlDocType ctermfg=yellow
hi xmlProcessingDelim ctermfg=yellow
hi xmlString ctermfg=green

hi shQuote ctermfg=green
hi shDoubleQuote ctermfg=green
hi shSingleQuote ctermfg=green

"
" Godot
hi gdString ctermfg=green
hi gdNumber ctermfg=blue
hi gdKeyword ctermfg=red
hi gdStatement ctermfg=red
hi gdConditional ctermfg=red



"
" ZSH
hi zshCommands ctermfg=red
hi zshKeyword ctermfg=red

hi zshString ctermfg=green
