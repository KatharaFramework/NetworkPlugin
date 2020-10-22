#!/bin/bash

set -e
set -x

if [ ! -f "/host_etc/nftables.conf" ]; then
  update-alternatives --set iptables /usr/sbin/iptables-legacy
  update-alternatives --set ip6tables /usr/sbin/ip6tables-legacy
  update-alternatives --set arptables /usr/sbin/arptables-legacy
  update-alternatives --set ebtables /usr/sbin/ebtables-legacy

  if [ ! -f "/run/xtables.lock" ]; then
    ln -s /host_run/xtables.lock /run/xtables.lock
  fi
else
  update-alternatives --set iptables /usr/sbin/iptables-nft
  update-alternatives --set ip6tables /usr/sbin/ip6tables-nft
  update-alternatives --set arptables /usr/sbin/arptables-nft
  update-alternatives --set ebtables /usr/sbin/ebtables-nft
fi

./katharanp
