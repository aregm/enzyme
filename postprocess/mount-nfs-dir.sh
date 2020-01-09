#!/bin/bash

set -e

SERVER_FILE="$1"
TARGET_DIR="$2"

if [ ! -f "${SERVER_FILE}" ]; then
    exit 0
fi

SERVER=$(cat "${SERVER_FILE}")
grep -q -v "${TARGET_DIR}" /proc/mounts && /usr/sbin/showmount -e "${SERVER}" | grep -q "${TARGET_DIR}" && sudo mount -t nfs "${SERVER}:${TARGET_DIR}" "${TARGET_DIR}" || echo "${TARGET_DIR} already mounted"
