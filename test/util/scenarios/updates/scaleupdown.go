//+build e2e

package updates

import (
	. "github.com/onsi/gomega"

	"github.com/openshift/openshift-azure/test/util/client/azure"
)

func ScaleOutScaleSet(az *azure.Client, scaleSetName string, count int) {
	err := az.ScaleOutScaleSet(scaleSetName, count)
	Expect(err).NotTo(HaveOccurred())
}

func ScaleInScaleSet(az *azure.Client, scaleSetName string, count int) {
	err := az.ScaleInScaleSet(scaleSetName, count)
	Expect(err).NotTo(HaveOccurred())
}
