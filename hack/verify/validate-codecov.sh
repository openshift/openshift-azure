#!/usr/bin/env bash

echo "The following directories contain Go source but no unit tests:"

rv=0
for dir in $(find cmd hack pkg -type d | grep -v ^pkg/util/mocks); do
	if [[ -n $(shopt -sq nullglob; echo $dir/*.go) && -z $(shopt -sq nullglob; echo $dir/*_test.go) ]]; then
		echo $dir
		rv=1
	fi
done

if [[ $rv -ne 0 ]]; then
	cat <<'EOF'

Preferably start adding some valid unit tests.  Worst case, at least add a file
called codecov_dummy_test.go with the following content:

==== 8< ====
package $PACKAGE

import (
	"testing"
)

// This file exists because this package has no unit tests.  Codecov does not
// report packages with no unit tests as 0% coverage, incorrectly inflating our
// coverage statistics.  When unit tests are added to this package, this file
// can be removed.

func TestDummyCodeCov(t *testing.T) {}
==== 8< ====

EOF
fi

exit $rv
