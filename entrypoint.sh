#!/bin/bash

set -e
set -x

if [ ! -f "/tmp/xtables.lock" ]; then
  if [ -f "/host_run/xtables.lock" ]; then
    ln -s /host_run/xtables.lock /tmp/xtables.lock
  else
	  touch /tmp/xtables.lock && chmod 600 /tmp/xtables.lock && chown root:root /tmp/xtables.lock
  fi
fi

./katharanp
