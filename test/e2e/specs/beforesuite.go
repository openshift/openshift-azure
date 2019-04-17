package specs

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	. "github.com/onsi/ginkgo"

	"github.com/onsi/ginkgo/config"

	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/util/log"
)

// rpFocus represents the supported RP APIs which e2e tests use to create their azure clients,
// The client will be configured to work either against the real, fake or admin apis
type rpFocus string

var (
	fakeRpFocus = rpFocus(regexp.QuoteMeta("[Fake]"))
	realRpFocus = rpFocus(regexp.QuoteMeta("[Real]"))
)

func (rpf rpFocus) match(focusString string) bool {
	return strings.Contains(focusString, string(rpf))
}

var _ = BeforeSuite(func() {
	var err error
	testlogger := log.GetTestLogger()
	focus := config.GinkgoConfig.FocusString
	switch {
	case fakeRpFocus.match(focus):
		fmt.Println("configuring the fake resource provider")
		err = azure.NewClientFromEnvironment(context.Background(), testlogger, false)
		if err != nil {
			testlogger.Error(err)
		}
	case realRpFocus.match(focus):
		fmt.Println("configuring the real resource provider")
		err = azure.NewClientFromEnvironment(context.Background(), testlogger, true)
		if err != nil {
			testlogger.Error(err)
		}
	default:
		panic(fmt.Sprintf("invalid focus %q - need to -ginkgo.focus=\\[Fake\\] or -ginkgo.focus=\\[Real\\]", config.GinkgoConfig.FocusString))
	}
	if azure.RPClient == nil {
		testlogger.Error("unable to provision either of the azure clients")
		panic("No Azure clients")
	}
})
