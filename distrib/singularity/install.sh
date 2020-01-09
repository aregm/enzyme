#!/bin/sh

set -e

cd `dirname $0`

SINGULARITY_VERSION=2.6.0
SINGULARITY_NAME=singularity-${SINGULARITY_VERSION}
tar -xvf ${SINGULARITY_NAME}.tar.gz
cd ${SINGULARITY_NAME}
./configure --prefix=/opt/${SINGULARITY_NAME}
make
sudo make install
sudo ln -s /opt/${SINGULARITY_NAME}/bin/singularity /usr/local/singularity

echo "$SINGULARITY_NAME installed successfully"
