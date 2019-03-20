#!/bin/bash -ex

if [ $# -eq 1 ]; then
  echo "cleaning"
  for pid in $(ps -ef | egrep "(./monitoring)" | grep -v grep | awk '{print $2}'); do kill -2 $pid; done
  exit 0
fi

./monitoring &
