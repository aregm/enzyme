#!/bin/sh

set -e

if [ ! -f ~/.zyme-login-node ]; then
    exit 0
fi

cd ~
HOME_SERVER=`cat ~/.zyme-login-node`
grep -v "$HOME" /proc/mounts && /usr/sbin/showmount -e "$HOME_SERVER" | grep "$HOME" && sudo mount -t nfs "$HOME_SERVER:$HOME" "$HOME"
