#!/bin/sh

TIMES="$1"

cd ~
sudo chown $USER ~/DrivAer -R
sed --in-place --regexp-extended "s/(endTime\s*)\s([0-9]+)([^0-9]+)/\1 $TIMES\3/g" DrivAer/system/controlDict

./runDrivAer.sh

