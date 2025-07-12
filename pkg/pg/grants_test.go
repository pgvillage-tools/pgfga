package pg

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pkg/Pg/Grants", func() {
	var myConn Conn
	BeforeEach(func() {
		myConn = NewConn(ConnParams{})
	})
	Describe("Grants", func() {
		Context("reconcile", func() {
			grantees := []string{"grantee1", "grantee2"}
			granteds := []string{"granted1", "granted2"}
			It("should reconcile without error", func() {
				grants := Grants{}
				for _, grantee := range grantees {
					for _, granted := range granteds {
						grants = append(grants, Grant{
							Grantee: Role{Name: grantee},
							Granted: Role{Name: granted},
							State:   Present,
						})
					}
				}
				err := grants.reconcile(myConn)
				Ω(err).NotTo(HaveOccurred())
			})
			It("should have granted all grants", func() {
				for _, grantee := range grantees {
					for _, granted := range granteds {
						exists, err := Grant{

							Grantee: Role{Name: grantee},
							Granted: Role{Name: granted},
							State:   Present,
						}.exists(myConn)
						Ω(err).NotTo(HaveOccurred())
						Ω(exists).To(BeTrue())
					}
				}
			})
		})
		Context("finalize", func() {
			grantees := []string{"grantee1", "grantee2"}
			granteds := []string{"granted1", "granted2"}
			It("should finalise without error", func() {
				grants := Grants{}
				for _, grantee := range grantees {
					for _, granted := range granteds {
						grants = append(grants, Grant{
							Grantee: Role{Name: grantee},
							Granted: Role{Name: granted},
							State:   Absent,
						})
					}
				}
				err := grants.finalize(myConn)
				Ω(err).NotTo(HaveOccurred())
			})
			It("should have revoked all grants", func() {
				for _, grantee := range grantees {
					for _, granted := range granteds {
						exists, err := Grant{

							Grantee: Role{Name: grantee},
							Granted: Role{Name: granted},
							State:   Absent,
						}.exists(myConn)
						Ω(err).NotTo(HaveOccurred())
						Ω(exists).To(BeFalse())
					}
				}
			})
		})
	})
	Describe("Grant", func() {
		Context("String", func() {
			It("should be parsable to a string", func() {
				for _, grantee := range []string{"grantee1", "grantee2"} {
					for _, granted := range []string{"granted1", "granted2"} {
						stringVal := Grant{
							Grantee: Role{Name: grantee},
							Granted: Role{Name: granted},
						}.String()
						Ω(stringVal).To(ContainSubstring(granted))
						Ω(stringVal).To(ContainSubstring(grantee))
					}
				}
			})
		})
	})
})
