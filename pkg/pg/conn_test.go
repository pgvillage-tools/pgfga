package pg_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pgvillage-tools/pgfga/pkg/pg"
)

var _ = Describe("Conn", func() {
	var myConn pg.Conn
	BeforeEach(func() {
		myConn = pg.NewConn(pg.ConnParams{})
	})
	Describe("Connecting", func() {
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
