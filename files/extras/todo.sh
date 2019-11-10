#!/bin/zsh -f
source ~/.todo.cfg
cat $TODO_FILE | grep "(A)" -c
