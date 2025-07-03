package pg

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Dsn", func() {
	var myDSN ConnParams
	BeforeEach(func() {
		myDSN = ConnParams{"host": "myhost", "port": "5433"}
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
