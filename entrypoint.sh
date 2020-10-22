#!/bin/bash

set -e
set -x

/host_sbin/iptables --version | grep 'nf_tables' &> /dev/null
if [ $? == 0 ]; then
  update-alternatives --set iptables /usr/sbin/iptables-nft
  update-alternatives --set ip6tables /usr/sbin/ip6tables-nft
else
  update-alternatives --set iptables /usr/sbin/iptables-legacy
  update-alternatives --set ip6tables /usr/sbin/ip6tables-legacy
  if [ ! -f "/run/xtables.lock" ]; then
    ln -s /host_run/xtables.lock /run/xtables.lock
  fi
fi

./katharanp
