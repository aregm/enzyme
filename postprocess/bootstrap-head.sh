#!/bin/bash

set -e

function append_unique {
    grep -F "$1" "$2" || echo "$1" | sudo tee -a "$2"
}

# setting up NFS mount
cd ~
NFS_EXPORT_LINE="`pwd` $ZYME_CLUSTER_CIDR(rw,sync,no_root_squash)"
append_unique "$NFS_EXPORT_LINE" /etc/exports
sudo /sbin/service nfs restart
sudo exportfs -ra

mkdir -p ~/clck
#set up nodefile and hack up resolvable hostnames
echo "`hostname` #role: head" > ~/clck/nodefile
append_unique "$1 `hostname`" /etc/hosts
shift

cat <<'EOF' >~/clck/cluster_checker.sh
#!/bin/sh
source /opt/intel/clck/2019.0/bin/clckvars.sh
/opt/intel/clck/2019.0/bin/intel64/clck -f ~/clck/nodefile -F ssf_compat-hpc-2016.0 -o ~/clck/clck_results.log
EOF

chmod +x ~/clck/*.sh

for worker in "$@"
do
    WORKER_HOSTNAME=`ssh -o StrictHostKeyChecking=no $worker hostname 2>/dev/null`
    append_unique "$worker $WORKER_HOSTNAME" /etc/hosts
    ssh -o StrictHostKeyChecking=no "$WORKER_HOSTNAME" pwd 2>/dev/null 1>/dev/null
    echo "$WORKER_HOSTNAME" >> ~/clck/nodefile
done
