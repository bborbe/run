// Copyright (c) 2020 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"
	"errors"

	"github.com/bborbe/run"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SkipErrors", func() {
	var err error
	var callCounter int
	BeforeEach(func() {
		callCounter = 0
		fn := run.SkipErrors(func(ctx context.Context) error {
			callCounter++
			return errors.New("banana")
		})
		err = fn(context.Background())
	})
	It("Returns no error", func() {
		Expect(err).To(BeNil())
	})
	It("calls fn", func() {
		Expect(callCounter).To(Equal(1))
	})
})

var _ = Describe("SkipErrorsAndReport", func() {
	var err error
	var callCounter int
	BeforeEach(func() {
		callCounter = 0
		fn := run.SkipErrorsAndReport(func(ctx context.Context) error {
			callCounter++
			return errors.New("banana")
		}, nil)
		err = fn(context.Background())
	})
	It("Returns no error", func() {
		Expect(err).To(BeNil())
	})
	It("calls fn", func() {
		Expect(callCounter).To(Equal(1))
	})
})
