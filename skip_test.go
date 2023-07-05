// Copyright (c) 2020 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package run_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/bborbe/run"
	"github.com/bborbe/run/mocks"
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
	var sentryClient *mocks.HasCaptureErrorAndWait
	var ctx context.Context
	BeforeEach(func() {
		ctx = context.Background()
		callCounter = 0
		sentryClient = &mocks.HasCaptureErrorAndWait{}

	})
	Context("without error", func() {
		BeforeEach(func() {
			fn := run.SkipErrorsAndReport(
				func(ctx context.Context) error {
					callCounter++
					return nil
				},
				sentryClient,
				nil,
			)
			err = fn(ctx)
		})
		It("Returns no error", func() {
			Expect(err).To(BeNil())
		})
		It("calls fn", func() {
			Expect(callCounter).To(Equal(1))
		})
		It("has not call capture", func() {
			Expect(sentryClient.CaptureErrorAndWaitCallCount()).To(Equal(0))
		})
	})
	Context("with error", func() {
		BeforeEach(func() {
			fn := run.SkipErrorsAndReport(
				func(ctx context.Context) error {
					callCounter++
					return errors.New("banana")
				},
				sentryClient,
				nil,
			)
			err = fn(ctx)
		})
		It("Returns no error", func() {
			Expect(err).To(BeNil())
		})
		It("calls fn", func() {
			Expect(callCounter).To(Equal(1))
		})
		It("calls capture", func() {
			Expect(sentryClient.CaptureErrorAndWaitCallCount()).To(Equal(1))
		})
	})
	Context("with context canceled error", func() {
		BeforeEach(func() {
			fn := run.SkipErrorsAndReport(
				func(ctx context.Context) error {
					callCounter++
					return context.Canceled
				},
				sentryClient,
				nil,
			)
			err = fn(ctx)
		})
		It("Returns no error", func() {
			Expect(err).To(BeNil())
		})
		It("calls fn", func() {
			Expect(callCounter).To(Equal(1))
		})
		It("has not call capture", func() {
			Expect(sentryClient.CaptureErrorAndWaitCallCount()).To(Equal(0))
		})
	})
	Context("with wrapped context canceled error", func() {
		BeforeEach(func() {
			fn := run.SkipErrorsAndReport(
				func(ctx context.Context) error {
					callCounter++
					return errors.Wrap(context.Canceled, "wrapped")
				},
				sentryClient,
				nil,
			)
			err = fn(ctx)
		})
		It("Returns no error", func() {
			Expect(err).To(BeNil())
		})
		It("calls fn", func() {
			Expect(callCounter).To(Equal(1))
		})
		It("has not call capture", func() {
			Expect(sentryClient.CaptureErrorAndWaitCallCount()).To(Equal(0))
		})
	})
})
