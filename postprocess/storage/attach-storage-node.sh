#!/bin/bash

set -e 

cd `dirname $0`/../
MOUNT_LINE="\"`pwd`/mount-nfs-dir.sh\" \"$HOME/.zyme-storage-node\" \"/storage\""

sudo mkdir -p /storage
./mount-nfs-dir.sh "$HOME/.zyme-storage-node" /storage

( crontab -l 2>/dev/null | grep -vF "$MOUNT_LINE" || true; echo "* * * * * $MOUNT_LINE" ) >/tmp/zyme-crontab
crontab /tmp/zyme-crontab
rm /tmp/zyme-crontab
