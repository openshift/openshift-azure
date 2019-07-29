#!/bin/bash

cd data

# Normalise the archive accross different OS. We need consistent results in CI and in . Known issues:
#  * RHEL `tar` doesn't include `--sort=name`, so have to use `find`
#  * MacOS sets GUID different to 0 even if you specify `--group=root` in `tar`,
#    so we have to explicitly set GUID to 0
#  * MacOS sets different file mode for symlinks, so we have to explicitly set permissions for them

TMP_TAR=$(mktemp)
TAR_COMMON_OPTIONS="--no-recursion --null --files-from=- --owner=root --group=root --owner=0 --group=0 --mtime=@1546300800"

find -not -path ./archive.tgz -and -not -path ./script.sh -and -not -type l -print0 \
  | LANG=C sort -z \
  | tar $TAR_COMMON_OPTIONS --mode=g+w,g-s -cf $TMP_TAR

find -type l -print0 \
  | LANG=C sort -z \
  | tar $TAR_COMMON_OPTIONS --mode=a+rwx --append -f $TMP_TAR


gzip -n < $TMP_TAR > archive.tgz

rm $TMP_TAR
