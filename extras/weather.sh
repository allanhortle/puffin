#!/bin/zsh -f
source ~/.functions
weather | grep -o "\d\{1,3\}."
