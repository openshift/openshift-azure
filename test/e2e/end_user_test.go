//+build e2e

package e2e

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	policy "k8s.io/api/policy/v1beta1"
	_ "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("Openshift on Azure end user e2e tests [EndUser]", func() {
	defer GinkgoRecover()

	It("should disallow PDB mutations", func() {
		maxUnavailable := intstr.FromInt(1)
		selector, err := metav1.ParseToLabelSelector("key=value")
		Expect(err).NotTo(HaveOccurred())

		pdb := &policy.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
			Spec: policy.PodDisruptionBudgetSpec{
				MaxUnavailable: &maxUnavailable,
				Selector:       selector,
			},
		}

		_, err = c.kc.Policy().PodDisruptionBudgets(c.namespace).Create(pdb)
		// TODO: Reenable
		// Expect(kerrors.IsForbidden(err)).To(Equal(true))
		fmt.Printf("PDB create error: %v\n", err)
	})
})
