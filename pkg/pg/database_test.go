package pg

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Conn", func() {
	var myConn Conn
	BeforeEach(func() {
		myConn = NewConn(ConnParams{})
	})
	Describe("Managing databases", func() {
		Context("with default connection parameters", func() {
			It("should succeed", func() {
				connectError := myConn.Connect()
				Ω(connectError).NotTo(HaveOccurred())
				Ω(myConn.DBName()).NotTo(BeEmpty())
				Ω(myConn.UserName()).NotTo(BeEmpty())
				Ω(myConn.ConnParams()).To(BeEmpty())
			})
		})
	})
})
