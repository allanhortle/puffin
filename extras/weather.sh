#!/bin/zsh -f
source ~/.functions
weather | grep -o "[0-9.]\{1,3\}C"
