package pg

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func dbExists(conn Conn, dbName string) {
	exists, err := conn.runQueryExists(
		"select datname from pg_database where datname = $1",
		dbName,
	)
	Ω(err).NotTo(HaveOccurred())
	Ω(exists).To(BeTrue())
}

func dbNotExists(conn Conn, dbName string) {
	exists, err := conn.runQueryExists(
		"select datname from pg_database where datname = $1",
		dbName,
	)
	Ω(err).NotTo(HaveOccurred())
	Ω(exists).NotTo(BeTrue())
}

var _ = Describe("Conn", Ordered, func() {
	const (
		shouldExist    = "should-exist"
		shouldNotExist = "should-not-exist"
		dbName         = "dbtest"
	)
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
			dbs := Databases{
				dbName:         Database{State: Present},
				shouldExist:    Database{State: Present},
				shouldNotExist: Database{State: Absent},
			}
			It("should succeed", func() {
				Ω(dbs.reconcile(myConn)).NotTo(HaveOccurred())
			})
			It("should have created databases with State Present", func() {
				dbExists(myConn, shouldExist)
				dbExists(myConn, dbName)
			})
			It("should have not have created databases with State Absent",
				func() {
					dbNotExists(myConn, shouldNotExist)
				})
		})
		Context("finalizing", func() {
			dbs := Databases{
				dbName:         Database{State: Absent},
				shouldExist:    Database{State: Present},
				shouldNotExist: Database{State: Absent},
			}
			It("should succeed", func() {
				Ω(dbs.finalize(myConn)).NotTo(HaveOccurred())
			})
			It("should have not removed databases with State Present", func() {
				dbExists(myConn, shouldExist)
			})
			It("should have removed databases with State Absent", func() {
				dbNotExists(myConn, dbName)
				dbNotExists(myConn, shouldNotExist)
			})
		})
	})
})
