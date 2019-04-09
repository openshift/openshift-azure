package specs

import (
	"context"

	. "github.com/onsi/ginkgo"

	"github.com/openshift/openshift-azure/test/clients/azure"
	"github.com/openshift/openshift-azure/test/util/log"
)

var _ = BeforeSuite(func() {
	var err error
	testlogger := log.GetTestLogger()

	//so far none of the RealRP tests required the Storage Client
	azure.RealRPClient, err = azure.NewClientFromEnvironment(context.Background(), testlogger, false, true)
	testlogger.Debugf("new real rp client: %v", azure.RealRPClient)
	if err != nil {
		testlogger.Error(err)
	}
	azure.FakeRPClient, err = azure.NewClientFromEnvironment(context.Background(), testlogger, true, false)
	testlogger.Debugf("new fake rp client: %v", azure.FakeRPClient)
	if err != nil {
		testlogger.Error(err)
	}
	if azure.RealRPClient == nil && azure.FakeRPClient == nil {
		testlogger.Error("unable to provision either of the azure clients")
		panic("No Azure clients")
	}
})
