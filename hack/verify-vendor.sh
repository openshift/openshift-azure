#!/bin/sh

if [ ! "$(command -v dep >/dev/null)" ]; then
  go get -u github.com/golang/dep/cmd/dep
fi

dep check
dep ensure
git diff --exit-code
