#!/bin/bash

"`dirname $0`/mount-nfs-dir.sh" "$HOME/.zyme-login-node" "$HOME"
exit $?