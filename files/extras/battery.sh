#!/bin/zsh -f
BATTERY=$(pmset -g batt | grep -o --color=never "[0-9]\{1,3\}%")
if [[ -z "$BATTERY" ]]
then
    echo "AC"
else
    echo $BATTERY
fi
