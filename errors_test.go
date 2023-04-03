// Copyright (c) 2019 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package run_test

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/bborbe/run"
)

var _ = Describe("Errors", func() {
	It("TestNewEmptyError", func() {
		err := run.NewErrorList()
		Expect(err).To(BeNil())
	})
	It("TestNewErorrList", func() {
		err := run.NewErrorList(fmt.Errorf("test"))
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(Equal("errors: [test]"))
	})
	It("multi errors", func() {
		err := run.NewErrorList(fmt.Errorf("test1"), fmt.Errorf("test2"))
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(Equal("errors: [test1, test2]"))
	})
	It("TestNewByChanEmptyError", func() {
		c := make(chan error, 10)
		close(c)
		err := run.NewErrorListByChan(c)
		Expect(err).To(BeNil())
	})
	It("TestNewByChanErorrList", func() {
		c := make(chan error, 10)
		c <- fmt.Errorf("test")
		close(c)
		err := run.NewErrorListByChan(c)
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(Equal("errors: [test]"))
	})
	It("return nil if list is empty", func() {
		errors := make([]error, 0, 1)
		err := run.NewErrorList(errors...)
		Expect(err).To(BeNil())
	})
})
