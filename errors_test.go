// Copyright (c) 2019 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package run_test

import (
	"context"
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
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
		Expect(err.Error()).To(Equal("[\ntest\n]"))
	})
	It("multi errors", func() {
		err := run.NewErrorList(fmt.Errorf("test1"), fmt.Errorf("test2"))
		Expect(err).NotTo(BeNil())
		Expect(err.Error()).To(Equal("[\ntest1\ntest2\n]"))
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
		Expect(err.Error()).To(Equal("[\ntest\n]"))
	})
	It("return nil if list is empty", func() {
		errors := make([]error, 0, 1)
		err := run.NewErrorList(errors...)
		Expect(err).To(BeNil())
	})
	Context("errors.Is", func() {
		var err error
		var target error
		var is bool
		JustBeforeEach(func() {
			is = errors.Is(err, target)
		})
		Context("same", func() {
			BeforeEach(func() {
				err = context.Canceled
				target = context.Canceled
			})
			It("returns true", func() {
				Expect(is).To(BeTrue())
			})
		})
		Context("array", func() {
			BeforeEach(func() {
				err = run.NewErrorList(context.Canceled)
				target = context.Canceled
			})
			It("returns true", func() {
				Expect(is).To(BeTrue())
			})
		})
	})
})
