package specs

import (
	"context"
	"errors"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	buildv1 "github.com/openshift/api/build/v1"
	_ "github.com/openshift/origin/pkg/api/install"
	buildclientmanual "github.com/openshift/origin/pkg/build/client/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	"github.com/openshift/openshift-azure/test/sanity"
)

var _ = Describe("Openshift on Azure end user e2e tests [EndUser][Builds][EveryPR]", func() {

	It("should be able to create builds [EndUser][Builds]", func() {
		ctx := context.Background()
		namespace, err := sanity.Checker.CreateProject(ctx)
		name := namespace + "-build"

		Expect(err).To(BeNil())
		defer func() {
			By("deleting project")
			_ = sanity.Checker.DeleteProject(ctx, namespace)
		}()

		By("creating a binary BuildConfig")
		bc := &buildv1.BuildConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: map[string]string{"build": name},
			},
			Spec: buildv1.BuildConfigSpec{
				CommonSpec: buildv1.CommonSpec{
					Source: buildv1.BuildSource{
						Type: buildv1.BuildSourceBinary,
					},
					Strategy: buildv1.BuildStrategy{
						DockerStrategy: &buildv1.DockerBuildStrategy{},
					},
				},
			},
		}
		build, buildErr := sanity.Checker.Client.EndUser.BuildV1.BuildConfigs(namespace).Create(bc)
		Expect(buildErr).To(BeNil())

		By("validating starting a build for a binary BuildConfig")

		buildClient := sanity.Checker.Client.EndUser.BuildV1.RESTClient()

		buildRequestOptions := &buildv1.BinaryBuildRequestOptions{
			ObjectMeta: metav1.ObjectMeta{
				Name:      build.Name,
				Namespace: build.Namespace,
			},
			AsFile: "Dockerfile",
		}
		instantiateClient := buildclientmanual.NewBuildInstantiateBinaryClient(buildClient, namespace)

		r := strings.NewReader("FROM scratch")
		buildResult, err := instantiateClient.InstantiateBinary(name, buildRequestOptions, r)
		Expect(err).To(BeNil())

		buildName := buildResult.GetName()

		By("validating build completed for a binary BuildConfig")
		var b *buildv1.Build
		err = wait.PollImmediate(2*time.Second, 5*time.Minute, func() (bool, error) {
			// Wait for build to complete
			// Verify build succeeded
			b, err = sanity.Checker.Client.EndUser.BuildV1.Builds(namespace).Get(buildName, metav1.GetOptions{})

			switch {
			case kerrors.IsNotFound(err):
				return false, nil
			case err != nil:
				return false, err
			}

			switch b.Status.Phase {
			case "New", "Pending", "Running":
				return false, nil
			case "Complete", "Failed", "Error", "Canceled":
				return true, nil
			}

			return false, errors.New("Unexpected build phase")

		})
		Expect(err).ToNot(HaveOccurred())
		Expect(b.Status.Phase).To(BeEquivalentTo("Complete"))

	})
})
