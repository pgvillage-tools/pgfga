package pg

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func verifySchemaState(conn Conn, ext Schema) {
	exists, err := ext.exists(&conn)
	Ω(err).NotTo(HaveOccurred())
	if ext.State == Absent {
		Ω(exists).To(BeFalse())
		return
	}
	Ω(exists).To(BeTrue())
	if ext.Owner != "" {
		owner, err := ext.currentOwner(&conn)
		Ω(err).NotTo(HaveOccurred())
		Ω(owner).To(Equal(ext.Owner))
	}
}

var _ = Describe("Pkg/Pg/Schema", Ordered, func() {
	const (
		schema1       = "schema1"
		owner1        = "owner1"
		schema2       = "schema2"
		absentSchema  = "absent"
		presentSchema = "present"
		dbName        = "schematest"
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
			mySchemas := Schemas{
				schema1:       Schema{State: Present},
				schema2:       Schema{State: Present},
				presentSchema: Schema{State: Present},
				absentSchema:  Schema{State: Absent},
			}
			It("should succeed", func() {
				Ω(mySchemas.reconcile(&dbConn)).NotTo(HaveOccurred())
			})
			It("should only create Present Schemas", func() {
				for name, schema := range mySchemas {
					schema.name = name
					verifySchemaState(dbConn, schema)
				}
			})
		})
		Context("reconciling schema", func() {
			It("should change schema as required", func() {
				for _, owner := range []string{"owner1", "owner2"} {
					mySchema := Schema{name: schema1, Owner: owner, State: Present}
					Ω(mySchema.reconcileOwner(&dbConn)).NotTo(HaveOccurred())
					verifySchemaState(dbConn, mySchema)
				}
			})
		})
	})
})
