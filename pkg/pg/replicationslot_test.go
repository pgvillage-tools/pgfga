package pg

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func repSlotExists(conn Conn, name string) {
	exists, err := replicationSlot{name: name}.exists(conn)
	Ω(err).NotTo(HaveOccurred())
	Ω(exists).To(BeTrue())
}

func repSlotNotExists(conn Conn, name string) {
	exists, err := replicationSlot{name: name}.exists(conn)
	Ω(err).NotTo(HaveOccurred())
	Ω(exists).To(BeFalse())
}

var _ = Describe("Pkg/Pg/Replicationslot", Ordered, func() {
	const (
		shouldExist    = "exists"
		shouldNotExist = "nonexistent"
	)
	var (
		myConn Conn
	)
	repSlots := replicationSlots{
		shouldExist: replicationSlot{
			name: shouldExist, State: Present},
		shouldNotExist: replicationSlot{
			name: shouldNotExist, State: Absent},
	}

	BeforeAll(func() {
		myConn = NewConn(ConnParams{})
		for _, rs := range []string{shouldExist, shouldNotExist} {
			Ω(replicationSlot{name: rs, State: Absent}.drop(myConn)).NotTo(
				HaveOccurred())
		}
	})
	BeforeEach(func() {
	})
	Describe("Managing databases", func() {
		Context("reconciling", func() {
			It("should succeed", func() {
				Ω(repSlots.reconcile(myConn)).NotTo(HaveOccurred())
			})
			It("has created repslots with State Present", func() {
				repSlotExists(myConn, shouldExist)
			})
			It("has not created repslots with State Absent", func() {
				repSlotNotExists(myConn, shouldNotExist)
			})
		})
		Context("finalizing", func() {
			It("should succeed", func() {
				Ω(repSlots.finalize(myConn)).NotTo(HaveOccurred())
			})
			It("should have not removed extensions with State Present", func() {
				repSlotExists(myConn, shouldExist)
			})
			It("should have removed databases with State Absent", func() {
				repSlotNotExists(myConn, shouldNotExist)
			})
		})
	})
})
