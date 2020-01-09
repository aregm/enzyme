#!/bin/bash

set -e

if ! echo $PACKER_BUILD_NAME | grep -q amazon ; then
    echo "This is not an Amazon image, not doing Amazon-specific preprocessing"
    exit 0
fi

# enlarge / partition
sudo yum install -y cloud-utils-growpart xfsprogs

ROOT_PART_DEV=`df / | grep dev | awk '{print $1}'`
ROOT_DISK_DEV=`lsblk --paths --output 'KNAME,PKNAME' | grep "$ROOT_PART_DEV" | awk '{print $2}'`
ROOT_PART_NUM=`lsblk --paths --output 'KNAME,PKNAME' | grep -E "\s$ROOT_DISK_DEV" | grep -ne "$ROOT_PART_DEV" | cut -d ':' -f 1`

df -h / | grep "$ROOT_PART_DEV" | awk '{print $2}' | grep -q "${USER_DISK_SIZE}G" || sudo growpart "${ROOT_DISK_DEV}" "${ROOT_PART_NUM}"
sudo xfs_growfs -d /

# Enable C5/M5 support

# Installation script follows ideas from here: https://gist.github.com/Ray33/ba189a729d81babc99d7cef0fb6fbcd8
sudo yum --enablerepo=extras -y install epel-release
sudo yum update -y
sudo yum -y install dos2unix unzip wget make patch dkms kernel-devel perl

export FUTURE_KERNEL_VERSION=$(ls /usr/src/kernels/ | tail -n 1)

mkdir ~/c5
cd ~/c5
wget https://codeload.github.com/amzn/amzn-drivers/tar.gz/ena_linux_1.5.3 -O ena_linux_1.5.3.tar.gz
wget https://github.com/awslabs/aws-support-tools/archive/master.zip -O aws-support-tools-master.zip
tar zxvf ena_linux_1.5.3.tar.gz
unzip aws-support-tools-master.zip
sudo mv amzn-drivers-ena_linux_1.5.3 /usr/src/ena-1.5.3

cat <<EOF > /usr/src/ena-1.5.3/dkms.conf
PACKAGE_NAME="ena"
PACKAGE_VERSION="1.5.3"
AUTOINSTALL="yes"
REMAKE_INITRD="yes"
BUILT_MODULE_LOCATION[0]="kernel/linux/ena"
BUILT_MODULE_NAME[0]="ena"
DEST_MODULE_LOCATION[0]="/updates"
DEST_MODULE_NAME[0]="ena"
CLEAN="cd kernel/linux/ena; make clean"
MAKE="cd kernel/linux/ena; make BUILD_KERNEL=\${kernelver}"
EOF

sudo dkms add -m ena -v 1.5.3 -k $FUTURE_KERNEL_VERSION
sudo dkms build -m ena -v 1.5.3 -k $FUTURE_KERNEL_VERSION
sudo dkms install -m ena -v 1.5.3 -k $FUTURE_KERNEL_VERSION
sudo dracut -f --kver $FUTURE_KERNEL_VERSION --add-drivers "ena nvme"

# now verify that image is C5-ready
dos2unix ~/c5/aws-support-tools-master/EC2/C5M5InstanceChecks/c5_m5_checks_script.sh
# patch up the script so it checker would-be-booted kernel instead of currently running
sed -e "s:uname -r:echo \"$FUTURE_KERNEL_VERSION\":g" ~/c5/aws-support-tools-master/EC2/C5M5InstanceChecks/c5_m5_checks_script.sh > ~/c5/check_next_kernel.sh

chmod +x ~/c5/check_next_kernel.sh
if ! sudo ~/c5/check_next_kernel.sh; then
    echo "Image preparation for C5 failed!"
    exit 10
fi

# Now following this guide, points 6 and 9:
# https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/enhanced-networking-ena.html#enhanced-networking-ena-linux
sudo rm -f /etc/udev/rules.d/70-persistent-net.rules
if grep -q GRUB_CMDLINE_LINUX /etc/default/grub; then
    sudo sed -i '/^GRUB\_CMDLINE\_LINUX/s/\"$/\ net\.ifnames\=0\"/' /etc/default/grub
else
    echo 'GRUB_CMDLINE_LINUX=net.ifnames=0' | sudo tee -a /etc/default/grub
fi
sudo grub2-mkconfig -o /boot/grub2/grub.cfg

rm -rf ~/c5
