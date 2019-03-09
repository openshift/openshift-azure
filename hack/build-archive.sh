#!/bin/bash

cd data

# RHEL tar doesn't include --sort=name, so have to use find

find -not -path ./archive.tgz -and -not -path ./script.sh -print0 \
  | LANG=C sort -z \
  | tar --no-recursion --null --files-from=- --owner=root --group=root --mtime=@1546300800 -c \
  | gzip -n \
  >archive.tgz
