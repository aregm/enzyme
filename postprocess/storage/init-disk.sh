#!/bin/bash

set -e

TARGET_CIDR="$1"

STORAGE_DEVICE=$(realpath /dev/disk/by-id/*storage_disk|sort -u|head -n 1)
STORAGE_PART="${STORAGE_DEVICE}1"

if [ ! -b "${STORAGE_PART}" ]; then
    # no partition table, create one
    sudo dd if=/dev/zero "of=${STORAGE_DEVICE}" bs=1M count=10
    sudo parted "${STORAGE_DEVICE}" mklabel gpt mkpart P1 ext4 1MiB 100%
fi

if ! sudo file -sL "${STORAGE_PART}" | grep -q ext4; then
    # no filesystem on the partition, make it
    sudo mkfs.ext4 "${STORAGE_PART}"
fi

sudo mkdir /storage -p

for uuid in /dev/disk/by-uuid/*
do
    if realpath "$uuid" | grep -q "${STORAGE_PART}"; then
        STORAGE_UUID=$(basename "$uuid")
        if ! grep -q "$STORAGE_UUID" /etc/fstab; then
            echo "UUID=${STORAGE_UUID} /storage ext4 defaults 0 0" | sudo tee -a /etc/fstab
        fi
        sudo mount /storage
    fi
done

if [ ! -z "${TARGET_CIDR}" ]; then
    # expose /storage via NFS
    NFS_EXPORT_LINE="/storage ${TARGET_CIDR}(rw,sync,no_root_squash)"
    if ! grep -q "${NFS_EXPORT_LINE}" /etc/exports; then
        echo "${NFS_EXPORT_LINE}" | sudo tee -a /etc/exports
    fi

    sudo /sbin/service nfs restart
    sudo exportfs -ra
fi
