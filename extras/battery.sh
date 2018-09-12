#!/bin/zsh -f
pmset -g batt | grep -o --color=never "[0-9]\{1,3\}%"
