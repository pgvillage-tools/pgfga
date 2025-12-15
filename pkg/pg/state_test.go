package pg

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pkg/Pg/State", func() {
	Context(".String()", func() {
		It("should return correct string and yaml values", func() {
			tests := []struct {
				input    State
				expected string
			}{
				{input: Present, expected: "Present"},
				{input: Absent, expected: "Absent"},
				{input: Allowed, expected: "Present"},
				{input: State{Present.value + Absent.value + Allowed.value}, expected: "Present"},
			}
			for _, test := range tests {
				Ω(test.input.String()).To(Equal(test.expected))
				marshalled, err := test.input.MarshalYAML()
				Ω(err).NotTo(HaveOccurred())
				Ω(marshalled).To(Equal(test.expected))
			}
		})
		It("should unmarshal correctly", func() {
			tests := []struct {
				input    string
				expected State
			}{
				{input: "Present", expected: Present},
				{input: "Absent", expected: Absent},
				{input: "Allowed", expected: Allowed},
				{input: "", expected: Present},
			}
			for _, test := range tests {
				var myState State
				unmarshal := func(val any) error {
					var myState = val.(*string)
					*myState = test.input
					return nil
				}
				err := myState.UnmarshalYAML(unmarshal)
				Ω(err).NotTo(HaveOccurred())
				Ω(myState).To(Equal(test.expected))
			}
		})
		It("should error out on unknown value", func() {
			unmarshal := func(val any) error {
				var myState = val.(*string)
				*myState = "invalid"
				return nil
			}
			var myState State
			Ω(myState.UnmarshalYAML(unmarshal)).To(HaveOccurred())
		})
		It("should error out when unmarshal func errors out", func() {
			unmarshal := func(_ any) error {
				return errors.New("unmarshal error error")
			}
			var myState State
			Ω(myState.UnmarshalYAML(unmarshal)).To(HaveOccurred())
		})
	})
})
