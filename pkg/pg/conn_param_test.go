package pg_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pgvillage-tools/pgfga/pkg/pg"
)

var _ = Describe("Dsn", func() {
	var myDSN pg.ConnParams
	BeforeEach(func() {
		myDSN = pg.ConnParams{"host": "myhost", "port": "5433"}
	})
	Describe("When instantiating a new DSN object", func() {
		Context("with a few keys set", func() {
			It("We should be able to get the DSN as a string", func() {
				Ω(myDSN.String()).To(Equal("host='myhost' port='5433'"))
			})
		})
	})
	Describe("When cloning an existing DSN object", func() {
		Context("with a few keys set", func() {
			It("the clone should have the same key/value pairs as the original DSN", func() {
				myDSNClone := myDSN.Clone()
				for key, value := range myDSN {
					Ω(myDSNClone).To(HaveKey(key))
					Ω(myDSNClone).To(ContainElement(value))
				}
			})
		})
	})
})
