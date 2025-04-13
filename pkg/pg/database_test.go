package pg_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pgvillage-tools/pgfga/pkg/pg"
)

var _ = Describe("Conn", func() {
	var myConn *pg.Conn
	BeforeEach(func() {
		myConn = pg.NewConn(pg.DSN{})
	})
	Describe("Managing databases", func() {
		Context("with default connection parameters", func() {
			It("should succeed", func() {
				connectError := myConn.Connect()
				Expect(connectError).NotTo(HaveOccurred())
				Expect(myConn.DBName()).NotTo(BeEmpty())
				Expect(myConn.UserName()).NotTo(BeEmpty())
				Expect(myConn.DSN()).To(BeEmpty())
			})
		})
	})
})
