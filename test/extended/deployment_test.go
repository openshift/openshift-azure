package extended

import (
	"fmt"

	g "github.com/onsi/ginkgo"
	log "github.com/sirupsen/logrus"
)

var _ = g.Describe("Deployment Suite", func() {
	defer g.GinkgoRecover()

	// Create temp namespace based on ginko seed
	g.BeforeEach(func() {
		// Create temp namespace for tests
		buf, err := Asset("Namespaces/test.yaml")
		o, err := cl.Unmarshal(buf)
		if err != nil {
			g.Fail(err.Error())
		}

		o.SetName(fmt.Sprintf("test-%d", g.GinkgoRandomSeed()))
		cl.Create(o)
	})

	// Clean after test suite
	g.AfterEach(func() {
		if g.CurrentGinkgoTestDescription().Failed {
			//TODO: Dump logs, events
		}
		buf, err := Asset("Namespaces/test.yaml")
		o, err := cl.Unmarshal(buf)
		if err != nil {
			g.Fail(err.Error())
		}

		o.SetName(fmt.Sprintf("test-%d", g.GinkgoRandomSeed()))
		cl.Delete(o)
	})

	g.It("should deploy deployment", func() {
		for _, asset := range AssetNames() {
			log.Println(asset)
		}
		buf, err := Asset("Deployment.apps/simple-deployment.yaml")
		if err != nil {
			log.Errorf("Error %v", err)
			g.Fail(err.Error())
		}

		o, err := cl.Unmarshal(buf)
		if err != nil {
			g.Fail(err.Error())
		}

		o.SetNamespace(fmt.Sprintf("test-%d", g.GinkgoRandomSeed()))

		err = cl.Create(o)
		if err != nil {
			g.Fail(err.Error())
		}

		err = cl.PoolStatus(o)
		if err != nil {
			g.Fail(err.Error())
		}
	})

})
