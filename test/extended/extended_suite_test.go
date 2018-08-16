package extended

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var cl Interface
var err error

func TestExtended(t *testing.T) {

	cl, err = newClient()
	if err != nil {
		t.Fatal(err)
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Extended Suite")
}
