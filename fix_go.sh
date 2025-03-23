#!/bin/bash
# Disable goenv
unset GOENV_SHELL GOENV_ROOT GOPATH
# Remove goenv from PATH
PATH=$(echo "$PATH" | tr ':' '\n' | grep -v "goenv" | tr '\n' ':' | sed 's/:$//')
# Set correct Go environment
export GOROOT="/opt/homebrew/opt/go/libexec"
export GOPATH="$HOME/go"
# Show fixed environment
echo "PATH=$PATH"
echo "GOROOT=$GOROOT"
echo "GOPATH=$GOPATH"
