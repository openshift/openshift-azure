#!/bin/bash -ex

if [ $# -eq 1 ]; then
  echo "cleaning"
  for pid in $(ps -ef | egrep "(./monitoring)" | grep -v grep | awk '{print $2}'); do kill -15 $pid; done
  exit 0
fi

./monitoring -outputdir=$ARTIFACT_DIR &
