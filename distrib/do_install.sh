#!/bin/sh

set -e

cd `dirname $0`/$1

DISTR_FILE=`ls *.tgz`
DISTR_NAME=`echo $DISTR_FILE | sed 's/\.[^.]*$//'`

if [ ! -f ./$DISTR_NAME/install.sh ]; then
    tar xvf $DISTR_FILE
fi

cp silent.cfg silent-fixed.cfg
dos2unix silent-fixed.cfg

if grep -qE '^ACTIVATION_TYPE=license_file' silent-fixed.cfg ; then
    LICENSE_FILE=`ls *.lic`
    if [ -z "$LICENSE_FILE" ]; then
        echo "License file needed for $1 but is not uploaded"
        exit 1
    fi
    echo "ACTIVATION_LICENSE_FILE=`pwd`/$LICENSE_FILE" >>silent-fixed.cfg
fi

sudo ./$DISTR_NAME/install.sh --silent `pwd`/silent-fixed.cfg
rm -f silent-fixed.cfg

echo "$1 installed successfully"
