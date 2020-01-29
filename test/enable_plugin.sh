#!/bin/bash

echo 'Stopping Docker Daemon...'
systemctl stop docker 

echo 'Running Kathara NP'
./katharanp 2>&1 > katharanp.log & 

echo 'Running Docker Daemon...'
systemctl start docker
tail -F katharanp.log
