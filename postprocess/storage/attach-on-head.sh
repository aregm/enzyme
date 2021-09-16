#!/bin/bash

set -e

STORAGE_INTERNAL_NODE_IP="$1"

echo ${STORAGE_INTERNAL_NODE_IP} > ~/.zyme-storage-node

~/zyme-postprocess/storage/attach-storage-node.sh

IFS=$'\n' # mark newline char as the only separator
for worker in $(cat ~/enzyme-nodefile)
do
    ssh $worker ~/zyme-postprocess/storage/attach-storage-node.sh
done
unset $IFS # revert to default behaviour
