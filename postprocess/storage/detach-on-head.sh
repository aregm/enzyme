#!/bin/bash

set -e 

rm ~/.zyme-storage-node -f

sudo umount -l /storage

IFS=$'\n' # mark newline char as the only separator
for worker in $(cat ~/enzyme-nodefile)
do
    ssh $worker sudo umount -l /storage
done
unset $IFS # revert to default behaviour
