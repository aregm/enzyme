#!/bin/bash

set -e

# first argument is login node (which is used to run this script), ignore it
shift

MPI_NODEFILE=~/enzyme-nodefile
# set up MPI nodefile
rm -f $MPI_NODEFILE

for worker in "$@"
do
    # force mounting $HOME via NFS at $WORKER
    ssh -o StrictHostKeyChecking=no $worker ~/zyme-postprocess/mount-nfs-home.sh
    # store $worker hostname in MPI nodefile
    WORKER_HOSTNAME=`ssh -o StrictHostKeyChecking=no $worker hostname 2>/dev/null`
    echo "$WORKER_HOSTNAME" >> $MPI_NODEFILE
done
