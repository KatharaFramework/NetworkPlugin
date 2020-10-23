#!/bin/bash

set -x

# Read where the host iptables points
host_iptables_path=$(readlink /host_sbin/iptables)
echo $host_iptables_path | grep 'alternatives' &> /dev/null
# Points to /etc/alternatives
if [ $? == 0 ]; then
  # Update the symlink to work into the container
  container_iptables_path=$(echo $host_iptables_path | sed "s/etc/host_etc/g")
  # Read the real iptables path
  host_iptables_path=$(readlink $container_iptables_path)
fi

# If the host iptables executable contains "xtables" or does not contain "nft", switch "legacy" into the container
echo $host_iptables_path | grep 'xtables' &> /dev/null
use_xtables=$?
echo $host_iptables_path | grep 'nft' &> /dev/null
use_nft=$?

if [ $use_xtables == 0 ] || [ $use_nft != 0 ]; then
  update-alternatives --set iptables /usr/sbin/iptables-legacy
  update-alternatives --set ip6tables /usr/sbin/ip6tables-legacy

  # Link the host xtables.lock into the container /run
  if [ ! -f "/run/xtables.lock" ]; then
    ln -s /host_run/xtables.lock /run/xtables.lock
  fi
fi

./katharanp