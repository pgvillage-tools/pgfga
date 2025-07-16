package pg

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func verifyExtState(conn Conn, ext Extension) {
	exists, err := ext.exists(&conn)
	Ω(err).NotTo(HaveOccurred())
	if ext.State == Absent {
		Ω(exists).To(BeFalse())
		return
	}
	Ω(exists).To(BeTrue())
	if ext.Schema != "" {
		schema, err := ext.currentSchema(&conn)
		Ω(err).NotTo(HaveOccurred())
		Ω(schema).To(Equal(ext.Schema))
	}
	if ext.Version != "" {
		version, err := ext.currentVersion(&conn)
		Ω(err).NotTo(HaveOccurred())
		Ω(version).To(Equal(ext.Version))
	}
}

var _ = Describe("Pkg/Pg/Extension", Ordered, func() {
	const (
		ext1        = "pg_stat_statements"
		ext1Version = "1.5"
		ext1Schema  = "schema1"
		ext2        = "pgcrypto"
		ext3        = "hstore"
		dbName      = "exttest"
	)
	var (
		myConn Conn
		dbConn Conn
	)

	BeforeAll(func() {
		myConn = NewConn(ConnParams{})
	})
	BeforeEach(func() {
		db := Database{name: dbName}
		Ω(db.drop(myConn)).NotTo(HaveOccurred())
		Ω(db.create(myConn)).NotTo(HaveOccurred())
		dbConn = myConn.SwitchDB(dbName)
	})
	When("managing extensions", func() {
		Context("reconciling extensions", Ordered, func() {
			myExtensions := Extensions{
				ext1: Extension{Version: ext1Version, Schema: ext1Schema,
					State: Present},
				ext2: Extension{State: Present},
				ext3: Extension{State: Absent},
			}
			It("should succeed", func() {
				Ω(myExtensions.reconcile(&dbConn)).NotTo(HaveOccurred())
			})
			It("should only create Present Extensions", func() {
				for name, ext := range myExtensions {
					ext.name = name
					verifyExtState(dbConn, ext)
				}
			})
		})
		Context("reconciling schema", func() {
			It("should change schema as required", func() {
				for _, schema := range []string{"schema1", "schema2"} {
					myExt := Extension{name: ext1, Schema: schema, State: Present}
					Ω(myExt.reconcileSchema(&dbConn)).NotTo(HaveOccurred())
					verifyExtState(dbConn, myExt)
				}
			})
		})
		Context("reconciling version", func() {
			It("should change vesion as required", func() {
				for _, version := range []string{"1.5", "1.6"} {
					myExt := Extension{name: ext1, Version: version, State: Present}
					Ω(myExt.reconcileVersion(&dbConn)).NotTo(HaveOccurred())
					verifyExtState(dbConn, myExt)
				}
			})
		})
	})
})
