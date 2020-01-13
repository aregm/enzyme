#!/bin/sh

set -e

sudo yum install -y yum-utils
sudo yum-config-manager --add-repo https://yum.repos.intel.com/clck/2019/setup/intel-clck-2019.repo
sudo yum-config-manager --add-repo https://yum.repos.intel.com/mpi/setup/intel-mpi.repo
sudo yum-config-manager --add-repo https://yum.repos.intel.com/mkl/setup/intel-mkl.repo

sudo rpm --import https://yum.repos.intel.com/intel-gpg-keys/GPG-PUB-KEY-INTEL-SW-PRODUCTS-2019.PUB

sudo yum install -y intel-clck-2019.6-038
sudo yum install -y intel-mpi-2019.5-075
sudo yum install -y intel-mkl-2019.5-075
