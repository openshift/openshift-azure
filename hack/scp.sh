#!/bin/bash -e

exec scp -S hack/ssh.sh "$@"
