#!/bin/sh

set -e

function append_unique {
    grep -F "$1" "$2" || echo "$1" | sudo tee -a "$2"
}

cd `dirname $0`
# TODO: maybe use bg option of mount.nfs and append an entry to fstab
MOUNT_SCRIPT=`pwd`/mount-nfs-home.sh
( crontab -l 2>/dev/null | grep -vF "$MOUNT_SCRIPT" || true; echo "* * * * * $MOUNT_SCRIPT" ) >/tmp/zyme-crontab
crontab /tmp/zyme-crontab
rm /tmp/zyme-crontab

# hack up resolvable hostnames
for node in "$@"
do
    NODE_HOSTNAME=`ssh -o StrictHostKeyChecking=no $node hostname 2>/dev/null`
    append_unique "$node $NODE_HOSTNAME" /etc/hosts
    ssh -o StrictHostKeyChecking=no $NODE_HOSTNAME pwd 2>/dev/null 1>/dev/null
done
