package pg

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func roleExists(conn Conn, dbName string) {
	exists, err := Database{name: dbName}.exists(conn)
	Ω(err).NotTo(HaveOccurred())
	Ω(exists).To(BeTrue())
}

func roleNotExists(conn Conn, dbName string) {
	exists, err := Database{name: dbName}.exists(conn)
	Ω(err).NotTo(HaveOccurred())
	Ω(exists).NotTo(BeTrue())
}

var _ = Describe("Pkg/Pg/Role", Ordered, func() {
	const (
		shouldExist    = "should-exist"
		shouldNotExist = "should-not-exist"
	)
	roles := Roles{
		shouldExist:    Role{State: Present},
		shouldNotExist: Role{State: Absent},
	}
	var (
		myConn Conn
	)

	BeforeAll(func() {
		myConn = NewConn(ConnParams{})
	})
	BeforeEach(func() {
	})
	Describe("Managing databases", func() {
		Context("reconciling", func() {
			It("should succeed", func() {
				Ω(roles.reconcile(myConn)).NotTo(HaveOccurred())
			})
			It("should have created databases with State Present", func() {
				roleExists(myConn, shouldExist)
			})
			It("should have not have created databases with State Absent",
				func() {
					roleNotExists(myConn, shouldNotExist)
				})
		})
		Context("finalizing", func() {
			It("should succeed", func() {
				Ω(roles.finalize(myConn)).NotTo(HaveOccurred())
			})
			It("should have not removed databases with State Present", func() {
				dbExists(myConn, shouldExist)
			})
			It("should have removed databases with State Absent", func() {
				roleNotExists(myConn, shouldNotExist)
			})
		})
	})
})
