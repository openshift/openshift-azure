#!/bin/bash -e

make content

if [[ -n "$(git status --porcelain)" ]]; then
	echo "content update produced template and image-stream changes that were not already present"
	
	. hack/tests/ci-operator-prepare.sh
	# HACK: using my repo for now to test 
	go run hack/giter/giter.go -sourcerepo mjudeikis/openshift-azure -targetrepo openshift/openshift-azure
else
	echo "Dependencies have no material difference than what is committed."
fi


