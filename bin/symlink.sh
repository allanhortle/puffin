#!/bin/bash
############################
# .make.sh
# This script creates symlinks from the home directory to any desired dotfiles in ~/dotfiles
############################

olddir=tmp
files="zshrc tigrc vimrc"

# create dotfiles_old in homedir
mkdir -p $olddir

# move any existing dotfiles in homedir to dotfiles_old directory, then create symlinks 
echo "Linking:"
for file in $files; do
    # echo "Moving $file to $olddir"
    mv ~/.$file $olddir
    ln -s $(pwd)/$file ~/.$file
    echo "    $file -> ~/.$file"
done