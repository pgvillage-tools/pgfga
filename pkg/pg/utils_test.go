package pg

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Pkg/Pg/Utils", func() {
	Context("identifier", func() {
		It("should return correct identifiers", func() {
			tests := []struct {
				input    string
				expected string
			}{
				{input: "my-cool-value", expected: `"my-cool-value"`},
				{input: `something with a "`, expected: `"something with a """`},
				{input: `"`, expected: `""""`},
			}
			for _, test := range tests {
				Ω(identifier(test.input)).To(Equal(test.expected))
			}
		})
	})
	Context("identifier", func() {
		It("should return correct quoted SQL value", func() {
			tests := []struct {
				input    string
				expected string
			}{
				{input: "my-cool-value", expected: `'my-cool-value'`},
				{input: `something with a "`, expected: `'something with a "'`},
				{input: `something with a '`, expected: `'something with a '''`},
				{input: `'`, expected: `''''`},
			}
			for _, test := range tests {
				Ω(quotedSQLValue(test.input)).To(Equal(test.expected))
			}
		})
	})
})
